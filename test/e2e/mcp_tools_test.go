// Package e2e: MCP tools coverage (set_breakpoint, run_command, list_sessions, list_breakpoints, list_hooks, add_c_hook).
package e2e

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/pkg/agent/mcp"
	"github.com/tomatopunk/phantom/pkg/agent/server"
	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
	"google.golang.org/grpc"
)

func startDebuggerServerWithBackend(t *testing.T) (addr string, backend mcp.Backend, cleanup func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = lis.Addr().String()
	mgr := session.NewManager("")
	ds := server.NewDebuggerServer(mgr)
	srv := grpc.NewServer()
	proto.RegisterDebuggerServiceServer(srv, ds)
	go func() { _ = srv.Serve(lis) }()
	backend = server.NewMCPServerBackend(ds)
	cleanup = func() { srv.GracefulStop(); _ = lis.Close() }
	return addr, backend, cleanup
}

// TestMCPConnectAndListSessions tests Connect and list_sessions via backend.
func TestMCPConnectAndListSessions(t *testing.T) {
	_, backend, cleanup := startDebuggerServerWithBackend(t)
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

// TestMCPSetBreakpoint runs set_breakpoint via backend (Execute "break sym"); no kprobe so expect error.
func TestMCPSetBreakpoint(t *testing.T) {
	_, backend, cleanup := startDebuggerServerWithBackend(t)
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
		if resp.GetBreakpoint() == nil || resp.GetBreakpoint().GetSymbol() != "tcp_sendmsg" {
			t.Errorf("set_breakpoint: want symbol tcp_sendmsg, got %v", resp.GetBreakpoint())
		}
	} else {
		if !strings.Contains(resp.GetErrorMessage(), "kprobe") && !strings.Contains(resp.GetErrorMessage(), "no kprobe") {
			t.Logf("set_breakpoint (no kprobe): %s", resp.GetErrorMessage())
		}
	}
}

// TestMCPRunCommand runs run_command via backend.
func TestMCPRunCommand(t *testing.T) {
	_, backend, cleanup := startDebuggerServerWithBackend(t)
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
	_, backend, cleanup := startDebuggerServerWithBackend(t)
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
	_, backend, cleanup := startDebuggerServerWithBackend(t)
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

// TestMCPAddCHook tests add_c_hook via Execute (hook add); expect error (no bpf include dir).
func TestMCPAddCHook(t *testing.T) {
	_, backend, cleanup := startDebuggerServerWithBackend(t)
	defer cleanup()
	ctx := context.Background()
	sid, err := backend.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := backend.Execute(ctx, sid, "hook add --point kprobe:do_sys_open --lang c --sec pid==1")
	if err != nil {
		t.Fatalf("Execute hook add: %v", err)
	}
	if resp.GetOk() {
		t.Error("add_c_hook: expected ok=false (no bpf include dir)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "include") && !strings.Contains(resp.GetErrorMessage(), "bpf") {
		t.Logf("add_c_hook error: %s", resp.GetErrorMessage())
	}
}
