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

package server

import (
	"sync"

	"github.com/tomatopunk/phantom/lib/agent/session"
)

// SessionQuota limits resources per session (breakpoints, hooks).
type SessionQuota struct {
	mu       sync.Mutex
	bySess   map[string]*quotaState
	MaxBreak int
	MaxHooks int
}

type quotaState struct {
	breakpoints int
	hooks       int
}

// NewSessionQuota returns a quota enforcer with the given limits (0 = no limit).
func NewSessionQuota(maxBreak, maxHooks int) *SessionQuota {
	return &SessionQuota{
		bySess:   make(map[string]*quotaState),
		MaxBreak: maxBreak,
		MaxHooks: maxHooks,
	}
}

// AllowBreak returns true if the session can add another breakpoint.
func (q *SessionQuota) AllowBreak(sessionID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.get(sessionID)
	if q.MaxBreak > 0 && s.breakpoints >= q.MaxBreak {
		return false
	}
	s.breakpoints++
	return true
}

// AllowHook returns true if the session can add another hook.
func (q *SessionQuota) AllowHook(sessionID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.get(sessionID)
	if q.MaxHooks > 0 && s.hooks >= q.MaxHooks {
		return false
	}
	s.hooks++
	return true
}

func (q *SessionQuota) get(sessionID string) *quotaState {
	if s, ok := q.bySess[sessionID]; ok {
		return s
	}
	s := &quotaState{}
	q.bySess[sessionID] = s
	return s
}

// RemoveBreak decrements breakpoint count for the session.
func (q *SessionQuota) RemoveBreak(sessionID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if s, ok := q.bySess[sessionID]; ok && s.breakpoints > 0 {
		s.breakpoints--
	}
}

// RemoveHook decrements hook count for the session (e.g. after delete or failed compile).
func (q *SessionQuota) RemoveHook(sessionID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if s, ok := q.bySess[sessionID]; ok && s.hooks > 0 {
		s.hooks--
	}
}

// RemoveSession drops all quota state for the session.
func (q *SessionQuota) RemoveSession(sessionID string) {
	q.mu.Lock()
	delete(q.bySess, sessionID)
	q.mu.Unlock()
}

type quotaSessionSink struct {
	q *SessionQuota
}

// QuotaSessionSink returns a session.SessionQuotaSink that decrements hook/break counts on detach.
// Returns nil when q is nil.
func QuotaSessionSink(q *SessionQuota) session.SessionQuotaSink {
	if q == nil {
		return nil
	}
	return &quotaSessionSink{q: q}
}

func (s *quotaSessionSink) ReleaseHookSlot(sessionID string) { s.q.RemoveHook(sessionID) }

func (s *quotaSessionSink) ReleaseBreakSlot(sessionID string) { s.q.RemoveBreak(sessionID) }
