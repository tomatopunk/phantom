package session

import (
	"context"
	"sync"
)

// Manager coordinates debug sessions and command execution.
type Manager struct {
	mu           sync.RWMutex
	byID         map[string]*Session
	kprobePath   string
}

// NewManager returns a session manager; kprobePath is used when creating new sessions for real eBPF load.
func NewManager(kprobePath string) *Manager {
	return &Manager{byID: make(map[string]*Session), kprobePath: kprobePath}
}

// GetOrCreate returns an existing session by id or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.byID[id]; ok {
		return s, nil
	}
	s := NewSession(id, m.kprobePath)
	m.byID[id] = s
	return s, nil
}

// Get returns the session for id or nil.
func (m *Manager) Get(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byID[id]
}

// Close removes the session and releases resources.
func (m *Manager) Close(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.byID[id]; ok {
		s.Stop()
		delete(m.byID, id)
	}
}

// List returns all session ids.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.byID))
	for id := range m.byID {
		ids = append(ids, id)
	}
	return ids
}
