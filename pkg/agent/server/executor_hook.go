package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/hook"
	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
)

// executeHookAdd compiles C snippet (from --code or from --sec) and attaches at the given point.
func (e *commandExecutor) executeHookAdd(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	point, code, sec, err := parseHookAddArgs(args)
	if err != nil {
		return errResponse("hook add: " + err.Error()), nil
	}
	plan, err := e.planner.PlanHook(point, code, sec)
	if err != nil {
		return errResponse("hook add: " + err.Error()), nil
	}
	if e.hookIncludeDir == "" {
		return errResponse("hook add: no bpf include dir configured"), nil
	}
	snippet := plan.Code
	if plan.Sec != "" {
		snippet, err = hook.SecToSnippet(plan.Sec)
		if err != nil {
			return errResponse("hook add: " + err.Error()), nil
		}
	}
	cr, err := hook.Compile(ctx, snippet, plan.AttachPoint, e.hookIncludeDir)
	if err != nil {
		return errResponse("hook add: " + err.Error()), nil
	}
	detach, hookReader, err := hook.AttachKprobeFromObject(cr.ObjectPath, cr.Symbol, cr.Cleanup)
	if err != nil {
		if cr.Cleanup != nil {
			cr.Cleanup()
		}
		return errResponse("hook add: " + err.Error()), nil
	}
	id := sess.AddHook(plan.AttachPoint, detach, hookReader)
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "hook set at " + plan.AttachPoint + " (" + id + ")",
		Result: &proto.ExecuteResponse_Hook{
			Hook: &proto.HookResult{HookId: id, AttachPoint: plan.AttachPoint, Compiled: true},
		},
	}, nil
}

// parseHookAddArgs returns point, code, sec. Exactly one of code or sec must be set (mutually exclusive).
func parseHookAddArgs(args []string) (point, code, sec string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--point", "-p":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--point requires value")
			}
			point = args[i+1]
			i++
		case "--lang", "-l":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--lang requires value")
			}
			if !strings.EqualFold(args[i+1], "c") {
				return "", "", "", fmt.Errorf("only --lang c supported")
			}
			i++
		case "--code", "-c":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--code requires value")
			}
			code = args[i+1]
			i++
		case "--sec", "-s":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--sec requires value")
			}
			sec = args[i+1]
			i++
		}
	}
	if point == "" {
		return "", "", "", fmt.Errorf("missing --point (e.g. kprobe:do_sys_open)")
	}
	if code != "" && sec != "" {
		return "", "", "", fmt.Errorf("cannot use both --code and --sec (use one)")
	}
	if code == "" && sec == "" {
		return "", "", "", fmt.Errorf("missing --code or --sec")
	}
	return point, code, sec, nil
}

// executeHookList returns all hooks.
func (*commandExecutor) executeHookList(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListHooks()
	var lines []string
	for _, h := range list {
		lines = append(lines, fmt.Sprintf("%s  %s", h.ID, h.AttachPoint))
	}
	output := "hooks:\n"
	if len(lines) == 0 {
		output += "  (none)\n"
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeHook dispatches hook add/list/delete.
func (e *commandExecutor) executeHook(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("hook: usage hook add|list|delete ..."), nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		return e.executeHookAdd(ctx, sess, args[1:])
	case "list":
		return e.executeHookList(ctx, sess)
	case "delete", "del":
		return e.executeHookDelete(ctx, sess, args[1:])
	default:
		return errResponse("hook: unknown " + sub), nil
	}
}

// executeHookDelete removes a hook by id.
func (*commandExecutor) executeHookDelete(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("hook delete: missing hook id"), nil
	}
	id := args[0]
	if sess.RemoveHook(id) {
		return &proto.ExecuteResponse{Ok: true, Output: "hook " + id + " deleted"}, nil
	}
	return errResponse("hook delete: no hook " + id), nil
}
