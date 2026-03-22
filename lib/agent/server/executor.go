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
	"unicode"

	"github.com/cilium/ebpf/btf"
	"github.com/tomatopunk/phantom/lib/agent/expression"
	"github.com/tomatopunk/phantom/lib/agent/probe"
	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
)

// commandExecutor parses command lines and drives session runtime + state.
type commandExecutor struct {
	hookIncludeDir string // path to bpf/include for C hook compile
	vmlinuxPath    string // optional: path to vmlinux for list disasm (Linux)
	btfSpec        *btf.Spec
	planner        *probe.Planner
	quota          *SessionQuota // optional: rollback hook slot on failed hook add; decrement on delete
}

func newCommandExecutor(
	hookIncludeDir, vmlinuxPath string,
	planner *probe.Planner,
	btfSpec *btf.Spec,
	quota *SessionQuota,
) *commandExecutor {
	if planner == nil {
		planner = probe.NewPlanner()
	}
	return &commandExecutor{hookIncludeDir: hookIncludeDir, vmlinuxPath: vmlinuxPath, btfSpec: btfSpec, planner: planner, quota: quota}
}

func (e *commandExecutor) executeBreak(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	return e.executeBreakOrTbreak(ctx, sess, args, "break", false)
}

func (e *commandExecutor) executeTbreak(ctx context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	return e.executeBreakOrTbreak(ctx, sess, args, "tbreak", true)
}

// executeBreakOrTbreak is shared logic for break and tbreak (dupl).
func (e *commandExecutor) executeBreakOrTbreak(
	ctx context.Context, sess *session.Session, args []string, cmdPrefix string, isTemp bool,
) (*proto.ExecuteResponse, error) {
	_ = ctx
	if len(args) < 1 {
		return errResponse(cmdPrefix + ": missing symbol"), nil
	}
	plan := e.planner.PlanBreak(args[0])
	rt, err := sess.EnsureRuntime()
	if err != nil {
		return errResponse(cmdPrefix + ": " + err.Error()), nil
	}
	if rt == nil {
		return errResponse(cmdPrefix + ": no kprobe object path configured"), nil
	}
	detach, err := rt.AttachKprobe(plan.Symbol)
	if err != nil {
		return errResponse(cmdPrefix + ": " + err.Error()), nil
	}
	id := sess.AddBreakpoint(plan.Symbol, detach, isTemp)
	sess.EnsureEventPump()
	msg := "breakpoint set at "
	if isTemp {
		msg = "temporary breakpoint set at "
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: msg + plan.Symbol + " (" + id + ")",
		Result: &proto.ExecuteResponse_Breakpoint{
			Breakpoint: &proto.BreakpointResult{
				BreakpointId: id,
				Symbol:       plan.Symbol,
				Enabled:      true,
			},
		},
	}, nil
}

func (*commandExecutor) executePrint(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("print: missing expression"), nil
	}
	expr := strings.TrimSpace(args[0])
	ev := sess.GetLastEvent()
	value := expression.Evaluate(ev, expr)
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "$" + expr + " = " + value,
		Result: &proto.ExecuteResponse_Print{
			Print: &proto.PrintResult{Expression: expr, Value: value},
		},
	}, nil
}

func (e *commandExecutor) executeTrace(_ context.Context, sess *session.Session, args []string) (*proto.ExecuteResponse, error) {
	if len(args) < 1 {
		return errResponse("trace: missing expression(s)"), nil
	}
	plan := e.planner.PlanTrace(args)
	id := sess.AddTrace(plan.Expressions, nil)
	if sess.Runtime() != nil {
		sess.EnsureEventPump()
	}
	return &proto.ExecuteResponse{
		Ok:     true,
		Output: "tracing " + strings.Join(plan.Expressions, ", ") + " (" + id + ")",
		Result: &proto.ExecuteResponse_Trace{
			Trace: &proto.TraceResult{TraceId: id, Expressions: plan.Expressions},
		},
	}, nil
}

func (*commandExecutor) executeContinue(ctx context.Context, sess *session.Session) (*proto.ExecuteResponse, error) {
	_ = ctx
	_ = sess
	return &proto.ExecuteResponse{Ok: true, Output: "continue"}, nil
}

// splitCommandLine splits the REPL line on whitespace, respecting "..." and '...' so
// e.g. --sec "pid>0" yields a sec value without quote characters (unlike strings.Fields).
func splitCommandLine(line string) []string {
	out := make([]string, 0, 8)
	var b strings.Builder
	rs := []rune(line)
	i := 0
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for i < len(rs) {
		for i < len(rs) && unicode.IsSpace(rs[i]) {
			i++
		}
		if i >= len(rs) {
			break
		}
		switch rs[i] {
		case '"':
			i++
			for i < len(rs) && rs[i] != '"' {
				if rs[i] == '\\' && i+1 < len(rs) {
					i++
					b.WriteRune(rs[i])
					i++
					continue
				}
				b.WriteRune(rs[i])
				i++
			}
			if i < len(rs) {
				i++
			}
			flush()
		case '\'':
			i++
			for i < len(rs) && rs[i] != '\'' {
				b.WriteRune(rs[i])
				i++
			}
			if i < len(rs) {
				i++
			}
			flush()
		default:
			for i < len(rs) && !unicode.IsSpace(rs[i]) {
				b.WriteRune(rs[i])
				i++
			}
			flush()
		}
	}
	return out
}

func errResponse(msg string) *proto.ExecuteResponse {
	return &proto.ExecuteResponse{Ok: false, ErrorMessage: msg}
}
