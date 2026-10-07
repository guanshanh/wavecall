package sfu

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
)

// Peer represents a single participant's WebRTC connection to the SFU.
//
// Negotiation is always server-initiated: the initial offer already carries
// every track published in the room at join time, and each new sender triggers
// a fresh offer. The client only ever answers. This keeps the signaling state
// machine single-direction and avoids perfect-negotiation complexity.
type Peer struct {
	mu     sync.Mutex
	userID string
	pc     *webrtc.PeerConnection

	started        bool
	awaitingAnswer bool
	pendingTracks  []*webrtc.TrackLocalStaticRTP

	remoteDescSet           bool
	pendingRemoteCandidates []webrtc.ICECandidateInit
	closed                  bool

	onLocalOffer     func(sdp string)
	onLocalCandidate func(c webrtc.ICECandidateInit)
	onInboundTrack   func(track *webrtc.TrackRemote)
}

// NewPeer creates a new SFU Peer with a PeerConnection.
// The callbacks deliver the server's SDP offers and local ICE candidates
// to the client via the signaling channel.
// api may be nil to use pion's default API (no SettingEngine customization).
func NewPeer(
	userID string,
	iceServers []webrtc.ICEServer,
	api *webrtc.API,
	onLocalOffer func(sdp string),
	onLocalCandidate func(c webrtc.ICECandidateInit),
) (*Peer, error) {
	pcCfg := webrtc.Configuration{ICEServers: iceServers}
	var pc *webrtc.PeerConnection
	var err error
	if api != nil {
		pc, err = api.NewPeerConnection(pcCfg)
	} else {
		pc, err = webrtc.NewPeerConnection(pcCfg)
	}
	if err != nil {
		return nil, err
	}

	p := &Peer{
		userID:           userID,
		pc:               pc,
		onLocalOffer:     onLocalOffer,
		onLocalCandidate: onLocalCandidate,
	}

	// Forward local ICE candidates to the client (Trickle ICE), including
	// sdpMid / sdpMLineIndex — browsers reject candidates that omit both.
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		if p.onLocalCandidate != nil {
			p.onLocalCandidate(c.ToJSON())
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("peer connection state changed", "userId", userID, "state", state.String())
	})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		slog.Info("ice connection state changed", "userId", userID, "state", state.String())
	})

	// Surface inbound tracks to the router (wired via OnInboundTrack).
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if p.onInboundTrack != nil {
			p.onInboundTrack(track)
		}
	})

	return p, nil
}

// OnInboundTrack registers the callback invoked when the client publishes
// an audio track. Must be set before Start.
func (p *Peer) OnInboundTrack(fn func(track *webrtc.TrackRemote)) {
	p.onInboundTrack = fn
}

// Start performs the initial negotiation:
//  1. a recvonly audio transceiver for the client's microphone (created first
//     so the client's addTrack pairs with it, not with forwarded-track m-lines)
//  2. every track queued before start (already-published room tracks)
//  3. the first offer
func (p *Peer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("peer %s closed", p.userID)
	}

	if _, err := p.pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	}); err != nil {
		return fmt.Errorf("add recvonly transceiver: %w", err)
	}

	for _, t := range p.pendingTracks {
		if _, err := p.pc.AddTrack(t); err != nil {
			return fmt.Errorf("add pending track: %w", err)
		}
	}
	p.pendingTracks = nil
	p.started = true

	return p.negotiateLocked()
}

// AddOutboundTrack queues or attaches a forwarded track to this peer.
// Before Start: queued and included in the initial offer.
// While an offer is awaiting the client's answer: queued and flushed on answer.
// Otherwise: attached immediately and triggers a renegotiation offer.
func (p *Peer) AddOutboundTrack(track *webrtc.TrackLocalStaticRTP) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("peer %s closed", p.userID)
	}

	if !p.started || p.awaitingAnswer {
		p.pendingTracks = append(p.pendingTracks, track)
		return nil
	}

	if _, err := p.pc.AddTrack(track); err != nil {
		return fmt.Errorf("add outbound track: %w", err)
	}
	return p.negotiateLocked()
}

// HandleRemoteAnswer applies the client's answer and flushes any tracks
// queued while the offer was outstanding (queueing another offer if needed).
func (p *Peer) HandleRemoteAnswer(sdp string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("peer %s closed", p.userID)
	}

	if err := p.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}); err != nil {
		return fmt.Errorf("set remote answer: %w", err)
	}

	p.remoteDescSet = true
	p.awaitingAnswer = false
	p.flushRemoteCandidatesLocked()

	if len(p.pendingTracks) == 0 {
		return nil
	}
	pending := p.pendingTracks
	p.pendingTracks = nil
	for _, t := range pending {
		if _, err := p.pc.AddTrack(t); err != nil {
			return fmt.Errorf("flush pending track: %w", err)
		}
	}
	return p.negotiateLocked()
}

// HandleRemoteCandidate adds a client ICE candidate, buffering it until the
// remote description is set (candidates may trickle in before the answer).
func (p *Peer) HandleRemoteCandidate(candidate webrtc.ICECandidateInit) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	if !p.remoteDescSet {
		p.pendingRemoteCandidates = append(p.pendingRemoteCandidates, candidate)
		return nil
	}
	return p.pc.AddICECandidate(candidate)
}

// UserID returns this peer's user ID.
func (p *Peer) UserID() string {
	return p.userID
}

// negotiateLocked creates and sends an offer. Callers must hold p.mu;
// the offer callback is invoked from its own goroutine to avoid holding
// the lock through the signaling channel write.
func (p *Peer) negotiateLocked() error {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local offer: %w", err)
	}
	p.awaitingAnswer = true

	onLocalOffer := p.onLocalOffer
	sdp := offer.SDP
	go onLocalOffer(sdp)
	return nil
}

// flushRemoteCandidatesLocked drains candidates buffered before the answer.
func (p *Peer) flushRemoteCandidatesLocked() {
	if len(p.pendingRemoteCandidates) == 0 {
		return
	}
	pending := p.pendingRemoteCandidates
	p.pendingRemoteCandidates = nil
	for _, c := range pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			slog.Warn("failed to add buffered ICE candidate", "userId", p.userID, "err", err)
		}
	}
}

// Close shuts down the PeerConnection and releases resources.
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true
	return p.pc.Close()
}
