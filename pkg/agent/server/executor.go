package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/expression"
	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
)

// commandExecutor parses command lines and drives session runtime + state.
type commandExecutor struct {
	hookIncludeDir string // path to bpf/include for C hook compile
}

func newCommandExecutor(hookIncludeDir string) *commandExecutor {
	return &commandExecutor{hookIncludeDir: hookIncludeDir}
}

func (e *commandExecutor) execute(ctx context.Context, sess *session.Session, line string) (*proto.ExecuteResponse, error) {
	if line == "" {
		return &proto.ExecuteResponse{Ok: true, Output: ""}, nil
	}
	parts := splitCommandLine(line)
	verb := strings.ToLower(parts[0])
	switch verb {
	case "break", "b":
		return e.executeBreak(ctx, sess, parts[1:])
	case "tbreak":
		return e.executeTbreak(ctx, sess, parts[1:])
	case "print", "p":
		return e.executePrint(ctx, sess, parts[1:])
	case "trace", "t":
		return e.executeTrace(ctx, sess, parts[1:])
	case "continue", "c":
		return e.executeContinue(ctx, sess)
	case "delete":
		return e.executeDelete(ctx, sess, parts[1:])
	case "disable":
		return e.executeDisable(ctx, sess, parts[1:])
	case "enable":
		return e.executeEnable(ctx, sess, parts[1:])
	case "condition":
		return e.executeCondition(ctx, sess, parts[1:])
	case "info":
		return e.executeInfo(ctx, sess, parts[1:])
	case "list":
		return e.executeList(ctx, sess, parts[1:])
	case "bt":
		return e.executeBt(ctx, sess)
	case "watch":
		return e.executeWatch(ctx, sess, parts[1:])
	case "help":
		return e.executeHelp(ctx, parts[1:])
	case "hook":
		return e.executeHook(ctx, sess, parts[1:])
	default:
		return errResponse(fmt.Sprintf("unknown command: %s", verb)), nil
	}
}

func (e *commandExecutor) executeBreak(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("break: missing symbol"), nil
	}
	symbol := args[0]
	rt, err := sess.EnsureRuntime()
	if err != nil {
		return errResponse("break: " + err.Error()), nil
	}
	if rt == nil {
		return errResponse("break: no kprobe object path configured"), nil
	}
	detach, err := rt.AttachKprobe(symbol)
	if err != nil {
		return errResponse("break: " + err.Error()), nil
	}
	id := sess.AddBreakpoint(symbol, detach, false)
	sess.EnsureEventPump()
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "breakpoint set at " + symbol + " (" + id + ")",
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{
				BreakpointId: id,
				Symbol:       symbol,
				Enabled:      true,
			},
		},
	}, nil
}

func (e *commandExecutor) executePrint(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("print: missing expression"), nil
	}
	expr := strings.TrimSpace(args[0])
	ev := sess.GetLastEvent()
	value := expression.Evaluate(ev, expr)
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "$" + expr + " = " + value,
		Result: &proto.ExecuteResponse_Print{
			Print: &proto.PrintResult{Expression: expr, Value: value},
		},
	}, nil
}

func (e *commandExecutor) executeTrace(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("trace: missing expression(s)"), nil
	}
	exprs := args
	id := sess.AddTrace(exprs, nil)
	if sess.Runtime() != nil {
		sess.EnsureEventPump()
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "tracing " + strings.Join(exprs, ", ") + " (" + id + ")",
		Result: &proto.ExecuteResponse_Trace{
			Trace: &proto.TraceResult{TraceId: id, Expressions: exprs},
		},
	}, nil
}

func (e *commandExecutor) executeContinue(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	_ = ctx
	_ = sess
	return &proto.ExecuteResponse{Ok: true, Output: "continue"}, nil
}

func splitCommandLine(line string) []string {
	var parts []string
	for _, s := range strings.Fields(line) {
		parts = append(parts, s)
	}
	return parts
}

func errResponse(msg string) *proto.ExecuteResponse {
	return &proto.ExecuteResponse{Ok: false, ErrorMessage: msg}
}
