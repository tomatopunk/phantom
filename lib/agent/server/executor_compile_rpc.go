package server

import (
	"context"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/hook"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

func (e *commandExecutor) compileAndAttach(ctx context.Context, sess *session.Session, req *proto.CompileAndAttachRequest) (*proto.CompileAndAttachResponse, error) {
	if strings.TrimSpace(req.GetSource()) == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "empty source"}, nil
	}
	pa, err := hook.ParseFullAttachPoint(req.GetAttach())
	if err != nil {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}, nil
	}
	if e.hookIncludeDir == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "no bpf include dir configured"}, nil
	}
	cr, err := hook.CompileRaw(ctx, req.GetSource(), e.hookIncludeDir)
	if err != nil {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}, nil
	}
	detach, rd, err := hook.AttachProbeFromObject(cr.ObjectPath, pa, strings.TrimSpace(req.GetProgramName()), cr.Cleanup)
	if err != nil {
		if cr.Cleanup != nil {
			cr.Cleanup()
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}, nil
	}
	id := sess.AddHook(req.GetAttach(), detach, rd, 0)
	return &proto.CompileAndAttachResponse{Ok: true, HookId: id, AttachPoint: req.GetAttach()}, nil
}
