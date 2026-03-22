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

// Package e2e helpers for agent start, session, and event polling.
package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tomatopunk/phantom/lib/proto"
	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
)

const (
	// LinuxGOOS is runtime.GOOS on Linux targets (e2e skips elsewhere).
	LinuxGOOS = "linux"

	// StreamEvents must be subscribed before traffic; CI runners need a bit more than local dev.
	e2eStreamAttachDelay = 500 * time.Millisecond
	// After canceling StreamEvents, wait for the recv loop to exit before returning (avoids gRPC hangs at test teardown).
	e2eStreamGoroutineWait = 5 * time.Second
	e2ePollInterval        = 50 * time.Millisecond
	e2eHTTPReadBuf         = 4096
	e2eRawTCPReadBuf       = 256
	httpMethodGETSpace     = "GET "
	// How long StartAgent waits for the agent's gRPC listen address to accept TCP (agent may load BPF first).
	e2eAgentListenWait = 20 * time.Second
	e2eAgentListenPoll = 50 * time.Millisecond
)

// FindRepoRoot returns the repository root (directory containing go.mod).
func FindRepoRoot(t *testing.T) string {
	t.Helper()
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

// E2EConfig holds paths for agent and kprobe; call SkipIfMissing to skip when binaries are absent.
func E2EConfig(t *testing.T, root string) (agentBin, kprobeObj string) {
	t.Helper()
	agentBin = os.Getenv("E2E_AGENT_BIN")
	if agentBin == "" {
		agentBin = filepath.Join(root, "phantom-agent")
	}
	kprobeObj = os.Getenv("E2E_KPROBE")
	if kprobeObj == "" {
		kprobeObj = os.Getenv("PHANTOM_KPROBE")
	}
	if kprobeObj == "" {
		kprobeObj = filepath.Join(root, "src", "agent", "bpf", "probes", "kernel", "minikprobe.o")
	}
	return agentBin, kprobeObj
}

// SkipIfMissing skips the test if any of the given paths are missing.
func SkipIfMissing(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("missing path %s: %v", p, err)
		}
	}
}

// e2eAgentUseSudo is true when E2E_AGENT_USE_SUDO=1, or on Linux GitHub Actions (GITHUB_ACTIONS=true).
// Shell e2e runs the agent under sudo there; Go e2e matches that because file caps on ./phantom-agent are
// lost whenever `go build -o phantom-agent` runs, and setcap can fail silently in scripts.
// Set E2E_AGENT_USE_SUDO=0 to force the non-sudo path on GHA (e.g. when relying on setcap after a manual build).
func e2eAgentUseSudo() bool {
	if os.Getenv("E2E_AGENT_USE_SUDO") == "0" {
		return false
	}
	if os.Getenv("E2E_AGENT_USE_SUDO") == "1" {
		return true
	}
	if runtime.GOOS != LinuxGOOS {
		return false
	}
	v := os.Getenv("GITHUB_ACTIONS")
	return v == "true" || v == "1"
}

// waitForAgentTCP dials listenAddr until it succeeds or ctx is done (agent crashed, wrong port, or still loading).
func waitForAgentTCP(ctx context.Context, listenAddr string) error {
	var d net.Dialer
	ticker := time.NewTicker(e2eAgentListenPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w waiting for agent on %s", ctx.Err(), listenAddr)
		default:
		}
		c, err := d.DialContext(ctx, "tcp", listenAddr)
		if err == nil {
			_ = c.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w waiting for agent on %s", ctx.Err(), listenAddr)
		case <-ticker.C:
		}
	}
}

// StartAgent starts the agent with kprobe; caller must kill the process when done.
func StartAgent(t *testing.T, agentBin, kprobeObj, listenAddr string) *exec.Cmd {
	t.Helper()
	return StartAgentWithBpfInclude(t, agentBin, kprobeObj, listenAddr, "")
}

// StartAgentWithBpfInclude starts the agent with kprobe and optional -bpf-include (needed for hook add / tracepoint / uprobe compile).
func StartAgentWithBpfInclude(t *testing.T, agentBin, kprobeObj, listenAddr, bpfIncludeDir string) *exec.Cmd {
	t.Helper()
	args := []string{"-listen", listenAddr, "-kprobe", kprobeObj}
	if bpfIncludeDir != "" {
		args = append(args, "-bpf-include", bpfIncludeDir)
	}
	if v := os.Getenv("E2E_VMLINUX"); v != "" {
		args = append(args, "-vmlinux", v)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" && !e2eAgentUseSudo() {
		// Go's Linux SysProcAttr has no memlock rlimit; raise soft memlock via bash before exec
		// (matches scripts/e2e_linux_bpf_env.sh). exec replaces bash so cmd.Process targets the agent.
		const bashPrelude = `ulimit -l unlimited 2>/dev/null || true; exec "$0" "$@"`
		bashArgs := append([]string{"-c", bashPrelude, agentBin}, args...)
		cmd = exec.CommandContext(context.Background(), "bash", bashArgs...) // #nosec G204
	} else if e2eAgentUseSudo() {
		if os.Getenv("GITHUB_ACTIONS") == "true" || os.Getenv("GITHUB_ACTIONS") == "1" {
			t.Log("e2e: starting agent under sudo -n -E + bash ulimit (GitHub Actions)")
		} else {
			t.Log("e2e: starting agent under sudo -n -E + bash ulimit (E2E_AGENT_USE_SUDO=1)")
		}
		// Same as phantom_e2e_run_agent_sudo: root + soft memlock before exec (reliable when file caps were stripped by go build).
		sudoArgs := append([]string{"-n", "-E", "bash", "-c", `ulimit -l unlimited 2>/dev/null || true; exec "$0" "$@"`, agentBin}, args...)
		cmd = exec.CommandContext(context.Background(), "sudo", sudoArgs...) // #nosec G204
	} else {
		cmd = exec.CommandContext(context.Background(), agentBin, args...) // #nosec G204
	}
	// Avoid wiring agent I/O to os.Stdout/os.Stderr: exec.Cmd waits for copy goroutines (WaitDelay) and
	// tests can hang at package shutdown if the agent process was only Kill'd without Wait.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), e2eAgentListenWait)
	defer cancel()
	if err := waitForAgentTCP(waitCtx, listenAddr); err != nil {
		StopAgentProcess(cmd)
		t.Fatalf("agent not listening on %s: %v (if BPF/memlock, run shell e2e first for setcap, or E2E_AGENT_USE_SUDO=1)", listenAddr, err)
	}
	return cmd
}

// StopAgentProcess kills an agent process started by StartAgent*; errors are ignored.
func StopAgentProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	// Reap the child (and sudo wrapper); required so os/exec completes I/O wait and the test process can exit.
	_ = cmd.Wait()
}

// WaitForBreakHits starts StreamEvents, runs trigger, and waits until at least minHits
// EVENT_TYPE_BREAK_HIT events are seen or timeout. Returns count and collected events (for L3/L4 asserts).
func WaitForBreakHits(
	ctx context.Context,
	c *grpcclient.Client,
	minHits int,
	timeout time.Duration,
	trigger func(),
) (count int, events []*proto.DebugEvent) {
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	var mu sync.Mutex
	var hits []*proto.DebugEvent
	done := make(chan struct{})
	go func() {
		stream, err := c.StreamEvents(streamCtx)
		if err != nil {
			close(done)
			return
		}
		for {
			ev, err := stream.Recv()
			if err != nil {
				close(done)
				return
			}
			if ev != nil && ev.GetEventType() == proto.EventType_EVENT_TYPE_BREAK_HIT {
				mu.Lock()
				hits = append(hits, ev)
				mu.Unlock()
			}
		}
	}()

	// Allow stream to attach
	time.Sleep(e2eStreamAttachDelay)
	trigger()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(hits)
		mu.Unlock()
		if n >= minHits {
			streamCancel()
			mu.Lock()
			out := make([]*proto.DebugEvent, len(hits))
			copy(out, hits)
			mu.Unlock()
			select {
			case <-done:
			case <-time.After(e2eStreamGoroutineWait):
			}
			return n, out
		}
		time.Sleep(e2ePollInterval)
	}
	streamCancel()
	<-done
	mu.Lock()
	out := make([]*proto.DebugEvent, len(hits))
	copy(out, hits)
	mu.Unlock()
	return len(hits), out
}

// WaitForTraceSamples starts StreamEvents, runs trigger, and waits until at least minSamples
// EVENT_TYPE_TRACE_SAMPLE events are seen or timeout. Used to assert hook-driven trace
// after ProcessProbeEvent (hook path + trace command).
func WaitForTraceSamples(
	ctx context.Context,
	c *grpcclient.Client,
	minSamples int,
	timeout time.Duration,
	trigger func(),
) (count int, events []*proto.DebugEvent) {
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	var mu sync.Mutex
	var samples []*proto.DebugEvent
	done := make(chan struct{})
	go func() {
		stream, err := c.StreamEvents(streamCtx)
		if err != nil {
			close(done)
			return
		}
		for {
			ev, err := stream.Recv()
			if err != nil {
				close(done)
				return
			}
			if ev != nil && ev.GetEventType() == proto.EventType_EVENT_TYPE_TRACE_SAMPLE {
				mu.Lock()
				samples = append(samples, ev)
				mu.Unlock()
			}
		}
	}()

	time.Sleep(e2eStreamAttachDelay)
	trigger()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(samples)
		mu.Unlock()
		if n >= minSamples {
			streamCancel()
			mu.Lock()
			out := make([]*proto.DebugEvent, len(samples))
			copy(out, samples)
			mu.Unlock()
			select {
			case <-done:
			case <-time.After(e2eStreamGoroutineWait):
			}
			return n, out
		}
		time.Sleep(e2ePollInterval)
	}
	streamCancel()
	<-done
	mu.Lock()
	out := make([]*proto.DebugEvent, len(samples))
	copy(out, samples)
	mu.Unlock()
	return len(samples), out
}

// ServeHTTP10 handles HTTP/1.0 GET on the listener (minimal response).
func ServeHTTP10(t *testing.T, lis net.Listener) {
	t.Helper()
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			b := make([]byte, e2eHTTPReadBuf)
			n, _ := c.Read(b)
			if n > 0 && len(b) >= 4 && string(b[:4]) == httpMethodGETSpace {
				_, _ = c.Write([]byte("HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n"))
			}
		}(conn)
	}
}

// ServeHTTP11 handles HTTP/1.1 GET (with Connection: keep-alive); same minimal response.
func ServeHTTP11(t *testing.T, lis net.Listener) {
	t.Helper()
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			b := make([]byte, e2eHTTPReadBuf)
			n, _ := c.Read(b)
			if n > 0 && len(b) >= 4 && string(b[:4]) == httpMethodGETSpace {
				_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"))
			}
		}(conn)
	}
}

// ServeRawTCP accepts connections and echoes nothing (just accept/close or read a bit); used to trigger tcp_sendmsg.
func ServeRawTCP(t *testing.T, lis net.Listener) {
	t.Helper()
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			b := make([]byte, e2eRawTCPReadBuf)
			_, _ = c.Read(b)
		}(conn)
	}
}
