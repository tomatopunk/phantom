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
	"strconv"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/breaktpl"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

const (
	infoSubBreak         = "break"
	infoSubBreakTemplates = "break-templates"
	infoSubWatch         = "watch"
	infoSubHook          = "hook"
	infoSubSession       = "session"
	infoNoneLine         = "  (none)\n"
	cmdDelete            = "delete"
	cmdList              = "list"
)

// executeDelete removes a breakpoint or watch by id.
func (*commandExecutor) executeDelete(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("delete: missing breakpoint or watch id"), nil
	}
	id := args[0]
	if sess.RemoveBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " deleted"}, nil
	}
	if sess.RemoveWatch(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "watch " + id + " deleted"}, nil
	}
	return errResponse("delete: no breakpoint or watch " + id), nil
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
		resp, err := e.reattachTemplateBreak(ctx, sess, id, bp)
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

// executeInfo dispatches to info break, break-templates, watch, hook, or session.
func (e *commandExecutor) executeInfo(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("info: usage info break|break-templates|watch|hook|session"), nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case infoSubBreak, "breakpoints", "b":
		return e.executeInfoBreak(ctx, sess)
	case infoSubBreakTemplates, "templates":
		return e.executeInfoBreakTemplates(ctx, sess)
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
		if bp.KprobeHook && strings.TrimSpace(bp.KernelFilterExpr) != "" {
			kf = " filter=" + bp.KernelFilterExpr
		}
		lines = append(lines, fmt.Sprintf("%s%s  probe_id=%s  enabled=%s%s%s", bp.ID, tmp, bp.ProbeID, en, cond, kf))
	}
	output := "breakpoints:\n"
	if len(lines) == 0 {
		output += infoNoneLine
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoBreakTemplates lists built-in break probe_id entries.
func (*commandExecutor) executeInfoBreakTemplates(_ context.Context, _ *session.Session) (*proto.ExecuteResponse, error) {
	var lines []string
	for _, e := range breaktpl.List() {
		kind := "kprobe"
		if e.Kind == breaktpl.KindTracepoint {
			kind = "tracepoint"
		}
		lines = append(lines, fmt.Sprintf("%s  kind=%s  params=%s  default_arg_indices=%s",
			e.ProbeID, kind, strings.Join(e.Params, ","), joinIntSlice(e.DefaultArgIndices)))
	}
	out := "break-templates:\n"
	if len(lines) == 0 {
		out += infoNoneLine
	} else {
		out += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: out}, nil
}

// executeInfoWatch returns a listing of arg-column watches.
func (*commandExecutor) executeInfoWatch(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListWatches()
	var lines []string
	for _, w := range list {
		ix := "(defaults)"
		if len(w.ArgParamIndices) > 0 {
			ix = joinIntSlice(w.ArgParamIndices)
		}
		lines = append(lines, fmt.Sprintf("%s  probe_id=%s  param_indices=%s", w.ID, w.ProbeID, ix))
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
	wchs := sess.ListWatches()
	hks := sess.ListHooks()
	output := fmt.Sprintf("session %s  breakpoints=%d  watches=%d  hooks=%d\n", sess.ID, len(bps), len(wchs), len(hks))
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

func parseWatchArgs(args []string) (probeID string, indices []int, errMsg string) {
	var argsPart string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--sec":
			if i+1 >= len(args) {
				return "", nil, "watch: --sec needs value"
			}
			probeID = strings.TrimSpace(args[i+1])
			i++
		case "--args":
			if i+1 >= len(args) {
				return "", nil, "watch: --args needs value"
			}
			argsPart = strings.TrimSpace(args[i+1])
			i++
		default:
			return "", nil, "watch: unexpected token " + args[i]
		}
	}
	if probeID == "" {
		return "", nil, "watch: missing --sec <probe_id>"
	}
	if argsPart != "" {
		for _, p := range strings.Split(argsPart, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 {
				return "", nil, "watch: --args must be comma-separated non-negative integers"
			}
			indices = append(indices, n)
		}
	}
	return probeID, indices, ""
}

// executeWatch registers arg-column output when the probe break hits (requires an active break on probe_id).
func (*commandExecutor) executeWatch(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	probeID, indices, errMsg := parseWatchArgs(args)
	if errMsg != "" {
		return errResponse(errMsg), nil
	}
	entry, ok := breaktpl.Lookup(probeID)
	if !ok {
		return errResponse("watch: unknown probe_id (see info break-templates)"), nil
	}
	for _, ix := range indices {
		if ix >= len(entry.Params) {
			return errResponse(fmt.Sprintf("watch: param index %d out of range for %s", ix, probeID)), nil
		}
	}
	if !sess.HasActiveBreakForProbe(probeID) {
		return errResponse("watch: no enabled break for " + probeID + " (run break first)"), nil
	}
	id := sess.AddArgWatch(probeID, indices)
	var args32 []uint32
	for _, ix := range indices {
		args32 = append(args32, uint32(ix))
	}
	desc := "default columns"
	if len(indices) > 0 {
		desc = "param_indices=" + joinIntSlice(indices)
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: fmt.Sprintf("watch %s on %s (%s)", desc, probeID, id),
		Result: &proto.ExecuteResponse_Watch{
			Watch: &proto.WatchResult{WatchId: id, ProbeId: probeID, Args: args32},
		},
	}, nil
}

// executeHelp returns short help for a command or all.
func joinIntSlice(a []int) string {
	if len(a) == 0 {
		return ""
	}
	s := make([]string, 0, len(a))
	for _, v := range a {
		s = append(s, strconv.Itoa(v))
	}
	return strings.Join(s, ",")
}

func (*commandExecutor) executeHelp(_ context.Context, args []string) (*proto.ExecuteResponse, error) {
	if len(args) >= 1 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case infoSubBreak, "b":
			return &proto.ExecuteResponse{Ok: true, Output: "break <probe_id> [--filter <dsl>] [--limit N]  template kprobe/tracepoint only (see info break-templates)"}, nil
		case "tbreak", "t":
			return &proto.ExecuteResponse{Ok: true, Output: "tbreak <probe_id> [...]  same as break; default --limit 1"}, nil
		case "print", "p":
			return &proto.ExecuteResponse{Ok: true, Output: "print <expr>  evaluate once on last probe event (pid, arg0.., ret, ...)"}, nil
		case cmdDelete:
			return &proto.ExecuteResponse{Ok: true, Output: "delete <id>  breakpoint or watch (hooks: hook delete <id>)"}, nil
		case "enable", "disable":
			return &proto.ExecuteResponse{Ok: true, Output: cmd + " <bp_id>  enable or disable breakpoint"}, nil
		case "condition":
			return &proto.ExecuteResponse{Ok: true, Output: "condition <bp_id> <expr>  user-side filter on BREAK_HIT"}, nil
		case "info":
			return &proto.ExecuteResponse{Ok: true, Output: "info break|break-templates|watch|hook|session  list state"}, nil
		case cmdList:
			return &proto.ExecuteResponse{Ok: true, Output: "list [symbol]  list kernel symbol(s) from /proc/kallsyms"}, nil
		case "bt":
			return &proto.ExecuteResponse{Ok: true, Output: "bt  backtrace (kernel stack of last event thread)"}, nil
		case "watch":
			return &proto.ExecuteResponse{Ok: true, Output: "watch --sec <probe_id> [--args 0,1,2]  arg columns on break hit (param indices into catalog Params)"}, nil
		case "continue", "c":
			return &proto.ExecuteResponse{Ok: true, Output: "continue  continue execution"}, nil
		case "hook":
			return &proto.ExecuteResponse{Ok: true, Output: "hook attach --source|--file [--program] [--limit]  full C; probe_point from ELF SEC"}, nil
		default:
			return &proto.ExecuteResponse{Ok: true, Output: "help " + cmd + ": unknown command"}, nil
		}
	}
	output := `commands:
  Probes:
  break, b   <probe_id> [--filter <dsl>] [--limit N]   catalog template only
  tbreak, t  same; default --limit 1
  hook attach|list|delete   hook attach: --source|--file [--program] [--limit]

  On each probe event:
  print, p <expr>       evaluate once on last event
  watch --sec <probe_id> [--args i,j,...]   EVENT_TYPE_WATCH_ARG columns when break hits

  Breakpoint control:
  delete <id>           breakpoint or watch (hooks: hook delete <id>)
  enable|disable <id>   breakpoint only
  condition <id> <expr> user-side filter on BREAK_HIT

  Other:
  info break|break-templates|watch|hook|session
  list [symbol]         kallsyms / disasm
  bt                    kernel stack for last event
  continue, c
  help [cmd]
`
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}
