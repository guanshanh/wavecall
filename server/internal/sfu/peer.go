package sfu

import (
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
)

// Peer represents a single participant's WebRTC connection in the SFU.
type Peer struct {
	mu             sync.Mutex
	userID         string
	peerConnection *webrtc.PeerConnection
	audioTrack     *webrtc.TrackLocalStaticRTP
}

// NewPeer creates a new SFU Peer with a PeerConnection.
func NewPeer(userID string, iceServers []webrtc.ICEServer) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})
	if err != nil {
		return nil, err
	}

	p := &Peer{
		userID:         userID,
		peerConnection: pc,
	}

	// Handle connection state changes
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("peer connection state changed", "userId", userID, "state", state.String())
	})

	return p, nil
}

// PeerConnection returns the underlying WebRTC PeerConnection.
func (p *Peer) PeerConnection() *webrtc.PeerConnection {
	return p.peerConnection
}

// UserID returns this peer's user ID.
func (p *Peer) UserID() string {
	return p.userID
}

// Close shuts down the PeerConnection and releases resources.
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peerConnection != nil {
		return p.peerConnection.Close()
	}
	return nil
}
