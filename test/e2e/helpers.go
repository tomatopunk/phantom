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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tomatopunk/phantom/lib/proto"
	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
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
	cmd := exec.Command(agentBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}
	time.Sleep(1 * time.Second)
	return cmd
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
	time.Sleep(300 * time.Millisecond)
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
			return n, out
		}
		time.Sleep(50 * time.Millisecond)
	}
	streamCancel()
	<-done
	mu.Lock()
	out := make([]*proto.DebugEvent, len(hits))
	copy(out, hits)
	mu.Unlock()
	return len(hits), out
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
			b := make([]byte, 4096)
			n, _ := c.Read(b)
			if n > 0 && len(b) >= 4 && string(b[:4]) == "GET " {
				c.Write([]byte("HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n"))
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
			b := make([]byte, 4096)
			n, _ := c.Read(b)
			if n > 0 && len(b) >= 4 && string(b[:4]) == "GET " {
				c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"))
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
			b := make([]byte, 256)
			_, _ = c.Read(b)
		}(conn)
	}
}
