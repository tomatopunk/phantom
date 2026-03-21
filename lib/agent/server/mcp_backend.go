package server

import (
	"context"

	"github.com/tomatopunk/phantom/lib/agent/mcp"
	"github.com/tomatopunk/phantom/lib/proto"
)

// Ensure mcpBackendAdapter implements mcp.Backend.
var _ mcp.Backend = (*mcpBackendAdapter)(nil)

// mcpBackendAdapter adapts the debugger server to mcp.Backend.
type mcpBackendAdapter struct {
	s *debuggerServer
}

// NewMCPServerBackend returns an MCP backend that uses the given debugger server.
func NewMCPServerBackend(s *debuggerServer) mcp.Backend {
	return &mcpBackendAdapter{s: s}
}

func (a *mcpBackendAdapter) Connect(ctx context.Context, sessionID string) (string, error) {
	return a.s.ConnectSession(ctx, sessionID)
}

func (a *mcpBackendAdapter) Execute(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error) {
	return a.s.ExecuteCommand(ctx, sessionID, commandLine)
}

func (a *mcpBackendAdapter) ListSessions(ctx context.Context) ([]string, error) {
	return a.s.ListSessionsBackend(ctx), nil
}

func (a *mcpBackendAdapter) ListBreakpoints(ctx context.Context, sessionID string) (string, error) {
	return a.s.ListBreakpointsBackend(ctx, sessionID)
}

func (a *mcpBackendAdapter) ListHooks(ctx context.Context, sessionID string) (string, error) {
	return a.s.ListHooksBackend(ctx, sessionID)
}
