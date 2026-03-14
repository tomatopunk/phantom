package server

import "errors"

// Sentinel errors for client handling.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrRateLimited     = errors.New("rate limited")
	ErrQuotaExceeded   = errors.New("quota exceeded")
)
