package sfu

import (
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
)

// Router manages track forwarding between peers in the SFU.
// When a peer publishes an audio track, the router forwards its RTP packets
// to all other peers (pure relay, no mixing).
type Router struct {
	mu      sync.Mutex
	peers   map[string]*Peer      // userID → Peer
	streams map[string]*forwarder // senderID → forwarding state
}

// forwarder relays one sender's inbound RTP to a dynamic set of receivers.
type forwarder struct {
	mu       sync.Mutex
	senderID string
	remote   *webrtc.TrackRemote
	dests    map[string]*webrtc.TrackLocalStaticRTP // receiverID → local track
}

// newTrack creates a local track mirroring the sender's codec.
// Track ID and stream ID are both the sender's userID so the client can
// attribute the incoming track to the right peer via the MediaStream id.
func (f *forwarder) newTrack() (*webrtc.TrackLocalStaticRTP, error) {
	f.mu.Lock()
	codec := f.remote.Codec()
	f.mu.Unlock()

	return webrtc.NewTrackLocalStaticRTP(codec.RTPCodecCapability, f.senderID, f.senderID)
}

// NewRouter creates a new media Router.
func NewRouter() *Router {
	return &Router{
		peers:   make(map[string]*Peer),
		streams: make(map[string]*forwarder),
	}
}

// AddPeer registers a peer and wires its inbound track callback:
// when the peer publishes audio, the router fans it out to everyone else.
func (r *Router) AddPeer(peer *Peer) {
	r.mu.Lock()
	r.peers[peer.UserID()] = peer
	r.mu.Unlock()

	senderID := peer.UserID()
	peer.OnInboundTrack(func(remoteTrack *webrtc.TrackRemote) {
		slog.Info("track published", "sender", senderID, "codec", remoteTrack.Codec().MimeType)
		r.publish(senderID, remoteTrack)
	})
}

// AttachPublishedTracks gives the peer local tracks for every stream already
// published in the room. Must be called before Peer.Start so the tracks are
// included in the initial offer (no renegotiation for late joiners).
func (r *Router) AttachPublishedTracks(peer *Peer) {
	receiverID := peer.UserID()

	r.mu.Lock()
	defer r.mu.Unlock()

	for senderID, stream := range r.streams {
		if senderID == receiverID {
			continue
		}
		track, err := stream.newTrack()
		if err != nil {
			slog.Warn("failed to create forward track", "sender", senderID, "receiver", receiverID, "err", err)
			continue
		}
		if err := peer.AddOutboundTrack(track); err != nil {
			slog.Warn("failed to attach published track", "sender", senderID, "receiver", receiverID, "err", err)
			continue
		}
		stream.mu.Lock()
		stream.dests[receiverID] = track
		stream.mu.Unlock()
	}
}

// Peer returns the peer with the given userID, or nil.
func (r *Router) Peer(userID string) *Peer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.peers[userID]
}

// publish starts forwarding a sender's track and attaches it to all
// other peers (each attachment renegotiates that peer's connection).
func (r *Router) publish(senderID string, remote *webrtc.TrackRemote) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream := &forwarder{
		senderID: senderID,
		remote:   remote,
		dests:    make(map[string]*webrtc.TrackLocalStaticRTP),
	}
	r.streams[senderID] = stream

	for receiverID, peer := range r.peers {
		if receiverID == senderID {
			continue
		}
		track, err := stream.newTrack()
		if err != nil {
			slog.Warn("failed to create forward track", "sender", senderID, "receiver", receiverID, "err", err)
			continue
		}
		if err := peer.AddOutboundTrack(track); err != nil {
			slog.Warn("failed to attach forward track", "sender", senderID, "receiver", receiverID, "err", err)
			continue
		}
		stream.dests[receiverID] = track
	}

	go r.forwardLoop(stream)
}

// forwardLoop reads RTP packets from the sender and writes them to every
// receiver's local track until the sender disconnects.
func (r *Router) forwardLoop(stream *forwarder) {
	for {
		pkt, _, err := stream.remote.ReadRTP()
		if err != nil {
			slog.Info("forwarding stopped", "sender", stream.senderID, "err", err)
			return
		}

		stream.mu.Lock()
		for _, dest := range stream.dests {
			// WriteRTP rewrites SSRC/payload type per binding, so each
			// receiver gets its own copy of the header.
			p := *pkt
			if err := dest.WriteRTP(&p); err != nil {
				slog.Debug("rtp write failed", "sender", stream.senderID, "err", err)
			}
		}
		stream.mu.Unlock()
	}
}

// RemovePeer removes a peer, stops its forwarding, and detaches it from
// all other streams. Closing the peer's PeerConnection also unblocks its
// forwardLoop's ReadRTP.
func (r *Router) RemovePeer(userID string) {
	r.mu.Lock()
	peer, ok := r.peers[userID]
	delete(r.peers, userID)
	delete(r.streams, userID)
	for _, s := range r.streams {
		s.mu.Lock()
		delete(s.dests, userID)
		s.mu.Unlock()
	}
	r.mu.Unlock()

	if ok {
		// Closing the PeerConnection also unblocks the sender's forwardLoop.
		if err := peer.Close(); err != nil {
			slog.Error("failed to close peer", "userId", userID, "err", err)
		}
	}
}

// PeerCount returns the number of connected peers.
func (r *Router) PeerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.peers)
}

// Close shuts down all peers in the router.
func (r *Router) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, peer := range r.peers {
		if err := peer.Close(); err != nil {
			slog.Error("failed to close peer on router shutdown", "userId", id, "err", err)
		}
	}
	r.peers = make(map[string]*Peer)
	r.streams = make(map[string]*forwarder)
}
