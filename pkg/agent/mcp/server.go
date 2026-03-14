package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Server runs the MCP JSON-RPC server over stdio.
type Server struct {
	backend Backend
}

// NewServer returns an MCP server that uses the given backend.
func NewServer(backend Backend) *Server {
	return &Server{backend: backend}
}

// Run reads JSON-RPC requests from stdin and writes responses to stdout until ctx is done.
func (s *Server) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	var mu sync.Mutex
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, readErr := reader.ReadBytes('\n')
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeResponse(&mu, nil, err, nil)
			continue
		}
		result, err := s.handleRequest(ctx, &req)
		s.writeResponse(&mu, req.ID, err, result)
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (*Server) writeResponse(mu *sync.Mutex, id any, err error, result any) {
	mu.Lock()
	defer mu.Unlock()
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id}
	if err != nil {
		resp.Error = &rpcError{Code: -32000, Message: err.Error()}
	} else {
		resp.Result = result
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		_ = err // stdout write failure; best-effort
	}
	_ = os.Stdout.Sync()
}

func (s *Server) handleRequest(ctx context.Context, req *jsonRPCRequest) (any, error) {
	switch req.Method {
	case "tools/call":
		return s.handleToolsCall(ctx, req.Params)
	default:
		return nil, fmt.Errorf("unknown method: %s", req.Method)
	}
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolsCallResult struct {
	Content []contentItem `json:"content"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (any, error) {
	var p toolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	args := p.Arguments
	if args == nil {
		args = make(map[string]any)
	}
	text, err := s.runTool(ctx, p.Name, args)
	if err != nil {
		return nil, err
	}
	return toolsCallResult{Content: []contentItem{{Type: "text", Text: text}}}, nil
}

//nolint:gocyclo,funlen // tool name switch with many cases and per-case logic
func (s *Server) runTool(ctx context.Context, name string, args map[string]any) (string, error) {
	str := func(k string) string {
		if v, ok := args[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	switch name {
	case "set_breakpoint":
		sid := str("session_id")
		if sid == "" {
			return "", fmt.Errorf("session_id required")
		}
		sym := str("symbol")
		if sym == "" {
			return "", fmt.Errorf("symbol required")
		}
		resp, err := s.backend.Execute(ctx, sid, "break "+sym)
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return "", fmt.Errorf("%s", resp.GetErrorMessage())
		}
		return resp.GetOutput(), nil
	case "run_command":
		sid := str("session_id")
		if sid == "" {
			return "", fmt.Errorf("session_id required")
		}
		cmd := str("command_line")
		if cmd == "" {
			return "", fmt.Errorf("command_line required")
		}
		resp, err := s.backend.Execute(ctx, sid, cmd)
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return resp.GetErrorMessage(), nil
		}
		return resp.GetOutput(), nil
	case "list_sessions":
		ids, err := s.backend.ListSessions(ctx)
		if err != nil {
			return "", err
		}
		out := ""
		for _, id := range ids {
			out += id + "\n"
		}
		return out, nil
	case "list_breakpoints":
		sid := str("session_id")
		if sid == "" {
			return "", fmt.Errorf("session_id required")
		}
		return s.backend.ListBreakpoints(ctx, sid)
	case "list_hooks":
		sid := str("session_id")
		if sid == "" {
			return "", fmt.Errorf("session_id required")
		}
		return s.backend.ListHooks(ctx, sid)
	case "add_c_hook":
		sid := str("session_id")
		if sid == "" {
			return "", fmt.Errorf("session_id required")
		}
		point := str("attach_point")
		code := str("code")
		sec := str("sec")
		if point == "" {
			return "", fmt.Errorf("attach_point required")
		}
		if code != "" && sec != "" {
			return "", fmt.Errorf("cannot use both code and sec (use one)")
		}
		if code == "" && sec == "" {
			return "", fmt.Errorf("code or sec required")
		}
		var cmd string
		if code != "" {
			cmd = "hook add --point " + point + " --lang c --code " + quoteCode(code)
		} else {
			cmd = "hook add --point " + point + " --lang c --sec " + sec
		}
		resp, err := s.backend.Execute(ctx, sid, cmd)
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return "", fmt.Errorf("%s", resp.GetErrorMessage())
		}
		return resp.GetOutput(), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func quoteCode(code string) string {
	s := strings.ReplaceAll(code, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
