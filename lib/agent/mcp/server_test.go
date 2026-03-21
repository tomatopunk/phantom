package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/proto"
)

type fakeMCPBackend struct {
	executeFn func(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error)
}

func (f *fakeMCPBackend) Connect(ctx context.Context, sessionID string) (string, error) {
	return sessionID, nil
}

func (f *fakeMCPBackend) Execute(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error) {
	if f.executeFn != nil {
		return f.executeFn(ctx, sessionID, commandLine)
	}
	return &proto.ExecuteResponse{Ok: true, Output: "ok"}, nil
}

func (*fakeMCPBackend) ListSessions(context.Context) ([]string, error) {
	return nil, nil
}

func (*fakeMCPBackend) ListBreakpoints(context.Context, string) (string, error) {
	return "", nil
}

func (*fakeMCPBackend) ListHooks(context.Context, string) (string, error) {
	return "", nil
}

func TestRunCommandToolFailsLikeSetBreakpoint(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "break: nope"}, nil
		},
	})
	_, err := s.runTool(context.Background(), "run_command", map[string]any{
		"session_id":    "s1",
		"command_line":  "break foo",
	})
	if err == nil {
		t.Fatal("run_command: want error when Execute returns ok=false")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("run_command: want agent message, got %v", err)
	}
}

func TestSetBreakpointToolPropagatesExecuteError(t *testing.T) {
	s := NewServer(&fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "missing symbol"}, nil
		},
	})
	_, err := s.runTool(context.Background(), "set_breakpoint", map[string]any{
		"session_id": "s1",
		"symbol":     "x",
	})
	if err == nil || !strings.Contains(err.Error(), "missing symbol") {
		t.Fatalf("set_breakpoint: want missing symbol error, got %v", err)
	}
}

func TestExecuteCommandLineEmptyErrorMessage(t *testing.T) {
	b := &fakeMCPBackend{
		executeFn: func(_ context.Context, _, _ string) (*proto.ExecuteResponse, error) {
			return &proto.ExecuteResponse{Ok: false, ErrorMessage: "  "}, nil
		},
	}
	_, err := ExecuteCommandLine(context.Background(), b, "s", "x")
	if err == nil || !strings.Contains(err.Error(), "command failed") {
		t.Fatalf("want generic failure, got %v", err)
	}
}
