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

// Package e2e: Execute command matrix (in-process server, no kprobe required).
package e2e

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/server"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
	"google.golang.org/grpc"
)

func startInProcessServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = lis.Addr().String()
	mgr := session.NewManager("", nil)
	srv := grpc.NewServer()
	proto.RegisterDebuggerServiceServer(srv, server.NewDebuggerServer(mgr))
	go func() { _ = srv.Serve(lis) }()
	cleanup = func() { srv.GracefulStop(); _ = lis.Close() }
	return addr, cleanup
}

func connectClient(t *testing.T, addr string) (client *grpcclient.Client, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	client, err := grpcclient.New(ctx, addr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client, func() { _ = client.Close() }
}

// TestExecuteCommandMatrix runs all supported Execute commands against an in-process server (no kprobe).
//
//nolint:gocyclo // enumerates one subtest per Execute subcommand
func TestExecuteCommandMatrix(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	c, closeClient := connectClient(t, addr)
	defer closeClient()
	if _, err := c.Connect(ctx, ""); err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Run("break_obsolete_syntax", func(t *testing.T) {
		resp, err := c.Execute(ctx, "break do_sys_open")
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "obsolete") {
			t.Errorf("break bare symbol: want obsolete error, ok=%v err=%q", resp.GetOk(), resp.GetErrorMessage())
		}
	})
	t.Run("break_needs_bpf_include", func(t *testing.T) {
		resp, err := c.Execute(ctx, `break --attach kprobe:do_sys_open --source "int x;"`)
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetOk() {
			t.Fatal("break: expected failure without bpf include in in-process server")
		}
		if !strings.Contains(resp.GetErrorMessage(), "bpf include") {
			t.Errorf("break: want bpf include error, got %q", resp.GetErrorMessage())
		}
	})
	t.Run("tbreak", func(t *testing.T) {
		resp, err := c.Execute(ctx, `tbreak --attach kprobe:do_sys_open --source "int y;"`)
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetOk() {
			t.Fatal("tbreak: expected failure without bpf include")
		}
		if !strings.Contains(resp.GetErrorMessage(), "bpf include") {
			t.Logf("tbreak err: %s", resp.GetErrorMessage())
		}
	})
	t.Run("print_p", func(t *testing.T) {
		resp, err := c.Execute(ctx, "print pid")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("print: %s", resp.GetErrorMessage())
		}
		if resp.GetPrint() == nil {
			t.Error("print: want Print result")
		}
	})
	t.Run("trace_t", func(t *testing.T) {
		resp, err := c.Execute(ctx, "trace pid tgid")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("trace: %s", resp.GetErrorMessage())
		}
		if resp.GetTrace() == nil {
			t.Error("trace: want Trace result")
		}
	})
	t.Run("continue_c", func(t *testing.T) {
		resp, err := c.Execute(ctx, "continue")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("continue: %s", resp.GetErrorMessage())
		}
	})
	t.Run("help", func(t *testing.T) {
		resp, err := c.Execute(ctx, "help")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("help: %s", resp.GetErrorMessage())
		}
		if !strings.Contains(resp.GetOutput(), "break") {
			t.Errorf("help: output should list commands, got %q", resp.GetOutput())
		}
	})
	t.Run("help_break", func(t *testing.T) {
		resp, err := c.Execute(ctx, "help break")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("help break: %s", resp.GetErrorMessage())
		}
	})
	t.Run("info_break", func(t *testing.T) {
		resp, err := c.Execute(ctx, "info break")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("info break: %s", resp.GetErrorMessage())
		}
		if !strings.Contains(resp.GetOutput(), "breakpoints") {
			t.Errorf("info break: output should contain breakpoints, got %q", resp.GetOutput())
		}
	})
	t.Run("info_trace", func(t *testing.T) {
		resp, err := c.Execute(ctx, "info trace")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("info trace: %s", resp.GetErrorMessage())
		}
	})
	t.Run("info_watch", func(t *testing.T) {
		resp, err := c.Execute(ctx, "info watch")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("info watch: %s", resp.GetErrorMessage())
		}
	})
	t.Run("info_session", func(t *testing.T) {
		resp, err := c.Execute(ctx, "info session")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("info session: %s", resp.GetErrorMessage())
		}
		out := resp.GetOutput()
		if !strings.Contains(out, "session") {
			t.Errorf("info session: output should contain session, got %q", out)
		}
		if !strings.Contains(out, "hooks=") {
			t.Errorf("info session: output should include hooks= count, got %q", out)
		}
	})
	t.Run("info_hooks", func(t *testing.T) {
		resp, err := c.Execute(ctx, "info hooks")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("info hooks: %s", resp.GetErrorMessage())
		}
	})
	t.Run("list", func(t *testing.T) {
		resp, err := c.Execute(ctx, "list")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("list: %s", resp.GetErrorMessage())
		}
		if !strings.Contains(resp.GetOutput(), "symbol") {
			t.Logf("list: %s", resp.GetOutput())
		}
	})
	t.Run("list_symbol", func(t *testing.T) {
		resp, err := c.Execute(ctx, "list do_sys_open")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("list do_sys_open: %s", resp.GetErrorMessage())
		}
	})
	t.Run("bt", func(t *testing.T) {
		resp, err := c.Execute(ctx, "bt")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("bt: %s", resp.GetErrorMessage())
		}
	})
	t.Run("watch", func(t *testing.T) {
		resp, err := c.Execute(ctx, "watch pid")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("watch: %s", resp.GetErrorMessage())
		}
		if !strings.Contains(resp.GetOutput(), "watch") {
			t.Errorf("watch: output should contain watch, got %q", resp.GetOutput())
		}
	})
	t.Run("delete_watch", func(t *testing.T) {
		// We added one watch in previous test; get its id from info watch or use known id watch-1
		resp, err := c.Execute(ctx, "info watch")
		if err != nil || !resp.GetOk() {
			t.Skip("need info watch to get id")
		}
		if !strings.Contains(resp.GetOutput(), "watch-") {
			t.Skip("no watch id in output")
		}
		resp, err = c.Execute(ctx, "delete watch-1")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("delete watch-1: %s", resp.GetErrorMessage())
		}
	})
	t.Run("hook_list", func(t *testing.T) {
		resp, err := c.Execute(ctx, "hook list")
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetOk() {
			t.Fatalf("hook list: %s", resp.GetErrorMessage())
		}
		if !strings.Contains(resp.GetOutput(), "hooks") {
			t.Errorf("hook list: output should contain hooks, got %q", resp.GetOutput())
		}
	})
	t.Run("hook_delete_missing", func(t *testing.T) {
		resp, err := c.Execute(ctx, "hook delete hook-999")
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetOk() {
			t.Error("hook delete hook-999: want ok false")
		}
	})
	t.Run("unknown_command", func(t *testing.T) {
		resp, err := c.Execute(ctx, "unknown_cmd")
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetOk() {
			t.Error("unknown_cmd: want ok false")
		}
		if !strings.Contains(resp.GetErrorMessage(), "unknown") {
			t.Errorf("unknown_cmd: want error message containing unknown, got %q", resp.GetErrorMessage())
		}
	})
}
