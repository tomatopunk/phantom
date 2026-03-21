package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/cilium/ebpf/btf"
	"github.com/tomatopunk/phantom/lib/agent/expression"
	"github.com/tomatopunk/phantom/lib/agent/probe"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// commandExecutor parses command lines and drives session runtime + state.
type commandExecutor struct {
	hookIncludeDir string // path to bpf/include for C hook compile
	vmlinuxPath    string // optional: path to vmlinux for list disasm (Linux)
	btfSpec        *btf.Spec
	planner        *probe.Planner
	quota          *SessionQuota // optional: rollback hook slot on failed hook add; decrement on delete
}

func newCommandExecutor(hookIncludeDir, vmlinuxPath string, planner *probe.Planner, btfSpec *btf.Spec, quota *SessionQuota) *commandExecutor {
	if planner == nil {
		planner = probe.NewPlanner()
	}
	return &commandExecutor{hookIncludeDir: hookIncludeDir, vmlinuxPath: vmlinuxPath, btfSpec: btfSpec, planner: planner, quota: quota}
}

//nolint:gocyclo // dispatch switch is inherently high cyclomatic complexity
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
	return e.executeBreakOrTbreak(ctx, sess, args, "break", false)
}

func (e *commandExecutor) executeTbreak(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	return e.executeBreakOrTbreak(ctx, sess, args, "tbreak", true)
}

// executeBreakOrTbreak is shared logic for break and tbreak (dupl).
func (e *commandExecutor) executeBreakOrTbreak(
	ctx context.Context, sess *session.Session, args []string, cmdPrefix string, isTemp bool,
) (*proto.ExecuteResponse, error) {
	_ = ctx
	if len(args) < 1 {
		return errResponse(cmdPrefix + ": missing symbol"), nil
	}
	plan := e.planner.PlanBreak(args[0])
	rt, err := sess.EnsureRuntime()
	if err != nil {
		return errResponse(cmdPrefix + ": " + err.Error()), nil
	}
	if rt == nil {
		return errResponse(cmdPrefix + ": no kprobe object path configured"), nil
	}
	detach, err := rt.AttachKprobe(plan.Symbol)
	if err != nil {
		return errResponse(cmdPrefix + ": " + err.Error()), nil
	}
	id := sess.AddBreakpoint(plan.Symbol, detach, isTemp)
	sess.EnsureEventPump()
	msg := "breakpoint set at "
	if isTemp {
		msg = "temporary breakpoint set at "
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: msg + plan.Symbol + " (" + id + ")",
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{
				BreakpointId: id,
				Symbol:       plan.Symbol,
				Enabled:      true,
			},
		},
	}, nil
}

func (*commandExecutor) executePrint(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
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

func (e *commandExecutor) executeTrace(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("trace: missing expression(s)"), nil
	}
	plan := e.planner.PlanTrace(args)
	id := sess.AddTrace(plan.Expressions, nil)
	if sess.Runtime() != nil {
		sess.EnsureEventPump()
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "tracing " + strings.Join(plan.Expressions, ", ") + " (" + id + ")",
		Result: &proto.ExecuteResponse_Trace{
			Trace: &proto.TraceResult{TraceId: id, Expressions: plan.Expressions},
		},
	}, nil
}

func (*commandExecutor) executeContinue(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	_ = ctx
	_ = sess
	return &proto.ExecuteResponse{Ok: true, Output: "continue"}, nil
}

func splitCommandLine(line string) []string {
	return strings.Fields(line)
}

func errResponse(msg string) *proto.ExecuteResponse {
	return &proto.ExecuteResponse{Ok: false, ErrorMessage: msg}
}
