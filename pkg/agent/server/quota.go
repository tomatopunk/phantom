package server

import "sync"

// SessionQuota limits resources per session (breakpoints, traces, hooks).
type SessionQuota struct {
	mu         sync.Mutex
	bySess     map[string]*quotaState
	MaxBreak   int
	MaxTrace   int
	MaxHooks   int
}

type quotaState struct {
	breakpoints int
	traces      int
	hooks       int
}

// NewSessionQuota returns a quota enforcer with the given limits (0 = no limit).
func NewSessionQuota(maxBreak, maxTrace, maxHooks int) *SessionQuota {
	return &SessionQuota{
		bySess:   make(map[string]*quotaState),
		MaxBreak: maxBreak,
		MaxTrace: maxTrace,
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

// AllowTrace returns true if the session can add another trace.
func (q *SessionQuota) AllowTrace(sessionID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.get(sessionID)
	if q.MaxTrace > 0 && s.traces >= q.MaxTrace {
		return false
	}
	s.traces++
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

// RemoveSession drops all quota state for the session.
func (q *SessionQuota) RemoveSession(sessionID string) {
	q.mu.Lock()
	delete(q.bySess, sessionID)
	q.mu.Unlock()
}
