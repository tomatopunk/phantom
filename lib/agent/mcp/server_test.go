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

package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/proto"
)

type fakeMCPBackend struct {
	executeFn           func(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error)
	compileAndAttachFn  func(ctx context.Context, sessionID, source, attach, programName string) (*proto.CompileAndAttachResponse, error)
	listTracepointsFn   func(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
	listKprobeSymbolsFn func(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
}

func (f *fakeMCPBackend) Connect(ctx context.Context, sessionID string) (string, error) {
	return sessionID, nil
}

func (f *fakeMCPBackend) Execute(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error) {
	if f.executeFn != nil {
		return f.executeFn(ctx, sessionID, commandLine)
	}
	return &proto.ExecuteResponse{Ok: true, Output: "ok"}, nil
}

func (*fakeMCPBackend) ListSessions(context.Context) ([]string, error) {
	return nil, nil
}

func (*fakeMCPBackend) ListBreakpoints(context.Context, string) (string, error) {
	return "", nil
}

func (*fakeMCPBackend) ListHooks(context.Context, string) (string, error) {
	return "", nil
}

func (f *fakeMCPBackend) CompileAndAttach(
	ctx context.Context, sessionID, source, attach, programName string,
) (*proto.CompileAndAttachResponse, error) {
	if f.compileAndAttachFn != nil {
		return f.compileAndAttachFn(ctx, sessionID, source, attach, programName)
	}
	return &proto.CompileAndAttachResponse{Ok: true, HookId: "h1", AttachPoint: attach}, nil
}

func (f *fakeMCPBackend) ListTracepoints(ctx context.Context, prefix string, maxEntries uint32) ([]string, error) {
	if f.listTracepointsFn != nil {
		return f.listTracepointsFn(ctx, prefix, maxEntries)
	}
	return []string{prefix + ":tp"}, nil
}

func (f *fakeMCPBackend) ListKprobeSymbols(ctx context.Context, prefix string, maxEntries uint32) ([]string, error) {
	if f.listKprobeSymbolsFn != nil {
		return f.listKprobeSymbolsFn(ctx, prefix, maxEntries)
	}
	return []string{prefix + "_sym"}, nil
}

func TestRunCommandToolFailsLikeSetBreakpoint(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "break: nope"}, nil
		},
	})
	_, err := s.runTool(context.Background(), "run_command", map[string]any{
		"session_id":    "s1",
		"command_line":  "break foo",
	})
	if err == nil {
		t.Fatal("run_command: want error when Execute returns ok=false")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("run_command: want agent message, got %v", err)
	}
}

func TestSetBreakpointToolPropagatesExecuteError(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "missing symbol"}, nil
		},
	})
	_, err := s.runTool(context.Background(), "set_breakpoint", map[string]any{
		"session_id": "s1",
		"symbol":     "x",
	})
	if err == nil || !strings.Contains(err.Error(), "missing symbol") {
		t.Fatalf("set_breakpoint: want missing symbol error, got %v", err)
	}
}

func TestCompileAndAttachToolSuccessJSON(t *testing.T) {
	s := NewServer(&fakeMCPBackend{})
	out, err := s.runTool(context.Background(), "compile_and_attach", map[string]any{
		"session_id": "s1",
		"source":     "int x;",
		"attach":     "kprobe:foo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"ok":true`) || !strings.Contains(out, "h1") {
		t.Fatalf("want JSON with ok and hook id, got %q", out)
	}
}

func TestCompileAndAttachToolLogicalError(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		compileAndAttachFn: func(_ context.Context, _, _, _, _ string) (*proto.CompileAndAttachResponse, error) {
			return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "compile bad"}, nil
		},
	})
	_, err := s.runTool(context.Background(), "compile_and_attach", map[string]any{
		"session_id": "s1",
		"source":     "bad",
		"attach":     "kprobe:x",
	})
	if err == nil || !strings.Contains(err.Error(), "compile bad") {
		t.Fatalf("want compile error, got %v", err)
	}
}

func TestListTracepointsTool(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		listTracepointsFn: func(_ context.Context, prefix string, max uint32) ([]string, error) {
			if max != 100 {
				t.Fatalf("max_entries: want 100 got %d", max)
			}
			return []string{prefix + "a", prefix + "b"}, nil
		},
	})
	out, err := s.runTool(context.Background(), "list_tracepoints", map[string]any{
		"prefix":       "sched",
		"max_entries":  float64(100),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "scheda\nschedb" {
		t.Fatalf("got %q", out)
	}
}

func TestListKprobeSymbolsTool(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		listKprobeSymbolsFn: func(_ context.Context, prefix string, _ uint32) ([]string, error) {
			return []string{prefix + "open"}, nil
		},
	})
	out, err := s.runTool(context.Background(), "list_kprobe_symbols", map[string]any{
		"prefix": "do_sys_",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "do_sys_open" {
		t.Fatalf("got %q", out)
	}
}

func TestExecuteCommandLineEmptyErrorMessage(t *testing.T) {
	b := &fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "  "}, nil
		},
	}
	_, err := ExecuteCommandLine(context.Background(), b, "s", "x")
	if err == nil || !strings.Contains(err.Error(), "command failed") {
		t.Fatalf("want generic failure, got %v", err)
	}
}
