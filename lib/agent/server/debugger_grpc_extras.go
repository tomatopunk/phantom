package server

import (
	"context"

	"github.com/tomatopunk/phantom/lib/agent/discovery"
	"github.com/tomatopunk/phantom/lib/proto"
)

func (s *debuggerServer) CompileAndAttach(ctx context.Context, req *proto.CompileAndAttachRequest) (*proto.CompileAndAttachResponse, error) {
	sid := req.GetSessionId()
	if sid == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "missing session_id"}, nil
	}
	sess := s.sessions.Get(sid)
	if sess == nil {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "session not found"}, nil
	}
	if s.cfg != nil && s.cfg.rateLimiter != nil && !s.cfg.rateLimiter.Allow(sid) {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "rate limited"}, nil
	}
	if s.cfg != nil && s.cfg.quota != nil && !s.cfg.quota.AllowHook(sid) {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "quota: max hooks reached"}, nil
	}
	ok := false
	defer func() {
		if !ok && s.cfg != nil && s.cfg.quota != nil {
			s.cfg.quota.RemoveHook(sid)
		}
	}()
	resp, err := s.exec.compileAndAttach(ctx, sess, req)
	if err != nil {
		return nil, err
	}
	if resp.GetOk() {
		ok = true
	}
	return resp, nil
}

func (s *debuggerServer) ListTracepoints(_ context.Context, req *proto.ListTracepointsRequest) (*proto.ListTracepointsResponse, error) {
	names, err := discovery.ListTracepoints(req.GetPrefix(), int(req.GetMaxEntries()))
	if err != nil {
		return nil, err
	}
	return &proto.ListTracepointsResponse{Names: names}, nil
}

func (s *debuggerServer) ListKprobeSymbols(_ context.Context, req *proto.ListKprobeSymbolsRequest) (*proto.ListKprobeSymbolsResponse, error) {
	syms, err := discovery.ListKprobeSymbols(req.GetPrefix(), int(req.GetMaxEntries()))
	if err != nil {
		return nil, err
	}
	return &proto.ListKprobeSymbolsResponse{Symbols: syms}, nil
}

func (s *debuggerServer) ListUprobeSymbols(_ context.Context, req *proto.ListUprobeSymbolsRequest) (*proto.ListUprobeSymbolsResponse, error) {
	path := req.GetBinaryPath()
	if path == "" {
		return &proto.ListUprobeSymbolsResponse{}, nil
	}
	syms, err := discovery.ListUprobeSymbols(path, req.GetPrefix(), int(req.GetMaxEntries()))
	if err != nil {
		return nil, err
	}
	return &proto.ListUprobeSymbolsResponse{Symbols: syms}, nil
}

func (s *debuggerServer) InspectELF(_ context.Context, req *proto.InspectELFRequest) (*proto.InspectELFResponse, error) {
	secs, err := discovery.InspectELFSections(req.GetElfData())
	if err != nil {
		return &proto.InspectELFResponse{}, err
	}
	return &proto.InspectELFResponse{SectionNames: secs}, nil
}

func (s *debuggerServer) GetHostMetrics(ctx context.Context, _ *proto.GetHostMetricsRequest) (*proto.GetHostMetricsResponse, error) {
	return collectHostMetrics(ctx), nil
}

func (s *debuggerServer) GetTaskTree(ctx context.Context, req *proto.GetTaskTreeRequest) (*proto.GetTaskTreeResponse, error) {
	return collectTaskTree(ctx, req.GetTgid()), nil
}
