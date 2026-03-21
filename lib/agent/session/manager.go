// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"sync"
)

// Manager coordinates debug sessions and command execution.
type Manager struct {
	mu         sync.RWMutex
	byID       map[string]*Session
	kprobePath string
}

// NewManager returns a session manager; kprobePath is used when creating new sessions for real eBPF load.
func NewManager(kprobePath string) *Manager {
	return &Manager{byID: make(map[string]*Session), kprobePath: kprobePath}
}

// GetOrCreate returns an existing session by id or creates a new one.
func (m *Manager) GetOrCreate(_ context.Context, id string) (*Session, error) {
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
