package server

import (
	"context"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/hook"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

func firstLineOf(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// tryCompileAttachHook compiles full C on the agent (same as hook attach), then attaches. Compile must succeed before attach.
func (e *commandExecutor) tryCompileAttachHook(ctx context.Context, sess *session.Session, source, attach, programName string, limit int) *proto.CompileAndAttachResponse {
	source = strings.TrimSpace(source)
	if source == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "empty source"}
	}
	pa, err := hook.ParseFullAttachPoint(attach)
	if err != nil {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}
	}
	if e.hookIncludeDir == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "no bpf include dir configured"}
	}
	cr, err := hook.CompileRaw(ctx, source, e.hookIncludeDir)
	if err != nil {
		if cf, ok := hook.AsCompileFailed(err); ok {
			out := string(cf.Stderr)
			return &proto.CompileAndAttachResponse{
				Ok:             false,
				ErrorMessage:   firstLineOf(out),
				CompilerOutput: out,
				Diagnostics:    hook.ParseClangDiagnostics(out),
				AttachPoint:    attach,
			}
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}
	}
	detach, rd, err := hook.AttachProbeFromObject(cr.ObjectPath, pa, strings.TrimSpace(programName), cr.Cleanup)
	if err != nil {
		if cr.Cleanup != nil {
			cr.Cleanup()
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}
	}
	id := sess.AddHook(attach, detach, rd, limit, &session.HookOpts{Note: "CompileAndAttach"})
	return &proto.CompileAndAttachResponse{Ok: true, HookId: id, AttachPoint: attach}
}

func (e *commandExecutor) compileAndAttach(ctx context.Context, sess *session.Session, req *proto.CompileAndAttachRequest) (*proto.CompileAndAttachResponse, error) {
	return e.tryCompileAttachHook(ctx, sess, req.GetSource(), req.GetAttach(), req.GetProgramName(), 0), nil
}
