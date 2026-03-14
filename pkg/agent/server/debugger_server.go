package server

import (
	"context"
	"strconv"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/probe"
	"github.com/tomatopunk/phantom/pkg/agent/runtime"
	"github.com/tomatopunk/phantom/pkg/agent/session"
	"github.com/tomatopunk/phantom/pkg/api/proto"
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
}

// AuditLogger is implemented by AuditLog and NopAuditLog.
type AuditLogger interface {
	LogCommand(sessionID, commandLine string, ok bool, errMsg string)
}

// NewDebuggerServer returns a DebuggerService server that uses the given session manager.
func NewDebuggerServer(sessions *session.Manager) *debuggerServer {
	return &debuggerServer{
		sessions: sessions,
		exec:     newCommandExecutor("", "", probe.NewPlanner()),
		cfg:      nil,
	}
}

// NewDebuggerServerWithConfig returns a server with rate limit, quota, audit, hook include dir, and vmlinux path.
func NewDebuggerServerWithConfig(sessions *session.Manager, cfg *serverConfig) *debuggerServer {
	includeDir := ""
	vmlinuxPath := ""
	if cfg != nil {
		includeDir = cfg.bpfIncludeDir
		vmlinuxPath = cfg.vmlinuxPath
	}
	return &debuggerServer{
		sessions: sessions,
		exec:     newCommandExecutor(includeDir, vmlinuxPath, probe.NewPlanner()),
		cfg:      cfg,
	}
}

// Connect creates or reuses a session and returns its id.
func (s *debuggerServer) Connect(ctx context.Context, req *proto.ConnectRequest) (*proto.ConnectResponse, error) {
	sid := req.GetSessionId()
	if sid == "" {
		sid = generateSessionID()
	}
	_, err := s.sessions.GetOrCreate(ctx, sid)
	if err != nil {
		return nil, err
	}
	SetSessionsActive(len(s.sessions.List()))
	return &proto.ConnectResponse{SessionId: sid, Message: "connected"}, nil
}

// Execute runs one command line in the session and returns the result.
func (s *debuggerServer) Execute(ctx context.Context, req *proto.ExecuteRequest) (*proto.ExecuteResponse, error) {
	sid := req.GetSessionId()
	if sid == "" {
		return errResponse("missing session_id"), nil
	}
	sess := s.sessions.Get(sid)
	if sess == nil {
		return errResponse("session not found"), nil
	}
	if s.cfg != nil && s.cfg.rateLimiter != nil && !s.cfg.rateLimiter.Allow(sid) {
		return errResponse("rate limited"), nil
	}
	line := strings.TrimSpace(req.GetCommandLine())
	if s.cfg != nil && s.cfg.quota != nil {
		if errMsg := checkQuota(s.cfg.quota, sid, line); errMsg != "" {
			return errResponse(errMsg), nil
		}
	}
	resp, err := s.exec.execute(ctx, sess, line)
	if err != nil {
		return nil, err
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
	return sessionID, nil
}

// ExecuteCommand runs one command in the session; for MCP backend (no rate limit/quota from MCP).
func (s *debuggerServer) ExecuteCommand(ctx context.Context, sessionID, commandLine string) (*proto.ExecuteResponse, error) {
	sess := s.sessions.Get(sessionID)
	if sess == nil {
		return errResponse("session not found"), nil
	}
	return s.exec.execute(ctx, sess, strings.TrimSpace(commandLine))
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
	case "break", "b":
		if !q.AllowBreak(sessionID) {
			return "quota: max breakpoints reached"
		}
	case "trace", "t":
		if !q.AllowTrace(sessionID) {
			return "quota: max traces reached"
		}
	case "hook":
		if !q.AllowHook(sessionID) {
			return "quota: max hooks reached"
		}
	}
	return ""
}
