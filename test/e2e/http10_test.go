// Package e2e holds end-to-end tests that require Linux and optional env (e.g. E2E_HTTP10=1).
// They skip when prerequisites are not met so that go test ./... can run without failure.
package e2e

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
)

// TestHttp10CaptureE2E runs a generic HTTP/1.0 traffic e2e when E2E_HTTP10=1 on Linux:
// starts agent with kprobe only, break tcp_sendmsg, sends HTTP/1.0 request via curl,
// asserts at least one break hit event is received (poll-based wait, no fixed sleep).
func TestHttp10CaptureE2E(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("HTTP/1.0 e2e only on Linux")
	}
	if os.Getenv("E2E_HTTP10") != "1" {
		t.Skip("E2E_HTTP10 not set")
	}
	root := FindRepoRoot(t)
	agentBin, kprobeObj := E2EConfig(t, root)
	SkipIfMissing(t, agentBin, kprobeObj)

	agentAddr := "127.0.0.1:19091"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer agentCmd.Process.Kill()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	httpPort := lis.Addr().(*net.TCPAddr).Port
	go ServeHTTP10(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := grpcclient.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, err := c.Connect(ctx, ""); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := c.Execute(ctx, "break tcp_sendmsg"); err != nil {
		t.Fatalf("break tcp_sendmsg: %v", err)
	}

	trigger := func() {
		curlCmd := exec.CommandContext(ctx, "curl", "-s", "--http1.0", fmt.Sprintf("http://127.0.0.1:%d/", httpPort))
		_ = curlCmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 8*time.Second, trigger)
	if count == 0 {
		t.Error("expected at least one break hit event on HTTP/1.0 traffic")
	}
}
