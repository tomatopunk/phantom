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
	"testing"
	"time"

	"github.com/tomatopunk/phantom/test/e2e/grpcclient"
)

// TestHttp10CaptureE2E runs a generic HTTP/1.0 traffic e2e when E2E_HTTP10=1 on Linux:
// starts agent with kprobe only, break tcp_sendmsg, sends HTTP/1.0 request via curl,
// asserts at least one break hit event is received (poll-based wait, no fixed sleep).
func TestHttp10CaptureE2E(t *testing.T) {
	if runtime.GOOS != LinuxGOOS {
		t.Skip("HTTP/1.0 e2e only on Linux")
	}
	if os.Getenv("E2E_HTTP10") != "1" {
		t.Skip("E2E_HTTP10 not set")
	}
	root := FindRepoRoot(t)
	agentBin, kprobeObj := E2EConfig(t, root)
	SkipIfMissing(t, agentBin, kprobeObj)
	bpfInc := filepath.Join(root, "src", "agent", "bpf", "include")
	SkipIfMissing(t, bpfInc)

	agentAddr := "127.0.0.1:19091"
	agentCmd := StartAgentWithBpfInclude(t, agentBin, kprobeObj, agentAddr, bpfInc)
	defer StopAgentProcess(agentCmd)

	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
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
	if _, cerr := c.Connect(ctx, ""); cerr != nil {
		t.Fatalf("connect: %v", cerr)
	}
	cr, err := c.CompileAndAttach(ctx, MinimalKprobeRingbufC("tcp_sendmsg"), "kprobe:tcp_sendmsg", "", 0)
	if err != nil {
		t.Fatalf("CompileAndAttach tcp_sendmsg: %v", err)
	}
	if !cr.GetOk() {
		t.Fatalf("CompileAndAttach tcp_sendmsg: %s", cr.GetErrorMessage())
	}

	trigger := func() {
		//nolint:gosec // G204: URL uses localhost and port from this test's listener only
		curlCmd := exec.CommandContext(ctx, "curl", "-s", "--http1.0", fmt.Sprintf("http://127.0.0.1:%d/", httpPort))
		_ = curlCmd.Run()
	}
	count, _ := WaitForBreakHits(ctx, c, 1, 8*time.Second, trigger)
	if count == 0 {
		t.Error("expected at least one break hit event on HTTP/1.0 traffic")
	}
}
