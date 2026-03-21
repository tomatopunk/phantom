package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// replHandler runs a REPL verb; args are tokens after the verb.
type replHandler func(*commandExecutor, context.Context, *session.Session, []string) (*proto.ExecuteResponse, error)

func normalizeReplVerb(v string) string {
	switch v {
	case "b":
		return "break"
	case "p":
		return "print"
	case "t":
		return "trace"
	case "c":
		return "continue"
	default:
		return v
	}
}

// replVerbTable maps the canonical verb (after alias normalization) to its handler.
var replVerbTable = map[string]replHandler{
	"break":     (*commandExecutor).executeBreak,
	"tbreak":    (*commandExecutor).executeTbreak,
	"print":     (*commandExecutor).executePrint,
	"trace":     (*commandExecutor).executeTrace,
	"continue": func(e *commandExecutor, ctx context.Context, sess *session.Session, _ []string) (*proto.ExecuteResponse, error) {
		return e.executeContinue(ctx, sess)
	},
	"delete":    (*commandExecutor).executeDelete,
	"disable":   (*commandExecutor).executeDisable,
	"enable":    (*commandExecutor).executeEnable,
	"condition": (*commandExecutor).executeCondition,
	"info":      (*commandExecutor).executeInfo,
	"list":      (*commandExecutor).executeList,
	"hook":      (*commandExecutor).executeHook,
	"watch":     (*commandExecutor).executeWatch,
	"bt": func(e *commandExecutor, ctx context.Context, sess *session.Session, _ []string) (*proto.ExecuteResponse, error) {
		return e.executeBt(ctx, sess)
	},
	"help": func(e *commandExecutor, ctx context.Context, _ *session.Session, args []string) (*proto.ExecuteResponse, error) {
		return e.executeHelp(ctx, args)
	},
}

func (e *commandExecutor) execute(ctx context.Context, sess *session.Session, line string) (*proto.ExecuteResponse, error) {
	if line == "" {
		return &proto.ExecuteResponse{Ok: true, Output: ""}, nil
	}
	parts := splitCommandLine(line)
	if len(parts) == 0 {
		return &proto.ExecuteResponse{Ok: true, Output: ""}, nil
	}
	rawVerb := parts[0]
	verb := normalizeReplVerb(strings.ToLower(rawVerb))
	h, ok := replVerbTable[verb]
	if !ok {
		return errResponse(fmt.Sprintf("unknown command: %s", rawVerb)), nil
	}
	return h(e, ctx, sess, parts[1:])
}
