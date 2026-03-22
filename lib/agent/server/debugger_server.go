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

package server

import (
	"context"
	"strconv"
	"strings"

	"github.com/cilium/ebpf/btf"
	"github.com/tomatopunk/phantom/lib/agent/probe"
	"github.com/tomatopunk/phantom/lib/agent/runtime"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// debuggerServer implements proto.DebuggerServiceServer using SessionManager and executor.
type debuggerServer struct {
	proto.UnimplementedDebuggerServiceServer
	sessions *session.Manager
	exec     *commandExecutor
	cfg      *serverConfig
}

// serverConfig holds optional rate limit, quota, audit, hook include path, and vmlinux path (injected by Run).
type serverConfig struct {
	rateLimiter   *RateLimiter
	quota         *SessionQuota
	audit         AuditLogger
	bpfIncludeDir string
	vmlinuxPath   string
	btfSpec       *btf.Spec // kernel or vmlinux BTF for CO-RE; optional
}

// AuditLogger is implemented by AuditLog and NopAuditLog.
type AuditLogger interface {
	LogCommand(sessionID, commandLine string, ok bool, errMsg string)
}

// NewDebuggerServer returns a DebuggerService server that uses the given session manager.
func NewDebuggerServer(sessions *session.Manager) *debuggerServer {
	return &debuggerServer{
		sessions: sessions,
		exec:     newCommandExecutor("", "", probe.NewPlanner(), nil, nil),
		cfg:      nil,
	}
}

// NewDebuggerServerWithConfig returns a server with rate limit, quota, audit, hook include dir, and vmlinux path.
func NewDebuggerServerWithConfig(sessions *session.Manager, cfg *serverConfig) *debuggerServer {
	includeDir := ""
	vmlinuxPath := ""
	var btfSpec *btf.Spec
	var quota *SessionQuota
	if cfg != nil {
		includeDir = cfg.bpfIncludeDir
		vmlinuxPath = cfg.vmlinuxPath
		btfSpec = cfg.btfSpec
		quota = cfg.quota
	}
	return &debuggerServer{
		sessions: sessions,
		exec:     newCommandExecutor(includeDir, vmlinuxPath, probe.NewPlanner(), btfSpec, quota),
		cfg:      cfg,
	}
}

// OpenSession creates or reuses a session and returns its id.
func (s *debuggerServer) OpenSession(ctx context.Context, req *proto.OpenSessionRequest) (*proto.OpenSessionResponse, error) {
	sid := req.GetSessionId()
	if sid == "" {
		sid = generateSessionID()
	}
	_, err := s.sessions.GetOrCreate(ctx, sid)
	if err != nil {
		return nil, err
	}
	SetSessionsActive(len(s.sessions.List()))
	DebugLogf("open_session session_id=%s requested=%q", sid, req.GetSessionId())
	return &proto.OpenSessionResponse{SessionId: sid, Message: "connected"}, nil
}

// Execute runs one command line in the session and returns the result.
func (s *debuggerServer) Execute(ctx context.Context, req *proto.ExecuteRequest) (*proto.ExecuteResponse, error) {
	sid := req.GetSessionId()
	if sid == "" {
		DebugLogf("grpc.Execute reject: missing session_id cmd=%q", truncateForLog(strings.TrimSpace(req.GetCommandLine()), 200))
		return errResponse("missing session_id"), nil
	}
	sess := s.sessions.Get(sid)
	if sess == nil {
		DebugLogf("grpc.Execute reject: session not found session=%s cmd=%q", sid, truncateForLog(strings.TrimSpace(req.GetCommandLine()), 200))
		return errResponse("session not found"), nil
	}
	if s.cfg != nil && s.cfg.rateLimiter != nil && !s.cfg.rateLimiter.Allow(sid) {
		DebugLogf("grpc.Execute reject: rate limited session=%s", sid)
		return errResponse("rate limited"), nil
	}
	line := strings.TrimSpace(req.GetCommandLine())
	if s.cfg != nil && s.cfg.quota != nil {
		if errMsg := checkQuota(s.cfg.quota, sid, line); errMsg != "" {
			DebugLogf("grpc.Execute reject: quota session=%s err=%q cmd=%q", sid, errMsg, truncateForLog(line, 200))
			return errResponse(errMsg), nil
		}
	}
	resp, err := s.exec.execute(ctx, sess, line)
	logExecuteDebug("grpc.Execute", sid, line, resp, err)
	if err != nil {
		return nil, err
	}
	if s.cfg != nil && s.cfg.quota != nil && !resp.GetOk() {
		rollbackBreakTbreakQuota(s.cfg.quota, sid, line)
	}
	IncCommandsTotal(sid, resp.GetOk())
	if s.cfg != nil && s.cfg.audit != nil {
		errMsg := resp.GetErrorMessage()
		s.cfg.audit.LogCommand(sid, line, resp.GetOk(), errMsg)
	}
	return resp, nil
}

// StreamEvents streams debug events for the session from the session's ringbuf pump.
func (s *debuggerServer) StreamEvents(req *proto.StreamEventsRequest, stream proto.DebuggerService_StreamEventsServer) error {
	sid := req.GetSessionId()
	if sid == "" {
		return nil
	}
	sess := s.sessions.Get(sid)
	if sess == nil {
		return nil
	}
	DebugLogf("stream_events start session_id=%s", sid)
	const eventChanCap = 64
	evCh := make(chan *runtime.Event, eventChanCap)
	sess.SubscribeEvents(evCh)
	defer sess.UnsubscribeEvents(evCh)
	sess.EnsureEventPump()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case ev := <-evCh:
			if ev == nil {
				continue
			}
			if err := stream.Send(runtimeEventToProto(sid, ev)); err != nil {
				return err
			}
			IncEventsTotal()
		}
	}
}

func runtimeEventToProto(sessionID string, ev *runtime.Event) *proto.DebugEvent {
	return &proto.DebugEvent{
		TimestampNs: int64(ev.TimestampNs), //nolint:gosec // G115: proto uses int64 for wire format
		SessionId:   sessionID,
		EventType:   proto.EventType(ev.EventType), //nolint:gosec // G115: proto enum matches runtime
		Pid:         ev.PID,
		Tgid:        ev.Tgid,
		Cpu:         ev.CPU,
		ProbeId:     strconv.FormatUint(uint64(ev.ProbeID), 10),
		Payload:     ev.Payload,
	}
}

// ListSessions returns all active session ids.
func (s *debuggerServer) ListSessions(_ context.Context, _ *proto.ListSessionsRequest) (*proto.ListSessionsResponse, error) {
	ids := s.sessions.List()
	return &proto.ListSessionsResponse{SessionIds: ids}, nil
}

// CloseSession removes the session and releases resources.
func (s *debuggerServer) CloseSession(_ context.Context, req *proto.CloseSessionRequest) (*proto.CloseSessionResponse, error) {
	sid := req.GetSessionId()
	DebugLogf("close_session session_id=%s", sid)
	s.sessions.Close(sid)
	SetSessionsActive(len(s.sessions.List()))
	if s.cfg != nil {
		if s.cfg.rateLimiter != nil {
			s.cfg.rateLimiter.RemoveSession(sid)
		}
		if s.cfg.quota != nil {
			s.cfg.quota.RemoveSession(sid)
		}
	}
	return &proto.CloseSessionResponse{Ok: true}, nil
}

// ConnectSession creates or reuses a session; for MCP backend.
func (s *debuggerServer) ConnectSession(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		sessionID = generateSessionID()
	}
	_, err := s.sessions.GetOrCreate(ctx, sessionID)
	if err != nil {
		return "", err
	}
	DebugLogf("mcp.ConnectSession session_id=%s", sessionID)
	return sessionID, nil
}

// ExecuteCommand runs one command in the session; for MCP backend (no rate limit/quota from MCP).
func (s *debuggerServer) ExecuteCommand(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error) {
	sess := s.sessions.Get(sessionID)
	if sess == nil {
		return errResponse("session not found"), nil
	}
	line := strings.TrimSpace(commandLine)
	resp, err := s.exec.execute(ctx, sess, line)
	logExecuteDebug("mcp.Execute", sessionID, line, resp, err)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListSessionsBackend returns session ids; for MCP backend.
func (s *debuggerServer) ListSessionsBackend(_ context.Context) []string {
	return s.sessions.List()
}

// ListBreakpointsBackend returns formatted breakpoints for the session; for MCP backend.
func (s *debuggerServer) ListBreakpointsBackend(_ context.Context, sessionID string) (string, error) {
	sess := s.sessions.Get(sessionID)
	if sess == nil {
		return "", nil
	}
	list := sess.ListBreakpoints()
	var lines []string
	for _, bp := range list {
		lines = append(lines, bp.ID+"  "+bp.Symbol)
	}
	return strings.Join(lines, "\n"), nil
}

// ListHooksBackend returns formatted hooks for the session; for MCP backend.
func (s *debuggerServer) ListHooksBackend(_ context.Context, sessionID string) (string, error) {
	sess := s.sessions.Get(sessionID)
	if sess == nil {
		return "", nil
	}
	list := sess.ListHooks()
	var lines []string
	for _, h := range list {
		lines = append(lines, h.ID+"  "+h.AttachPoint)
	}
	return strings.Join(lines, "\n"), nil
}

// checkQuota returns an error message if the command would exceed session quota.
func checkQuota(q *SessionQuota, sessionID, line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	verb := strings.ToLower(parts[0])
	switch verb {
	case "break", "b", "tbreak":
		if !q.AllowBreak(sessionID) {
			return "quota: max breakpoints reached"
		}
		if !q.AllowHook(sessionID) {
			q.RemoveBreak(sessionID)
			return "quota: max hooks reached"
		}
	case "trace", "t":
		if !q.AllowTrace(sessionID) {
			return "quota: max traces reached"
		}
	case "hook":
		if len(parts) < 2 {
			return ""
		}
		sub := strings.ToLower(parts[1])
		if sub != "attach" {
			return ""
		}
		if !q.AllowHook(sessionID) {
			return "quota: max hooks reached"
		}
	}
	return ""
}

// rollbackBreakTbreakQuota reverses pre-reserved break+hook quota when execute fails after checkQuota.
func rollbackBreakTbreakQuota(q *SessionQuota, sessionID, line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}
	switch strings.ToLower(parts[0]) {
	case "break", "b", "tbreak":
		q.RemoveBreak(sessionID)
		q.RemoveHook(sessionID)
	}
}
