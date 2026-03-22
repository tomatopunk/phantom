// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"log"
	"os"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
)

// Operational startup logs (listen addresses, etc.) use [phantom] and are always emitted from main / server / health.
// DebugLogf is for verbose per-request traces and only runs when PHANTOM_AGENT_DEBUG is set.

// AgentDebugEnabled reports whether PHANTOM_AGENT_DEBUG requests verbose agent logs (1/true/yes/on).
func AgentDebugEnabled() bool {
	v := strings.TrimSpace(os.Getenv("PHANTOM_AGENT_DEBUG"))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// DebugLogf writes to the standard logger when AgentDebugEnabled is true.
func DebugLogf(format string, args ...interface{}) {
	if !AgentDebugEnabled() {
		return
	}
	log.Printf("[phantom-debug] "+format, args...)
}

func truncateForLog(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

func logExecuteDebug(source, sessionID, line string, resp *proto.ExecuteResponse, execErr error) {
	if !AgentDebugEnabled() {
		return
	}
	cmd := truncateForLog(strings.TrimSpace(line), 500)
	if execErr != nil {
		log.Printf("[phantom-debug] %s session=%s cmd=%q transport_err=%v", source, sessionID, cmd, execErr)
		return
	}
	if resp == nil {
		log.Printf("[phantom-debug] %s session=%s cmd=%q resp=<nil>", source, sessionID, cmd)
		return
	}
	out := truncateForLog(resp.GetOutput(), 200)
	errMsg := resp.GetErrorMessage()
	log.Printf("[phantom-debug] %s session=%s ok=%v cmd=%q err_msg=%q out=%q",
		source, sessionID, resp.GetOk(), cmd, errMsg, out)
}
