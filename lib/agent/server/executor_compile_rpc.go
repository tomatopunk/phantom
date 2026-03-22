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

// compileAttachFromSource compiles full C, derives probe_point from the ELF SEC of the chosen program, then attaches.
func (e *commandExecutor) compileAttachFromSource(
	ctx context.Context,
	sess *session.Session,
	source, programName string,
	limit int,
	note string,
) *proto.CompileAndAttachResponse {
	source = strings.TrimSpace(source)
	if source == "" {
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "empty source"}
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
			}
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}
	}
	pa, pickedProg, err := hook.ParseProbePointFromELF(cr.ObjectPath, strings.TrimSpace(programName))
	if err != nil {
		if cr.Cleanup != nil {
			cr.Cleanup()
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: err.Error()}
	}
	detach, rd, coll, err := hook.AttachProbeFromObject(cr.ObjectPath, pa, pickedProg, cr.Cleanup)
	if err != nil {
		if cr.Cleanup != nil {
			cr.Cleanup()
		}
		return &proto.CompileAndAttachResponse{Ok: false, ErrorMessage: "attach failed: " + err.Error()}
	}
	if note == "" {
		note = "CompileAndAttach"
	}
	probePoint := hook.FormatProbePoint(pa)
	id := sess.AddHook(probePoint, detach, rd, coll, limit, &session.HookOpts{Note: note})
	return &proto.CompileAndAttachResponse{Ok: true, HookId: id, ProbePoint: probePoint}
}

func (e *commandExecutor) compileAndAttach(
	ctx context.Context,
	sess *session.Session,
	req *proto.CompileAndAttachRequest,
) *proto.CompileAndAttachResponse {
	return e.compileAttachFromSource(ctx, sess, req.GetSource(), req.GetProgramName(), int(req.GetLimit()), "CompileAndAttach")
}

// validateCompileSource runs clang only (no attach, no quota).
func (e *commandExecutor) validateCompileSource(ctx context.Context, source string) *proto.ValidateCompileSourceResponse {
	source = strings.TrimSpace(source)
	if source == "" {
		return &proto.ValidateCompileSourceResponse{Ok: false, ErrorMessage: "empty source"}
	}
	if e.hookIncludeDir == "" {
		return &proto.ValidateCompileSourceResponse{Ok: false, ErrorMessage: "no bpf include dir configured"}
	}
	_, err := hook.CompileRaw(ctx, source, e.hookIncludeDir)
	if err != nil {
		if cf, ok := hook.AsCompileFailed(err); ok {
			out := string(cf.Stderr)
			return &proto.ValidateCompileSourceResponse{
				Ok:             false,
				ErrorMessage:   firstLineOf(out),
				CompilerOutput: out,
				Diagnostics:    hook.ParseClangDiagnostics(out),
			}
		}
		return &proto.ValidateCompileSourceResponse{Ok: false, ErrorMessage: err.Error()}
	}
	return &proto.ValidateCompileSourceResponse{Ok: true}
}
