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

//go:build linux

// Package e2e: BPF scenarios (recv path, file open, fork tracepoint, uprobe) behind E2E_SCENARIOS=1.
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
	"testing"
	"time"

	"github.com/tomatopunk/phantom/lib/agent/discovery"
	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
)

const e2eScenariosEnv = "E2E_SCENARIOS"

func requireE2EScenarios(t *testing.T) (root, agentBin, kprobeObj, bpfInclude string) {
	t.Helper()
	if runtime.GOOS != LinuxGOOS {
		t.Skip("e2e scenarios only on Linux")
	}
	if os.Getenv(e2eScenariosEnv) != "1" {
		t.Skip(e2eScenariosEnv + " not set")
	}
	root = FindRepoRoot(t)
	agentBin, kprobeObj = E2EConfig(t, root)
	SkipIfMissing(t, agentBin, kprobeObj)
	bpfInclude = filepath.Join(root, "src", "agent", "bpf", "include")
	if st, err := os.Stat(bpfInclude); err != nil || !st.IsDir() {
		t.Skipf("bpf include dir missing: %s", bpfInclude)
	}
	return root, agentBin, kprobeObj, bpfInclude
}

func pickOpenKprobeSymbol(t *testing.T) string {
	t.Helper()
	syms, err := discovery.ListKprobeSymbols("do_sys_open", 200)
	if err != nil {
		t.Skipf("kallsyms: %v", err)
		return ""
	}
	prefer := []string{"do_sys_openat2", "do_sys_open"}
	for _, want := range prefer {
		for _, s := range syms {
			if s == want {
				return want
			}
		}
	}
	// Fallback: any do_sys_open* text symbol
	for _, s := range syms {
		if strings.HasPrefix(s, "do_sys_open") {
			return s
		}
	}
	t.Skip("no do_sys_open* kprobe symbol in kallsyms")
	return ""
}

func uprobeHelperPath(t *testing.T, root string) string {
	t.Helper()
	if p := os.Getenv("E2E_UPROBE_HELPER"); p != "" {
		return p
	}
	return filepath.Join(root, "test", "e2e", "uprobe_helper", "uprobe_helper")
}

// serveHTTP10WithBody serves HTTP/1.0 with a non-empty body so the client exercises tcp_recvmsg.
func serveHTTP10WithBody(t *testing.T, lis net.Listener) {
	t.Helper()
	body := []byte("hello")
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
				resp := fmt.Sprintf(
					"HTTP/1.0 200 OK\r\nContent-Length: %d\r\n\r\n%s",
					len(body), string(body),
				)
				_, _ = c.Write([]byte(resp))
			}
		}(conn)
	}
}

// TestTcpdumpStyleTcpRecvmsg asserts break hit on tcp_recvmsg when the client receives a response body.
func TestTcpdumpStyleTcpRecvmsg(t *testing.T) {
	agentBin, kprobeObj := requireE2ENetwork(t)
	agentAddr := "127.0.0.1:19097"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer StopAgentProcess(agentCmd)

	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	port := lis.Addr().(*net.TCPAddr).Port
	go serveHTTP10WithBody(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := grpcclient.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, cerr := c.Connect(ctx, ""); cerr != nil {
		t.Fatalf("connect: %v", cerr)
	}
	br, err := c.Execute(ctx, "break tcp_recvmsg")
	if err != nil {
		t.Fatalf("break tcp_recvmsg: %v", err)
	}
	if !br.GetOk() {
		t.Fatalf("break tcp_recvmsg: %s", br.GetErrorMessage())
	}

	trigger := func() {
		//nolint:gosec // G204: URL uses localhost and port from this test's listener only
		cmd := exec.CommandContext(ctx, "curl", "-s", "--http1.0", fmt.Sprintf("http://127.0.0.1:%d/", port))
		_ = cmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 10*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected at least one break hit on tcp_recvmsg (client recv path)")
	}
}

// TestE2EOpenBreak attaches a breakpoint on the best-effort open syscall symbol and triggers a local open.
func TestE2EOpenBreak(t *testing.T) {
	_, agentBin, kprobeObj, _ := requireE2EScenarios(t)
	sym := pickOpenKprobeSymbol(t)
	agentAddr := "127.0.0.1:19098"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer StopAgentProcess(agentCmd)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := grpcclient.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, cerr := c.Connect(ctx, ""); cerr != nil {
		t.Fatalf("connect: %v", cerr)
	}
	resp, err := c.Execute(ctx, "break "+sym)
	if err != nil {
		t.Fatalf("break: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("break %s not attached: %s", sym, resp.GetErrorMessage())
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("phantom-e2e-open-%d", os.Getpid()))
	defer os.Remove(tmp)
	if err := os.WriteFile(tmp, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	trigger := func() {
		f, err := os.OpenFile(tmp, os.O_RDONLY, 0)
		if err == nil {
			_ = f.Close()
		}
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 12*time.Second, trigger)
	if count == 0 {
		t.Fatalf("expected break hit on %s", sym)
	}
}

// TestE2EForkTracepoint uses hook add on sched_process_fork and triggers a child process.
func TestE2EForkTracepoint(t *testing.T) {
	_, agentBin, kprobeObj, bpfInc := requireE2EScenarios(t)
	agentAddr := "127.0.0.1:19099"
	agentCmd := StartAgentWithBpfInclude(t, agentBin, kprobeObj, agentAddr, bpfInc)
	defer StopAgentProcess(agentCmd)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	c, err := grpcclient.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, cerr := c.Connect(ctx, ""); cerr != nil {
		t.Fatalf("connect: %v", cerr)
	}

	line := "hook add --point tracepoint:sched:sched_process_fork --lang c --code (void)0; --limit 32"
	resp, err := c.Execute(ctx, line)
	if err != nil {
		t.Fatalf("hook add: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("fork tracepoint hook: %s", resp.GetErrorMessage())
	}

	trigger := func() {
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "true")
		_ = cmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 15*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected at least one break hit from sched_process_fork hook")
	}
}

// TestE2EUprobeMarker attaches a uprobe on phantom_e2e_marker in the helper binary.
func TestE2EUprobeMarker(t *testing.T) {
	root, agentBin, kprobeObj, bpfInc := requireE2EScenarios(t)
	helper := uprobeHelperPath(t, root)
	if st, err := os.Stat(helper); err != nil || st.IsDir() {
		t.Skipf("uprobe helper binary missing (run make build-uprobe-e2e-helper): %s", helper)
	}
	absHelper, err := filepath.Abs(helper)
	if err != nil {
		t.Fatal(err)
	}

	agentAddr := "127.0.0.1:19100"
	agentCmd := StartAgentWithBpfInclude(t, agentBin, kprobeObj, agentAddr, bpfInc)
	defer StopAgentProcess(agentCmd)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	c, err := grpcclient.New(ctx, agentAddr, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()
	if _, cerr := c.Connect(ctx, ""); cerr != nil {
		t.Fatalf("connect: %v", cerr)
	}

	point := "uprobe:" + absHelper + ":phantom_e2e_marker"
	line := "hook add --point " + point + " --lang c --code (void)0; --limit 8"
	resp, err := c.Execute(ctx, line)
	if err != nil {
		t.Fatalf("hook add: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("uprobe hook: %s", resp.GetErrorMessage())
	}

	trigger := func() {
		cmd := exec.CommandContext(ctx, absHelper)
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 15*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected uprobe break hit on phantom_e2e_marker")
	}
}
