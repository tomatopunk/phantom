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

// Package e2e: gRPC API coverage (Connect, Execute, StreamEvents, ListSessions, CloseSession).
package e2e

import (
	"context"
	"testing"

	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGrpcConnect creates and reuses a session.
func TestGrpcConnect(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	c, closeClient := connectClient(t, addr)
	defer closeClient()

	sid, err := c.Connect(ctx, "")
	if err != nil {
		t.Fatalf("Connect(empty): %v", err)
	}
	if sid == "" {
		t.Error("Connect: expected non-empty session id")
	}
	sid2, err := c.Connect(ctx, sid)
	if err != nil {
		t.Fatalf("Connect(reuse): %v", err)
	}
	if sid2 != sid {
		t.Errorf("Connect(reuse): want %q got %q", sid, sid2)
	}
}

// TestGrpcExecute runs a command and checks response shape.
func TestGrpcExecute(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	c, closeClient := connectClient(t, addr)
	defer closeClient()
	if _, err := c.Connect(ctx, ""); err != nil {
		t.Fatal(err)
	}
	resp, err := c.Execute(ctx, "help")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !resp.GetOk() {
		t.Errorf("Execute help: ok=false, msg=%s", resp.GetErrorMessage())
	}
	if resp.GetOutput() == "" {
		t.Error("Execute help: expected non-empty output")
	}
}

// TestGrpcStreamEvents starts streaming (no traffic in this test; just ensure stream works).
func TestGrpcStreamEvents(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	c, closeClient := connectClient(t, addr)
	defer closeClient()
	if _, err := c.Connect(ctx, ""); err != nil {
		t.Fatal(err)
	}
	stream, err := c.StreamEvents(ctx)
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	// Receive one or just cancel; we only need to confirm stream is established
	ctxCancel, cancel := context.WithCancel(ctx)
	cancel()
	_ = stream
	_ = ctxCancel
}

// TestGrpcListSessions lists sessions after creating two.
func TestGrpcListSessions(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	debug := proto.NewDebuggerServiceClient(conn)

	_, err = debug.OpenSession(ctx, &proto.OpenSessionRequest{SessionId: ""})
	if err != nil {
		t.Fatal(err)
	}
	resp2, err := debug.OpenSession(ctx, &proto.OpenSessionRequest{SessionId: "custom-id"})
	if err != nil {
		t.Fatal(err)
	}
	if resp2.SessionId != "custom-id" {
		t.Errorf("Connect(custom-id): want custom-id got %q", resp2.SessionId)
	}
	listResp, err := debug.ListSessions(ctx, &proto.ListSessionsRequest{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(listResp.SessionIds) < 2 {
		t.Errorf("ListSessions: want at least 2 sessions, got %v", listResp.SessionIds)
	}
}

// TestGrpcCloseSession closes a session and verifies Execute fails for it.
func TestGrpcCloseSession(t *testing.T) {
	addr, stop := startInProcessServer(t)
	defer stop()
	ctx := context.Background()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	debug := proto.NewDebuggerServiceClient(conn)

	connectResp, err := debug.OpenSession(ctx, &proto.OpenSessionRequest{SessionId: ""})
	if err != nil {
		t.Fatal(err)
	}
	sid := connectResp.SessionId
	_, err = debug.Execute(ctx, &proto.ExecuteRequest{SessionId: sid, CommandLine: "help"})
	if err != nil {
		t.Fatalf("Execute before close: %v", err)
	}
	_, err = debug.CloseSession(ctx, &proto.CloseSessionRequest{SessionId: sid})
	if err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	resp, err := debug.Execute(ctx, &proto.ExecuteRequest{SessionId: sid, CommandLine: "help"})
	if err != nil {
		t.Fatalf("Execute after close (transport): %v", err)
	}
	if resp.GetOk() {
		t.Error("Execute after close: expected ok=false (session not found)")
	}
	if resp.GetErrorMessage() != "session not found" {
		t.Logf("Execute after close: %s", resp.GetErrorMessage())
	}
}
