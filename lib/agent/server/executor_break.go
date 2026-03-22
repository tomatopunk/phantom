// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"context"

	"github.com/tomatopunk/phantom/lib/agent/hook"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// parseBreakProgramArgs parses break/tbreak: --attach, --source|--file, optional --program, optional --limit.
func parseBreakProgramArgs(cmdPrefix string, args []string, isTemp bool) (attach, inlineSource, file, program string, limit int, errMsg string) {
	if isTemp {
		limit = 1
	} else {
		limit = 0
	}
	if len(args) == 1 && !strings.HasPrefix(args[0], "-") {
		return "", "", "", "", 0, cmdPrefix + ": obsolete syntax (bare symbol); use --attach and --source or --file (see help break)"
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--attach", "-a":
			if i+1 >= len(args) {
				return "", "", "", "", 0, cmdPrefix + ": --attach needs a value"
			}
			attach = strings.TrimSpace(args[i+1])
			i++
		case "--file", "-f":
			if i+1 >= len(args) {
				return "", "", "", "", 0, cmdPrefix + ": --file needs a value"
			}
			file = args[i+1]
			i++
		case "--source":
			if i+1 >= len(args) {
				return "", "", "", "", 0, cmdPrefix + ": --source needs a value"
			}
			inlineSource = args[i+1]
			i++
		case "--program", "-P":
			if i+1 >= len(args) {
				return "", "", "", "", 0, cmdPrefix + ": --program needs a value"
			}
			program = strings.TrimSpace(args[i+1])
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", "", "", 0, cmdPrefix + ": --limit needs a value"
			}
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil || n < 0 {
				return "", "", "", "", 0, cmdPrefix + ": --limit must be a non-negative integer"
			}
			limit = n
			i++
		default:
			return "", "", "", "", 0, cmdPrefix + ": unexpected argument " + args[i] +
				" (usage: " + cmdPrefix + " --attach <point> (--source <c> | --file /abs/path.c) [--program name] [--limit N])"
		}
	}
	if attach == "" {
		return "", "", "", "", 0, cmdPrefix + ": missing --attach (e.g. kprobe:do_sys_open)"
	}
	if file != "" && inlineSource != "" {
		return "", "", "", "", 0, cmdPrefix + ": cannot use both --file and --source"
	}
	if file == "" && inlineSource == "" {
		return "", "", "", "", 0, cmdPrefix + ": missing --file or --source"
	}
	return attach, inlineSource, file, program, limit, ""
}

func (e *commandExecutor) executeBreakOrTbreak(
	ctx context.Context, sess *session.Session, args []string, cmdPrefix string, isTemp bool,
) (*proto.ExecuteResponse, error) {
	attach, inline, file, program, limit, errMsg := parseBreakProgramArgs(cmdPrefix, args, isTemp)
	if errMsg != "" {
		return errResponse(errMsg), nil
	}
	var src string
	if file != "" {
		path := filepath.Clean(file)
		if !filepath.IsAbs(path) {
			return errResponse(cmdPrefix + ": --file path must be absolute"), nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return errResponse(cmdPrefix + ": read file: " + err.Error()), nil
		}
		if len(data) > hook.MaxRawSourceLen {
			return errResponse(fmt.Sprintf(cmdPrefix+": file larger than %d bytes", hook.MaxRawSourceLen)), nil
		}
		src = string(data)
	} else {
		src = inline
	}
	note := "break"
	if isTemp {
		note = "tbreak"
	}
	r := e.tryCompileAttachHook(ctx, sess, src, attach, program, limit, note)
	if !r.GetOk() {
		return errResponse(cmdPrefix + ": " + r.GetErrorMessage()), nil
	}
	bpID := sess.AddProgramBreakpoint(attach, isTemp, r.GetHookId(), src, program, limit)
	msg := "breakpoint set at "
	if isTemp {
		msg = "temporary breakpoint set at "
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: msg + attach + " (" + bpID + ") hook " + r.GetHookId(),
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{BreakpointId: bpID, Symbol: attach, Enabled: true},
		},
	}, nil
}

// reattachUserProgramBreak recompiles and attaches after disable for user eBPF breakpoints.
func (e *commandExecutor) reattachUserProgramBreak(ctx context.Context, sess *session.Session, bpID string, bp *session.BreakpointState) (*proto.ExecuteResponse, error) {
	if bp.Symbol == "" || !bp.UserProgramBreak || strings.TrimSpace(bp.UserBreakSource) == "" {
		return errResponse("enable: breakpoint has no saved source"), nil
	}
	limit := bp.HookEventLimit
	r := e.tryCompileAttachHook(ctx, sess, bp.UserBreakSource, bp.Symbol, bp.UserBreakProgram, limit, "break")
	if !r.GetOk() {
		return errResponse("enable: " + r.GetErrorMessage()), nil
	}
	if !sess.LinkBreakpointHook(bpID, r.GetHookId()) {
		_ = sess.RemoveHook(r.GetHookId())
		return errResponse("enable: lost breakpoint " + bpID), nil
	}
	return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + bpID + " enabled"}, nil
}
