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
	CompileAndAttach(ctx context.Context, sessionID, source, attach, programName string) (*proto.CompileAndAttachResponse, error)
	ListTracepoints(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
	ListKprobeSymbols(ctx context.Context, prefix string, maxEntries uint32) ([]string, error)
}
