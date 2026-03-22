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

package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
)

// ExecuteCommandLine runs a debugger command and maps application-level failure (ok=false) to an error,
// matching REPL / gRPC semantics for MCP tools.
func ExecuteCommandLine(ctx context.Context, b Backend, sessionID, commandLine string) (string, error) {
	resp, err := b.Execute(ctx, sessionID, commandLine)
	if err != nil {
		return "", err
	}
	if !resp.GetOk() {
		msg := strings.TrimSpace(resp.GetErrorMessage())
		if msg == "" {
			return "", fmt.Errorf("command failed")
		}
		return "", fmt.Errorf("%s", msg)
	}
	return resp.GetOutput(), nil
}

// Backend is the interface the MCP server uses to run commands and list state.
type Backend interface {
	Connect(ctx context.Context, sessionID string) (string, error)
	Execute(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error)
	ListSessions(ctx context.Context) ([]string, error)
	ListBreakpoints(ctx context.Context, sessionID string) (string, error)
	ListHooks(ctx context.Context, sessionID string) (string, error)
	CompileAndAttach(ctx context.Context, sessionID, source, attach, programName string, limit uint32) (*proto.CompileAndAttachResponse, error)
	ListTracepoints(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
	ListKprobeSymbols(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
}
