package room

import (
	"fmt"
	"log/slog"
	"sync"
)

// Manager manages all active rooms.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room // roomID → Room
}

// NewManager creates a new room Manager.
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// GetOrCreate returns an existing room or creates a new one.
func (m *Manager) GetOrCreate(roomID, password string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	if r, ok := m.rooms[roomID]; ok {
		return r
	}

	r := NewRoom(roomID, password)
	m.rooms[roomID] = r
	slog.Info("room created", "roomId", roomID)
	return r
}

// Get returns a room by ID, or an error if not found.
func (m *Manager) Get(roomID string) (*Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, ok := m.rooms[roomID]
	if !ok {
		return nil, fmt.Errorf("room %s not found", roomID)
	}
	return r, nil
}

// Remove deletes a room. Called when the last peer leaves.
func (m *Manager) Remove(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if r, ok := m.rooms[roomID]; ok {
		r.Close()
		delete(m.rooms, roomID)
		slog.Info("room removed", "roomId", roomID)
	}
}

// CleanupEmpty removes all rooms with no peers.
func (m *Manager) CleanupEmpty() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, r := range m.rooms {
		if r.IsEmpty() {
			r.Close()
			delete(m.rooms, id)
			slog.Info("empty room cleaned up", "roomId", id)
		}
	}
}

// RoomCount returns the number of active rooms.
func (m *Manager) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}
