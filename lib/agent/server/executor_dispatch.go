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
	"break":  (*commandExecutor).executeBreak,
	"tbreak": (*commandExecutor).executeTbreak,
	"print":  (*commandExecutor).executePrint,
	"trace":  (*commandExecutor).executeTrace,
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
