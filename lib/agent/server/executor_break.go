// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/breaktpl"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// parseTemplateBreakArgs parses break/tbreak: <probe_id> [--filter <dsl>] [--limit N] (tbreak defaults --limit 1).
func parseTemplateBreakArgs(cmdPrefix string, args []string, isTemp bool) (probeID, filter string, limit int, errMsg string) {
	if isTemp {
		limit = 1
	} else {
		limit = 0
	}
	if len(args) < 1 {
		return "", "", 0, cmdPrefix + ": missing probe_id (try info break-templates)"
	}
	if strings.HasPrefix(args[0], "-") {
		return "", "", 0, cmdPrefix + ": missing probe_id before options"
	}
	probeID = strings.TrimSpace(args[0])
	if probeID == "" {
		return "", "", 0, cmdPrefix + ": empty probe_id"
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--filter":
			if i+1 >= len(args) {
				return "", "", 0, cmdPrefix + ": --filter needs a value"
			}
			filter = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", 0, cmdPrefix + ": --limit needs a value"
			}
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil || n < 0 {
				return "", "", 0, cmdPrefix + ": --limit must be a non-negative integer"
			}
			limit = n
			i++
		default:
			return "", "", 0, cmdPrefix + ": unexpected argument " + args[i]
		}
	}
	return probeID, filter, limit, ""
}

func (e *commandExecutor) executeBreakOrTbreak(
	ctx context.Context, sess *session.Session, args []string, cmdPrefix string, isTemp bool,
) (*proto.ExecuteResponse, error) {
	probeID, filter, limit, errMsg := parseTemplateBreakArgs(cmdPrefix, args, isTemp)
	if errMsg != "" {
		return errResponse(errMsg), nil
	}
	entry, ok := breaktpl.Lookup(probeID)
	if !ok {
		return errResponse(cmdPrefix + ": unknown probe_id (see info break-templates)"), nil
	}
	src, err := breaktpl.GenerateC(entry, filter)
	if err != nil {
		return errResponse(cmdPrefix + ": " + err.Error()), nil
	}
	note := "break"
	if isTemp {
		note = "tbreak"
	}
	prog := breaktpl.ProgramName(entry.ProbeID)
	r := e.compileAttachFromSource(ctx, sess, src, prog, limit, note)
	if !r.GetOk() {
		return errResponse(cmdPrefix + ": " + r.GetErrorMessage()), nil
	}
	bpID := sess.AddTemplateBreakpoint(probeID, isTemp, r.GetHookId(), filter, limit)
	msg := "breakpoint set on "
	if isTemp {
		msg = "temporary breakpoint set on "
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: msg + probeID + " (" + bpID + ") hook " + r.GetHookId(),
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{BreakpointId: bpID, Symbol: probeID, Enabled: true},
		},
	}, nil
}

// reattachTemplateBreak recompiles and attaches after disable for template breakpoints.
func (e *commandExecutor) reattachTemplateBreak(ctx context.Context, sess *session.Session, bpID string, bp *session.BreakpointState) (*proto.ExecuteResponse, error) {
	if bp.ProbeID == "" {
		return errResponse("enable: breakpoint has no probe_id"), nil
	}
	entry, ok := breaktpl.Lookup(bp.ProbeID)
	if !ok {
		return errResponse("enable: unknown probe_id " + bp.ProbeID), nil
	}
	src, err := breaktpl.GenerateC(entry, bp.KernelFilterExpr)
	if err != nil {
		return errResponse("enable: " + err.Error()), nil
	}
	limit := bp.HookEventLimit
	prog := breaktpl.ProgramName(entry.ProbeID)
	r := e.compileAttachFromSource(ctx, sess, src, prog, limit, "break")
	if !r.GetOk() {
		return errResponse("enable: " + r.GetErrorMessage()), nil
	}
	if !sess.LinkBreakpointHook(bpID, r.GetHookId()) {
		_ = sess.RemoveHook(r.GetHookId())
		return errResponse("enable: lost breakpoint " + bpID), nil
	}
	return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + bpID + " enabled"}, nil
}
