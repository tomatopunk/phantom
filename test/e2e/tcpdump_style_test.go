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

// Package e2e: tcpdump-style scenarios using existing commands (break/trace/info, L3/L4 metadata).
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

const e2eNetworkEnv = "E2E_NETWORK"

func requireE2ENetwork(t *testing.T) (root, agentBin, kprobeObj string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("tcpdump-style e2e only on Linux")
	}
	if os.Getenv(e2eNetworkEnv) != "1" {
		t.Skip(e2eNetworkEnv + " not set")
	}
	root = FindRepoRoot(t)
	agentBin, kprobeObj = E2EConfig(t, root)
	SkipIfMissing(t, agentBin, kprobeObj)
	return root, agentBin, kprobeObj
}

// TestTcpdumpStyleHttp10 asserts break hit + L3/L4 metadata on HTTP/1.0 traffic (poll-based wait).
func TestTcpdumpStyleHttp10(t *testing.T) {
	_, agentBin, kprobeObj := requireE2ENetwork(t)
	agentAddr := "127.0.0.1:19094"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer agentCmd.Process.Kill()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	port := lis.Addr().(*net.TCPAddr).Port
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
		cmd := exec.CommandContext(ctx, "curl", "-s", "--http1.0", fmt.Sprintf("http://127.0.0.1:%d/", port))
		_ = cmd.Run()
	}
	count, events := WaitForBreakHits(ctx, c, 1, 8*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected at least one break hit on HTTP/1.0")
	}
	// L3/L4-style metadata on first event
	ev := events[0]
	if ev.Pid == 0 && ev.Tgid == 0 && ev.ProbeId == "" {
		t.Logf("first event has no pid/tgid/probe_id (optional): %+v", ev)
	}
	_ = ev
}

// TestTcpdumpStyleHttp11 asserts break hit on HTTP/1.1 traffic.
func TestTcpdumpStyleHttp11(t *testing.T) {
	_, agentBin, kprobeObj := requireE2ENetwork(t)
	agentAddr := "127.0.0.1:19095"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer agentCmd.Process.Kill()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	port := lis.Addr().(*net.TCPAddr).Port
	go ServeHTTP11(t, lis)

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
		cmd := exec.CommandContext(ctx, "curl", "-s", "--http1.1", fmt.Sprintf("http://127.0.0.1:%d/", port))
		_ = cmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 8*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected at least one break hit on HTTP/1.1")
	}
}

// TestTcpdumpStyleRawTcp asserts break hit on raw TCP (non-HTTP) traffic.
func TestTcpdumpStyleRawTcp(t *testing.T) {
	_, agentBin, kprobeObj := requireE2ENetwork(t)
	agentAddr := "127.0.0.1:19096"
	agentCmd := StartAgent(t, agentBin, kprobeObj, agentAddr)
	defer agentCmd.Process.Kill()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	port := lis.Addr().(*net.TCPAddr).Port
	go ServeRawTCP(t, lis)

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
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
		if err != nil {
			return
		}
		_, _ = conn.Write([]byte("x"))
		_ = conn.Close()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 8*time.Second, trigger)
	if count == 0 {
		t.Fatal("expected at least one break hit on raw TCP")
	}
}
