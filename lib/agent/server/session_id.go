package server

import (
	"crypto/rand"
	"encoding/hex"
)

const sessionIDBytes = 8

// generateSessionID returns a new random session id (e.g. for Connect when client omits it).
func generateSessionID() string {
	b := make([]byte, sessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "default"
	}
	return hex.EncodeToString(b)
}
