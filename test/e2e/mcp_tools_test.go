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

// Package e2e: MCP tools coverage (set_breakpoint, run_command, list_sessions, list_breakpoints, list_hooks, hook attach path).
package e2e

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/mcp"
	"github.com/tomatopunk/phantom/lib/agent/server"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/grpc"
)

func startDebuggerServerWithBackend(t *testing.T) (backend mcp.Backend, cleanup func()) {
	t.Helper()
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mgr := session.NewManager("", nil)
	ds := server.NewDebuggerServer(mgr)
	srv := grpc.NewServer()
	proto.RegisterDebuggerServiceServer(srv, ds)
	go func() { _ = srv.Serve(lis) }()
	backend = server.NewMCPServerBackend(ds)
	cleanup = func() { srv.GracefulStop(); _ = lis.Close() }
	return backend, cleanup
}

// TestMCPConnectAndListSessions tests Connect and list_sessions via backend.
func TestMCPConnectAndListSessions(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()

	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if sid == "" {
		t.Error("Connect: expected non-empty session id")
	}
	ids, err := backend.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(ids) < 1 {
		t.Errorf("ListSessions: want at least 1, got %v", ids)
	}
}

// TestMCPSetBreakpoint runs set_breakpoint via backend; bare-symbol break is obsolete without bpf.
func TestMCPSetBreakpoint(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := backend.Execute(ctx, sid, "break tcp_sendmsg")
	if err != nil {
		t.Fatalf("Execute break: %v", err)
	}
	if resp.GetOk() {
		t.Fatal("expected bare break to fail (obsolete syntax)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "obsolete") {
		t.Errorf("set_breakpoint: want obsolete error, got %q", resp.GetErrorMessage())
	}
}

// TestMCPRunCommand runs run_command via backend.
func TestMCPRunCommand(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := backend.Execute(ctx, sid, "help")
	if err != nil {
		t.Fatalf("Execute help: %v", err)
	}
	if !resp.GetOk() {
		t.Errorf("run_command help: ok=false, %s", resp.GetErrorMessage())
	}
	if resp.GetOutput() == "" {
		t.Error("run_command help: expected output")
	}
}

// TestMCPListBreakpoints tests list_breakpoints via backend.
func TestMCPListBreakpoints(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	text, err := backend.ListBreakpoints(ctx, sid)
	if err != nil {
		t.Fatalf("ListBreakpoints: %v", err)
	}
	// Empty or header line
	if !strings.Contains(text, "bp") && text != "" {
		t.Logf("ListBreakpoints: %q", text)
	}
}

// TestMCPListHooks tests list_hooks via backend.
func TestMCPListHooks(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	text, err := backend.ListHooks(ctx, sid)
	if err != nil {
		t.Fatalf("ListHooks: %v", err)
	}
	// Empty or header
	_ = text
}

// TestMCPAddCHookPath tests hook attach via Execute; expect error (no bpf include dir in test server).
func TestMCPAddCHookPath(t *testing.T) {
	backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := backend.Execute(ctx, sid, "hook attach --attach kprobe:do_sys_open --source 'int x=0;'")
	if err != nil {
		t.Fatalf("Execute hook attach: %v", err)
	}
	if resp.GetOk() {
		t.Error("hook attach: expected ok=false (no bpf include dir)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "include") && !strings.Contains(resp.GetErrorMessage(), "bpf") {
		t.Logf("hook attach error: %s", resp.GetErrorMessage())
	}
}
