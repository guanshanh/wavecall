package sfu

import (
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
)

// Router manages track forwarding between peers in a room.
// When a peer publishes an audio track, the router forwards it to all other peers.
type Router struct {
	mu    sync.RWMutex
	peers map[string]*Peer // userID → Peer
}

// NewRouter creates a new media Router.
func NewRouter() *Router {
	return &Router{
		peers: make(map[string]*Peer),
	}
}

// AddPeer registers a peer and sets up track forwarding.
func (r *Router) AddPeer(peer *Peer) {
	r.mu.Lock()
	r.peers[peer.UserID()] = peer
	r.mu.Unlock()

	// When this peer sends a track, forward it to all other peers
	peer.PeerConnection().OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		slog.Info("track received", "userId", peer.UserID(), "kind", remoteTrack.Kind().String())
		r.forwardTrack(peer.UserID(), remoteTrack)
	})
}

// RemovePeer removes a peer and cleans up its resources.
func (r *Router) RemovePeer(userID string) {
	r.mu.Lock()
	peer, ok := r.peers[userID]
	delete(r.peers, userID)
	r.mu.Unlock()

	if ok {
		if err := peer.Close(); err != nil {
			slog.Error("failed to close peer", "userId", userID, "err", err)
		}
	}
}

// forwardTrack takes a remote track and forwards its RTP packets to all other peers.
func (r *Router) forwardTrack(senderID string, remoteTrack *webrtc.TrackRemote) {
	// TODO: implement RTP forwarding
	// 1. Create a local track for each receiving peer
	// 2. Add the local track to each receiver's PeerConnection
	// 3. Read RTP packets from remoteTrack in a goroutine
	// 4. Write each packet to all local tracks
	// 5. Handle peer removal (stop forwarding)
	slog.Info("forwarding track", "from", senderID, "codec", remoteTrack.Codec().MimeType)
}

// PeerCount returns the number of connected peers.
func (r *Router) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
}
