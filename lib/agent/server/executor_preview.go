// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/hook"
	"github.com/tomatopunk/phantom/lib/proto"
)

// previewHookTemplate expands a template hook (--sec or --code) to full C and optionally compiles (no attach).
func (e *commandExecutor) previewHookTemplate(ctx context.Context, attachPoint, secExpr, codeSnippet string) *proto.PreviewHookTemplateResponse {
	secExpr = strings.TrimSpace(secExpr)
	codeSnippet = strings.TrimSpace(codeSnippet)
	if secExpr != "" && codeSnippet != "" {
		return &proto.PreviewHookTemplateResponse{Ok: false, ErrorMessage: "use either sec_expression or code_snippet, not both"}
	}
	if secExpr == "" && codeSnippet == "" {
		return &proto.PreviewHookTemplateResponse{Ok: false, ErrorMessage: "missing sec_expression or code_snippet"}
	}
	attachPoint = strings.TrimSpace(attachPoint)
	if attachPoint == "" {
		return &proto.PreviewHookTemplateResponse{Ok: false, ErrorMessage: "missing attach_point"}
	}
	var snippet string
	var err error
	if secExpr != "" {
		snippet, err = hook.SecToSnippet(secExpr, attachPoint)
	} else {
		snippet = codeSnippet
	}
	if err != nil {
		return &proto.PreviewHookTemplateResponse{Ok: false, ErrorMessage: err.Error()}
	}
	src, err := hook.BuildTemplateSource(snippet, attachPoint)
	if err != nil {
		return &proto.PreviewHookTemplateResponse{Ok: false, ErrorMessage: err.Error()}
	}
	out := &proto.PreviewHookTemplateResponse{
		Ok:               true,
		GeneratedSourceC: src,
		CompileAttempted: false,
	}
	if e.hookIncludeDir == "" {
		return out
	}
	out.CompileAttempted = true
	cr, err := hook.Compile(ctx, snippet, attachPoint, e.hookIncludeDir)
	if err != nil {
		out.CompileOk = false
		if cf, ok := hook.AsCompileFailed(err); ok {
			stderr := string(cf.Stderr)
			out.CompilerOutput = stderr
			out.Diagnostics = hook.ParseClangDiagnostics(stderr)
		} else {
			out.CompilerOutput = err.Error()
		}
		return out
	}
	out.CompileOk = true
	if cr.Cleanup != nil {
		cr.Cleanup()
	}
	return out
}
