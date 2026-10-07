package room

import (
	"log/slog"
	"sync"

	"github.com/guanshanh/wavecall/internal/sfu"
)

// Room represents a voice chat room with connected peers.
type Room struct {
	mu       sync.RWMutex
	id       string
	password string
	peers    map[string]*PeerState // userID → state
	router   *sfu.Router
}

// PeerState holds a peer's metadata within a room.
type PeerState struct {
	UserID   string
	UserName string
	Muted    bool
}

// NewRoom creates a new voice chat room.
func NewRoom(id, password string) *Room {
	return &Room{
		id:       id,
		password: password,
		peers:    make(map[string]*PeerState),
		router:   sfu.NewRouter(),
	}
}

// ID returns the room's identifier.
func (r *Room) ID() string {
	return r.id
}

// CheckPassword validates the room password. Empty password means no auth.
func (r *Room) CheckPassword(password string) bool {
	return r.password == "" || r.password == password
}

// AddPeer adds a new peer to the room.
func (r *Room) AddPeer(userID, userName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.peers[userID] = &PeerState{
		UserID:   userID,
		UserName: userName,
		Muted:    false,
	}
	slog.Info("peer joined room", "room", r.id, "userId", userID, "userName", userName)
}

// RemovePeer removes a peer from the room.
func (r *Room) RemovePeer(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.peers, userID)
	r.router.RemovePeer(userID)
	slog.Info("peer left room", "room", r.id, "userId", userID)
}

// SetMuted updates a peer's mute state.
func (r *Room) SetMuted(userID string, muted bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if peer, ok := r.peers[userID]; ok {
		peer.Muted = muted
	}
}

// Peers returns a snapshot of all peers in the room.
func (r *Room) Peers() []PeerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PeerState, 0, len(r.peers))
	for _, p := range r.peers {
		result = append(result, *p)
	}
	return result
}

// PeerCount returns the number of peers in the room.
func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// Router returns the room's SFU media router.
func (r *Room) Router() *sfu.Router {
	return r.router
}

// IsEmpty returns true if the room has no peers.
func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers) == 0
}

// Close shuts down the room and all its resources.
func (r *Room) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.router.Close()
	r.peers = make(map[string]*PeerState)
	slog.Info("room closed", "room", r.id)
}
