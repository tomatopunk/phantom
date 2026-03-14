package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
)

// executeTbreak sets a one-shot breakpoint (same as break but IsTemp true).
func (e *commandExecutor) executeTbreak(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("tbreak: missing symbol"), nil
	}
	symbol := args[0]
	rt, err := sess.EnsureRuntime()
	if err != nil {
		return errResponse("tbreak: " + err.Error()), nil
	}
	if rt == nil {
		return errResponse("tbreak: no kprobe object path configured"), nil
	}
	detach, err := rt.AttachKprobe(symbol)
	if err != nil {
		return errResponse("tbreak: " + err.Error()), nil
	}
	id := sess.AddBreakpoint(symbol, detach, true)
	sess.EnsureEventPump()
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "temporary breakpoint set at " + symbol + " (" + id + ")",
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{BreakpointId: id, Symbol: symbol, Enabled: true},
		},
	}, nil
}

// executeDelete removes a breakpoint, trace, or watch by id.
func (e *commandExecutor) executeDelete(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
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
func (e *commandExecutor) executeDisable(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("disable: missing breakpoint id"), nil
	}
	id := args[0]
	if sess.DisableBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " disabled"}, nil
	}
	return errResponse("disable: no breakpoint number " + id), nil
}

// executeEnable re-enables a breakpoint (Phase 2: we only flip Enabled; re-attach in later iteration).
func (e *commandExecutor) executeEnable(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("enable: missing breakpoint id"), nil
	}
	id := args[0]
	if sess.EnableBreakpoint(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "breakpoint " + id + " enabled"}, nil
	}
	return errResponse("enable: no breakpoint number " + id), nil
}

// executeCondition sets a condition on a breakpoint.
func (e *commandExecutor) executeCondition(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 2 {
		return errResponse("condition: usage condition <bp_id> <expr>"), nil
	}
	id, expr := args[0], strings.Join(args[1:], " ")
	if sess.SetBreakpointCondition(id, expr) {
		return &proto.ExecuteResponse{Ok: true, Output: "condition set for " + id}, nil
	}
	return errResponse("condition: no breakpoint number " + id), nil
}

// executeInfo dispatches to info break, trace, watch, or session.
func (e *commandExecutor) executeInfo(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("info: usage info break|trace|watch|session"), nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "break", "breakpoints", "b":
		return e.executeInfoBreak(ctx, sess)
	case "trace", "traces", "t":
		return e.executeInfoTrace(ctx, sess)
	case "watch", "watches", "w":
		return e.executeInfoWatch(ctx, sess)
	case "session", "sess":
		return e.executeInfoSession(ctx, sess)
	case "hook", "hooks":
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
func (e *commandExecutor) executeInfoBreak(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
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
		lines = append(lines, fmt.Sprintf("%s%s  %s  enabled=%s%s", bp.ID, tmp, bp.Symbol, en, cond))
	}
	output := "breakpoints:\n"
	if len(lines) == 0 {
		output += "  (none)\n"
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoTrace returns a listing of all traces.
func (e *commandExecutor) executeInfoTrace(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListTraces()
	var lines []string
	for _, tr := range list {
		lines = append(lines, fmt.Sprintf("%s  %s", tr.ID, strings.Join(tr.Expressions, ", ")))
	}
	output := "traces:\n"
	if len(lines) == 0 {
		output += "  (none)\n"
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoWatch returns a listing of all watches and their last value.
func (e *commandExecutor) executeInfoWatch(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
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
		output += "  (none)\n"
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeInfoSession returns session id and basic stats.
func (e *commandExecutor) executeInfoSession(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	bps := sess.ListBreakpoints()
	trs := sess.ListTraces()
	wchs := sess.ListWatches()
	output := fmt.Sprintf("session %s  breakpoints=%d  traces=%d  watches=%d\n", sess.ID, len(bps), len(trs), len(wchs))
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeList returns symbol info from kernel symbol table (best-effort); source/disasm not available.
func (e *commandExecutor) executeList(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	sym := ""
	if len(args) >= 1 {
		sym = args[0]
	}
	if sym == "" {
		return &proto.ExecuteResponse{Ok: true, Output: "list: specify a symbol (e.g. list do_sys_open)"}, nil
	}
	out, err := listSymbolKernel(sym)
	if err != nil {
		return &proto.ExecuteResponse{Ok: true, Output: "list " + sym + ": " + err.Error()}, nil
	}
	if out == "" {
		return &proto.ExecuteResponse{Ok: true, Output: "list " + sym + ": symbol not found in /proc/kallsyms"}, nil
	}
	return &proto.ExecuteResponse{Ok: true, Output: out}, nil
}

// executeBt returns kernel stack for the thread from the last event (best-effort).
func (e *commandExecutor) executeBt(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	ev := sess.GetLastEvent()
	output := readKernelStack(ev)
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeWatch registers a watch expression and returns its id; value changes are reported via STATE_CHANGE events.
func (e *commandExecutor) executeWatch(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("watch: missing expression"), nil
	}
	expr := strings.Join(args, " ")
	id := sess.AddWatch(expr)
	if sess.Runtime() != nil {
		sess.EnsureEventPump()
	}
	return &proto.ExecuteResponse{Ok: true, Output: "watch " + expr + " (" + id + ")"}, nil
}

// executeHelp returns short help for a command or all.
func (e *commandExecutor) executeHelp(ctx context.Context, args []string) (*proto.ExecuteResponse, error) {
	if len(args) >= 1 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case "break", "b":
			return &proto.ExecuteResponse{Ok: true, Output: "break <symbol>  set breakpoint at symbol"}, nil
		case "tbreak":
			return &proto.ExecuteResponse{Ok: true, Output: "tbreak <symbol>  temporary breakpoint"}, nil
		case "print", "p":
			return &proto.ExecuteResponse{Ok: true, Output: "print <expr>  print pid,tgid,cpu,event_type,timestamp_ns,probe_id"}, nil
		case "trace", "t":
			return &proto.ExecuteResponse{Ok: true, Output: "trace <expr...>  trace expressions"}, nil
		case "delete":
			return &proto.ExecuteResponse{Ok: true, Output: "delete <id>  delete breakpoint, trace, or watch"}, nil
		case "enable", "disable":
			return &proto.ExecuteResponse{Ok: true, Output: cmd + " <bp_id>  enable or disable breakpoint"}, nil
		case "condition":
			return &proto.ExecuteResponse{Ok: true, Output: "condition <bp_id> <expr>  set breakpoint condition"}, nil
		case "info":
			return &proto.ExecuteResponse{Ok: true, Output: "info break|trace|watch|session  list state"}, nil
		case "list":
			return &proto.ExecuteResponse{Ok: true, Output: "list [symbol]  list kernel symbol(s) from /proc/kallsyms"}, nil
		case "bt":
			return &proto.ExecuteResponse{Ok: true, Output: "bt  backtrace (kernel stack of last event thread)"}, nil
		case "watch":
			return &proto.ExecuteResponse{Ok: true, Output: "watch <expr>  emit event when expression value changes"}, nil
		case "continue", "c":
			return &proto.ExecuteResponse{Ok: true, Output: "continue  continue execution"}, nil
		default:
			return &proto.ExecuteResponse{Ok: true, Output: "help " + cmd + ": unknown command"}, nil
		}
	}
	output := `commands:
  break, b <symbol>     set breakpoint
  tbreak <symbol>       temporary breakpoint
  print, p <expr>       print expression
  trace, t <expr...>    trace expressions
  delete <id>           delete breakpoint/trace/watch
  enable <id>           enable breakpoint
  disable <id>          disable breakpoint
  condition <id> <expr> set condition
  info break|trace|watch|session
  list [symbol]         list kernel symbol(s)
  bt                    backtrace (kernel stack)
  watch <expr>          watch expression (emit on change)
  continue, c           continue
  help [cmd]
`
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}
