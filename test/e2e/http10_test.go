// Package e2e holds end-to-end tests that require Linux and optional env (e.g. E2E_HTTP10=1).
// They skip when prerequisites are not met so that go test ./... can run without failure.
package e2e

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tomatopunk/phantom/pkg/cli/client"
)

// TestHttp10CaptureE2E runs a generic HTTP/1.0 traffic e2e when E2E_HTTP10=1 on Linux:
// starts agent with kprobe only, break tcp_sendmsg, sends HTTP/1.0 request via curl,
// asserts at least one break hit event is received (no HTTP-specific event type).
//
// Agent must be given -kprobe <path> so that "break tcp_sendmsg" can load the kprobe
// object; otherwise the agent returns "no kprobe object path configured". Path can be
// set via E2E_KPROBE or PHANTOM_KPROBE; if unset, defaults to repo/bpf/probes/kernel/minikprobe.o.
func TestHttp10CaptureE2E(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("HTTP/1.0 e2e only on Linux")
	}
	if os.Getenv("E2E_HTTP10") != "1" {
		t.Skip("E2E_HTTP10 not set")
	}
	root := findRepoRoot(t)
	agentBin := os.Getenv("E2E_AGENT_BIN")
	if agentBin == "" {
		agentBin = filepath.Join(root, "phantom-agent")
	}
	kprobeObj := os.Getenv("E2E_KPROBE")
	if kprobeObj == "" {
		kprobeObj = os.Getenv("PHANTOM_KPROBE")
	}
	if kprobeObj == "" {
		kprobeObj = filepath.Join(root, "bpf", "probes", "kernel", "minikprobe.o")
	}
	for _, p := range []string{agentBin, kprobeObj} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("missing path %s: %v", p, err)
		}
	}

	agentPort := "19091"
	agentAddr := "127.0.0.1:" + agentPort
	agentCmd := exec.Command(agentBin, "-listen", agentAddr, "-kprobe", kprobeObj)
	agentCmd.Stdout = os.Stdout
	agentCmd.Stderr = os.Stderr
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}
	defer agentCmd.Process.Kill()
	time.Sleep(1 * time.Second)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	httpPort := lis.Addr().(*net.TCPAddr).Port
	go serveHTTP10(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := client.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, err := c.Connect(ctx, ""); err != nil {
		t.Fatalf("connect: %v", err)
	}
	_, err = c.Execute(ctx, "break tcp_sendmsg")
	if err != nil {
		t.Fatalf("break tcp_sendmsg: %v", err)
	}

	var mu sync.Mutex
	var breakHitCount int
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		stream, err := c.StreamEvents(streamCtx)
		if err != nil {
			return
		}
		for {
			ev, err := stream.Recv()
			if err != nil {
				return
			}
			if ev != nil && ev.GetEventType().String() == "EVENT_TYPE_BREAK_HIT" {
				mu.Lock()
				breakHitCount++
				mu.Unlock()
			}
		}
	}()
	time.Sleep(500 * time.Millisecond)
	curlCmd := exec.CommandContext(ctx, "curl", "-s", "--http1.0", fmt.Sprintf("http://127.0.0.1:%d/", httpPort))
	_ = curlCmd.Run()
	time.Sleep(1 * time.Second)
	streamCancel()
	cancel()
	mu.Lock()
	n := breakHitCount
	mu.Unlock()
	if n == 0 {
		t.Error("expected at least one break hit event on HTTP/1.0 traffic")
	}
}

func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func serveHTTP10(t *testing.T, lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			b := make([]byte, 4096)
			n, _ := c.Read(b)
			if n > 0 && strings.HasPrefix(string(b[:n]), "GET ") {
				c.Write([]byte("HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n"))
			}
		}(conn)
	}
}
