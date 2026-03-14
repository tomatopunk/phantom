package server

import (
	"testing"
)

func TestRateLimiterAllow(t *testing.T) {
	r := NewRateLimiter(10, 2)
	if !r.Allow("s1") {
		t.Error("first allow should succeed")
	}
	if !r.Allow("s1") {
		t.Error("second allow should succeed")
	}
	// burst exhausted; may or may not allow depending on time
	r.RemoveSession("s1")
	if !r.Allow("s1") {
		t.Error("after remove session allow should succeed")
	}
}
