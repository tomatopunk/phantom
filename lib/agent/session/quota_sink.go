// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

// SessionQuotaSink releases per-session quota when probes detach (optional).
// Wired from the agent server when a quota enforcer is configured.
type SessionQuotaSink interface {
	ReleaseHookSlot(sessionID string)
	ReleaseBreakSlot(sessionID string)
}
