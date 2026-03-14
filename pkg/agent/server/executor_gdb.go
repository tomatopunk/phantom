package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
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

// executeEnable re-enables a breakpoint (Phase 2: we only flip Enabled; re-attach in later iteration).
func (*commandExecutor) executeEnable(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
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

// executeInfo dispatches to info break, trace, watch, or session.
func (e *commandExecutor) executeInfo(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("info: usage info break|trace|watch|session"), nil
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
		lines = append(lines, fmt.Sprintf("%s%s  %s  enabled=%s%s", bp.ID, tmp, bp.Symbol, en, cond))
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
	output := fmt.Sprintf("session %s  breakpoints=%d  traces=%d  watches=%d\n", sess.ID, len(bps), len(trs), len(wchs))
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeList returns symbol info from kernel symbol table (best-effort); source/disasm not available.
func (*commandExecutor) executeList(_ context.Context, _ *session.Session, args []string) (*proto.ExecuteResponse, error) {
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
	if sess.Runtime() != nil {
		sess.EnsureEventPump()
	}
	return &proto.ExecuteResponse{Ok: true, Output: "watch " + expr + " (" + id + ")"}, nil
}

// executeHelp returns short help for a command or all.
func (*commandExecutor) executeHelp(_ context.Context, args []string) (*proto.ExecuteResponse, error) {
	if len(args) >= 1 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case infoSubBreak, "b":
			return &proto.ExecuteResponse{Ok: true, Output: "break <symbol>  set breakpoint at symbol"}, nil
		case "tbreak":
			return &proto.ExecuteResponse{Ok: true, Output: "tbreak <symbol>  temporary breakpoint"}, nil
		case "print", "p":
			return &proto.ExecuteResponse{Ok: true, Output: "print <expr>  print pid,tgid,cpu,event_type,timestamp_ns,probe_id"}, nil
		case infoSubTrace, "t":
			return &proto.ExecuteResponse{Ok: true, Output: "trace <expr...>  trace expressions"}, nil
		case cmdDelete:
			return &proto.ExecuteResponse{Ok: true, Output: "delete <id>  delete breakpoint, trace, or watch"}, nil
		case "enable", "disable":
			return &proto.ExecuteResponse{Ok: true, Output: cmd + " <bp_id>  enable or disable breakpoint"}, nil
		case "condition":
			return &proto.ExecuteResponse{Ok: true, Output: "condition <bp_id> <expr>  set breakpoint condition"}, nil
		case "info":
			return &proto.ExecuteResponse{Ok: true, Output: "info break|trace|watch|session  list state"}, nil
		case cmdList:
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
