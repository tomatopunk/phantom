package server

import (
	"testing"
)

func TestSessionQuotaAllowBreak(t *testing.T) {
	q := NewSessionQuota(2, 0, 0)
	if !q.AllowBreak("s1") {
		t.Error("first break should be allowed")
	}
	if !q.AllowBreak("s1") {
		t.Error("second break should be allowed")
	}
	if q.AllowBreak("s1") {
		t.Error("third break should be denied")
	}
	q.RemoveBreak("s1")
	if !q.AllowBreak("s1") {
		t.Error("after remove one break should be allowed again")
	}
}
