// Package grpcclient provides a minimal gRPC client for e2e tests (replaces removed pkg/cli/client).
package grpcclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const metadataKeyAuth = "authorization"

// Client talks to the remote debugger agent via gRPC.
type Client struct {
	conn   *grpc.ClientConn
	debug  proto.DebuggerServiceClient
	token  string
	sessID string
}

// New builds a client for the given agent address and optional token.
func New(_ context.Context, agentAddr, token string) (*Client, error) {
	conn, err := grpc.NewClient(agentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", agentAddr, err)
	}
	return &Client{
		conn:  conn,
		debug: proto.NewDebuggerServiceClient(conn),
		token: token,
	}, nil
}

// Connect creates or reuses a session; sessionID can be empty to let the server generate one.
func (c *Client) Connect(ctx context.Context, sessionID string) (string, error) {
	ctx = c.withAuth(ctx)
	resp, err := c.debug.OpenSession(ctx, &proto.OpenSessionRequest{SessionId: sessionID})
	if err != nil {
		return "", err
	}
	c.sessID = resp.SessionId
	return resp.SessionId, nil
}

// Execute runs one command line in the current session.
func (c *Client) Execute(ctx context.Context, commandLine string) (*proto.ExecuteResponse, error) {
	if c.sessID == "" {
		return nil, fmt.Errorf("not connected: call Connect first")
	}
	ctx = c.withAuth(ctx)
	return c.debug.Execute(ctx, &proto.ExecuteRequest{
		SessionId:   c.sessID,
		CommandLine: commandLine,
	})
}

// StreamEvents starts streaming debug events for the current session.
func (c *Client) StreamEvents(ctx context.Context) (proto.DebuggerService_StreamEventsClient, error) {
	if c.sessID == "" {
		return nil, fmt.Errorf("not connected: call Connect first")
	}
	ctx = c.withAuth(ctx)
	return c.debug.StreamEvents(ctx, &proto.StreamEventsRequest{SessionId: c.sessID})
}

// SessionID returns the current session id or empty if not connected.
func (c *Client) SessionID() string {
	return c.sessID
}

// Close releases the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) withAuth(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, metadataKeyAuth, "Bearer "+strings.TrimSpace(c.token))
}
