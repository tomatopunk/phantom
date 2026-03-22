// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

const (
	infoSubBreak   = "break"
	infoSubTrace   = "trace"
	infoSubWatch   = "watch"
	infoSubHook    = "hook"
	infoSubSession = "session"
	infoNoneLine   = "  (none)\n"
	cmdDelete      = "delete"
	cmdList        = "list"
)

// executeDelete removes a breakpoint, trace, or watch by id.
func (*commandExecutor) executeDelete(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("delete: missing breakpoint, trace, or watch id"), nil
	}
	id := args[0]
	if sess.RemoveBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " deleted"}, nil
	}
	if sess.RemoveTrace(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "trace " + id + " deleted"}, nil
	}
	if sess.RemoveWatch(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "watch " + id + " deleted"}, nil
	}
	return errResponse("delete: no breakpoint, trace, or watch " + id), nil
}

// executeDisable disables a breakpoint (detaches).
func (*commandExecutor) executeDisable(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("disable: missing breakpoint id"), nil
	}
	id := args[0]
	if sess.DisableBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " disabled"}, nil
	}
	return errResponse("disable: no breakpoint number " + id), nil
}

// executeEnable re-enables a breakpoint; template break/tbreak hooks are recompiled and re-attached when needed.
func (e *commandExecutor) executeEnable(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("enable: missing breakpoint id"), nil
	}
	id := args[0]
	bp := sess.GetBreakpoint(id)
	if bp == nil {
		return errResponse("enable: no breakpoint number " + id), nil
	}
	if bp.KprobeHook {
		if bp.HookID != "" && bp.Enabled {
			return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " enabled"}, nil
		}
		var success bool
		if e.quota != nil && !e.quota.AllowHook(sess.ID) {
			return errResponse("quota: max hooks reached"), nil
		}
		defer func() {
			if !success && e.quota != nil {
				e.quota.RemoveHook(sess.ID)
			}
		}()
		resp, err := e.reattachUserProgramBreak(ctx, sess, id, bp)
		if err != nil {
			return resp, err
		}
		if resp.GetOk() {
			success = true
		}
		return resp, nil
	}
	if sess.EnableBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " enabled"}, nil
	}
	return errResponse("enable: no breakpoint number " + id), nil
}

// executeCondition sets a condition on a breakpoint.
func (*commandExecutor) executeCondition(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 2 {
		return errResponse("condition: usage condition <bp_id> <expr>"), nil
	}
	id, expr := args[0], strings.Join(args[1:], " ")
	if sess.SetBreakpointCondition(id, expr) {
		return &proto.ExecuteResponse{Ok: true, Output: "condition set for " + id}, nil
	}
	return errResponse("condition: no breakpoint number " + id), nil
}

// executeInfo dispatches to info break, trace, watch, hook, or session.
func (e *commandExecutor) executeInfo(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("info: usage info break|trace|watch|hook|session"), nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case infoSubBreak, "breakpoints", "b":
		return e.executeInfoBreak(ctx, sess)
	case infoSubTrace, "traces", "t":
		return e.executeInfoTrace(ctx, sess)
	case infoSubWatch, "watches", "w":
		return e.executeInfoWatch(ctx, sess)
	case infoSubSession, "sess":
		return e.executeInfoSession(ctx, sess)
	case infoSubHook, "hooks":
		return e.executeInfoHook(ctx, sess)
	default:
		return errResponse("info: unknown " + sub), nil
	}
}

// executeInfoHook returns a listing of all hooks.
func (e *commandExecutor) executeInfoHook(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	return e.executeHookList(ctx, sess)
}

// executeInfoBreak returns a listing of all breakpoints.
func (*commandExecutor) executeInfoBreak(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListBreakpoints()
	var lines []string
	for _, bp := range list {
		en := "y"
		if !bp.Enabled {
			en = "n"
		}
		tmp := ""
		if bp.IsTemp {
			tmp = " (temp)"
		}
		cond := ""
		if bp.Condition != "" {
			cond = " condition " + bp.Condition
		}
		kf := ""
		if bp.KprobeHook && !bp.UserProgramBreak && strings.TrimSpace(bp.KernelFilterExpr) != "" {
			kf = " kernel_sec=" + bp.KernelFilterExpr
		}
		userTag := ""
		if bp.UserProgramBreak {
			userTag = " user_ebpf=y"
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s  enabled=%s%s%s%s", bp.ID, tmp, bp.Symbol, en, cond, kf, userTag))
	}
	output := "breakpoints:\n"
	if len(lines) == 0 {
		output += infoNoneLine
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoTrace returns a listing of all traces.
func (*commandExecutor) executeInfoTrace(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListTraces()
	var lines []string
	for _, tr := range list {
		lines = append(lines, fmt.Sprintf("%s  %s", tr.ID, strings.Join(tr.Expressions, ", ")))
	}
	output := "traces:\n"
	if len(lines) == 0 {
		output += infoNoneLine
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoWatch returns a listing of all watches and their last value.
func (*commandExecutor) executeInfoWatch(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListWatches()
	var lines []string
	for _, w := range list {
		val := w.LastValue
		if !w.HasValue {
			val = "(not yet set)"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  last=%s", w.ID, w.Expression, val))
	}
	output := "watches:\n"
	if len(lines) == 0 {
		output += infoNoneLine
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoSession returns session id and basic stats.
func (*commandExecutor) executeInfoSession(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	bps := sess.ListBreakpoints()
	trs := sess.ListTraces()
	wchs := sess.ListWatches()
	hks := sess.ListHooks()
	output := fmt.Sprintf("session %s  breakpoints=%d  traces=%d  watches=%d  hooks=%d\n", sess.ID, len(bps), len(trs), len(wchs), len(hks))
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeList returns symbol info from kernel symbol table (best-effort); with VmlinuxPath set, appends disassembly.
func (e *commandExecutor) executeList(_ context.Context, _ *session.Session, args []string) (*proto.ExecuteResponse, error) {
	sym := ""
	if len(args) >= 1 {
		sym = args[0]
	}
	if sym == "" {
		return &proto.ExecuteResponse{Ok: true, Output: "list: specify a symbol (e.g. list do_sys_open)"}, nil
	}
	out, err := listSymbolKernelAndDisasm(sym, e.vmlinuxPath)
	if err != nil {
		return &proto.ExecuteResponse{Ok: true, Output: "list " + sym + ": " + err.Error()}, nil
	}
	if out == "" {
		return &proto.ExecuteResponse{Ok: true, Output: "list " + sym + ": symbol not found in /proc/kallsyms"}, nil
	}
	return &proto.ExecuteResponse{Ok: true, Output: out}, nil
}

// executeBt returns kernel stack for the thread from the last event (best-effort).
func (*commandExecutor) executeBt(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	ev := sess.GetLastEvent()
	output := readKernelStack(ev)
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeWatch registers a watch expression and returns its id; value changes are reported via STATE_CHANGE events.
func (*commandExecutor) executeWatch(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("watch: missing expression"), nil
	}
	expr := strings.Join(args, " ")
	id := sess.AddWatch(expr)
	_ = sess.EnsureEventPump()
	return &proto.ExecuteResponse{Ok: true, Output: "watch " + expr + " (" + id + ")"}, nil
}

// executeHelp returns short help for a command or all.
func (*commandExecutor) executeHelp(_ context.Context, args []string) (*proto.ExecuteResponse, error) {
	if len(args) >= 1 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case infoSubBreak, "b":
			return &proto.ExecuteResponse{Ok: true, Output: "break --attach <point> (--source <c> | --file /abs.c) [--program name] [--limit N]  user eBPF (CompileRaw); same flags as hook attach"}, nil
		case "tbreak":
			return &proto.ExecuteResponse{Ok: true, Output: "tbreak ...  same as break; default --limit 1 (temporary)"}, nil
		case "print", "p":
			return &proto.ExecuteResponse{Ok: true, Output: "print <expr>  evaluate once on last probe event (pid, arg0.., ret, ...)"}, nil
		case infoSubTrace, "t":
			return &proto.ExecuteResponse{Ok: true, Output: "trace <expr...>  after each break/hook event emit TRACE_SAMPLE with evaluated columns"}, nil
		case cmdDelete:
			return &proto.ExecuteResponse{Ok: true, Output: "delete <id>  delete breakpoint, trace, or watch (hooks: hook delete <id>)"}, nil
		case "enable", "disable":
			return &proto.ExecuteResponse{Ok: true, Output: cmd + " <bp_id>  enable or disable breakpoint"}, nil
		case "condition":
			return &proto.ExecuteResponse{Ok: true, Output: "condition <bp_id> <expr>  user-side filter on BREAK_HIT"}, nil
		case "info":
			return &proto.ExecuteResponse{Ok: true, Output: "info break|trace|watch|hook|session  list state"}, nil
		case cmdList:
			return &proto.ExecuteResponse{Ok: true, Output: "list [symbol]  list kernel symbol(s) from /proc/kallsyms"}, nil
		case "bt":
			return &proto.ExecuteResponse{Ok: true, Output: "bt  backtrace (kernel stack of last event thread)"}, nil
		case "watch":
			return &proto.ExecuteResponse{Ok: true, Output: "watch <expr>  emit STATE_CHANGE when expression string value changes vs last event"}, nil
		case "continue", "c":
			return &proto.ExecuteResponse{Ok: true, Output: "continue  continue execution"}, nil
		case "hook":
			return &proto.ExecuteResponse{Ok: true, Output: "hook attach ...  full eBPF C (same flags as break; no breakpoint id). hook list | hook delete <id>  (hook add removed)"}, nil
		default:
			return &proto.ExecuteResponse{Ok: true, Output: "help " + cmd + ": unknown command"}, nil
		}
	}
	output := `commands:
  Probes:
  break, b  --attach P (--source S | --file /abs.c) [--program N] [--limit L]  user eBPF (like hook attach)
  tbreak    same; default --limit 1
  hook attach|list|delete   hook attach: full C (--attach, --source|--file, [--program], [--limit]); see docs/command-spec.md

  On each probe event:
  print, p <expr>       evaluate once on last event
  trace, t <expr...>    TRACE_SAMPLE columns after each break/hook hit
  watch <expr>          STATE_CHANGE when value string changes

  Breakpoint control:
  delete <id>           breakpoint / trace / watch only (hooks: hook delete <id>)
  enable|disable <id>   breakpoint only
  condition <id> <expr> user-side filter on BREAK_HIT

  Other:
  info break|trace|watch|hook|session
  list [symbol]         kallsyms / disasm
  bt                    kernel stack for last event
  continue, c
  help [cmd]
`
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}
