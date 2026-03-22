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
	"os"
	"path/filepath"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/hook"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// withHookQuota rolls back a reserved hook slot when the inner call returns !ok (and no transport error).
func (e *commandExecutor) withHookQuota(sess *session.Session, run func() (*proto.ExecuteResponse, error)) (*proto.ExecuteResponse, error) {
	var success bool
	defer func() {
		if !success && e.quota != nil {
			e.quota.RemoveHook(sess.ID)
		}
	}()
	resp, err := run()
	if err != nil {
		return resp, err
	}
	if resp.GetOk() {
		success = true
	}
	return resp, err
}

// parseHookSourceArgs returns inline source, file path, optional BPF program name, optional hit limit.
func parseHookSourceArgs(args []string) (source, file, program string, limit int, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file", "-f":
			if i+1 >= len(args) {
				return "", "", "", 0, fmt.Errorf("--file requires value")
			}
			file = args[i+1]
			i++
		case "--source":
			if i+1 >= len(args) {
				return "", "", "", 0, fmt.Errorf("--source requires value")
			}
			source = args[i+1]
			i++
		case "--program", "-P":
			if i+1 >= len(args) {
				return "", "", "", 0, fmt.Errorf("--program requires value")
			}
			program = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", "", 0, fmt.Errorf("--limit requires value")
			}
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil || n < 0 {
				return "", "", "", 0, fmt.Errorf("--limit must be a non-negative integer")
			}
			limit = n
			i++
		}
	}
	if file != "" && source != "" {
		return "", "", "", 0, fmt.Errorf("cannot use both --file and --source")
	}
	if file == "" && source == "" {
		return "", "", "", 0, fmt.Errorf("missing --file or --source")
	}
	return source, file, program, limit, nil
}

func (e *commandExecutor) executeHookAttach(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	return e.withHookQuota(sess, func() (*proto.ExecuteResponse, error) {
		inline, file, program, limit, err := parseHookSourceArgs(args)
		if err != nil {
			return errResponse("hook attach: " + err.Error()), nil
		}
		var src string
		if file != "" {
			path := filepath.Clean(file)
			if !filepath.IsAbs(path) {
				return errResponse("hook attach: --file path must be absolute"), nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return errResponse("hook attach: read file: " + err.Error()), nil
			}
			if len(data) > hook.MaxRawSourceLen {
				return errResponse(fmt.Sprintf("hook attach: file larger than %d bytes", hook.MaxRawSourceLen)), nil
			}
			src = string(data)
		} else {
			src = inline
		}
		r := e.compileAttachFromSource(ctx, sess, src, program, limit, "hook attach")
		if !r.GetOk() {
			return errResponse("hook attach: " + r.GetErrorMessage()), nil
		}
		return &proto.ExecuteResponse{
			Ok:     true,
			Output: "hook set at " + r.GetProbePoint() + " (" + r.GetHookId() + ")",
			Result: &proto.ExecuteResponse_Hook{
				Hook: &proto.HookResult{HookId: r.GetHookId(), ProbePoint: r.GetProbePoint(), Compiled: true},
			},
		}, nil
	})
}

// executeHookList returns all hooks.
func (*commandExecutor) executeHookList(_ context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	list := sess.ListHooks()
	var lines []string
	for _, h := range list {
		line := fmt.Sprintf("%s  %s", h.ID, h.ProbePoint)
		if h.Note != "" {
			line += fmt.Sprintf("  note=%s", h.Note)
		}
		lines = append(lines, line)
	}
	output := "hooks:\n"
	if len(lines) == 0 {
		output += "  (none)\n"
	} else {
		output += strings.Join(lines, "\n") + "\n"
	}
	return &proto.ExecuteResponse{Ok: true, Output: output}, nil
}

// executeHook dispatches hook attach/list/delete.
func (e *commandExecutor) executeHook(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("hook: usage hook attach|list|delete ..."), nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		return errResponse("hook add is removed; use hook attach --source '...' or --file /abs/path.c [--program name]"), nil
	case "attach":
		return e.executeHookAttach(ctx, sess, args[1:])
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
