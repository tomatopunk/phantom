package mcp

import (
	"context"

	"github.com/tomatopunk/phantom/pkg/api/proto"
)

// Backend is the interface the MCP server uses to run commands and list state.
type Backend interface {
	Connect(ctx context.Context, sessionID string) (string, error)
	Execute(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error)
	ListSessions(ctx context.Context) ([]string, error)
	ListBreakpoints(ctx context.Context, sessionID string) (string, error)
	ListHooks(ctx context.Context, sessionID string) (string, error)
}
