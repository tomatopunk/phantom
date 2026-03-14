package repl

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/pkg/agent/server"
	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
	"github.com/tomatopunk/phantom/pkg/cli/client"
	"google.golang.org/grpc"
)

// TestBreakPrintTraceE2E starts an in-process server and client and verifies break/print/trace commands.
//
//nolint:gocyclo // E2E test with multiple steps
func TestBreakPrintTraceE2E(t *testing.T) {
	ctx := context.Background()
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	addr := lis.Addr().String()

	srv := grpc.NewServer()
	mgr := session.NewManager("") // no kprobe path in test; break will fail, print/trace still work
	proto.RegisterDebuggerServiceServer(srv, server.NewDebuggerServer(mgr))
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	c, err := client.New(ctx, addr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()

	if _, connErr := c.Connect(ctx, ""); connErr != nil {
		t.Fatalf("connect: %v", connErr)
	}

	// break do_sys_open (without kprobe path returns error; with path would attach)
	resp, err := c.Execute(ctx, "break do_sys_open")
	if err != nil {
		t.Fatalf("execute break: %v", err)
	}
	if resp.GetOk() {
		if resp.GetBreakpoint() == nil || resp.GetBreakpoint().GetSymbol() != "do_sys_open" {
			t.Errorf("break: expected symbol do_sys_open, got %v", resp.GetBreakpoint())
		}
	} else {
		if resp.GetErrorMessage() != "break: no kprobe object path configured" {
			t.Logf("break (no path): %s", resp.GetErrorMessage())
		}
	}

	// print pid
	resp, err = c.Execute(ctx, "print pid")
	if err != nil {
		t.Fatalf("execute print: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("print: %s", resp.GetErrorMessage())
	}
	if resp.GetPrint() == nil {
		t.Error("print: expected Print result")
	}

	// trace arg0
	resp, err = c.Execute(ctx, "trace arg0")
	if err != nil {
		t.Fatalf("execute trace: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("trace: %s", resp.GetErrorMessage())
		if resp.GetTrace() == nil {
			t.Error("trace: expected Trace result")
		}
	}
	if resp.GetTrace() != nil && !strings.Contains(strings.Join(resp.GetTrace().GetExpressions(), ","), "arg0") {
		t.Errorf("trace: expected arg0 in expressions")
	}
}
