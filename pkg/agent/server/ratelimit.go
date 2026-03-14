package server

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter applies per-session request rate limits.
type RateLimiter struct {
	mu      sync.Mutex
	bySess  map[string]*rate.Limiter
	perSess rate.Limit
	burst   int
}

// NewRateLimiter creates a limiter allowing perSess requests per second per session, with burst size.
func NewRateLimiter(perSess float64, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		bySess:  make(map[string]*rate.Limiter),
		perSess: rate.Limit(perSess),
		burst:   burst,
	}
}

// Allow returns true if the session is within rate limit; false if rate limited.
func (r *RateLimiter) Allow(sessionID string) bool {
	r.mu.Lock()
	lim, ok := r.bySess[sessionID]
	if !ok {
		lim = rate.NewLimiter(r.perSess, r.burst)
		r.bySess[sessionID] = lim
	}
	r.mu.Unlock()
	return lim.Allow()
}

// Wait blocks until the session is within rate limit or ctx is cancelled.
func (r *RateLimiter) Wait(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	lim, ok := r.bySess[sessionID]
	if !ok {
		lim = rate.NewLimiter(r.perSess, r.burst)
		r.bySess[sessionID] = lim
	}
	r.mu.Unlock()
	return lim.Wait(ctx)
}

// RemoveSession drops the limiter for the session (call on CloseSession).
func (r *RateLimiter) RemoveSession(sessionID string) {
	r.mu.Lock()
	delete(r.bySess, sessionID)
	r.mu.Unlock()
}
