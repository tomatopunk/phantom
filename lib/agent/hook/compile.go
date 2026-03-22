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

package hook

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed embed/hook.c
var hookTemplate []byte

const (
	placeholderPrologue = "{{PROLOGUE}}"
	placeholderSnippet  = "{{SNIPPET}}"
	placeholderSecLine  = "{{SEC_LINE}}"
	placeholderCtxDecl  = "{{CTX_DECL}}"
	placeholderArgInit  = "{{ARG_INIT}}"
	ctxDeclPtRegs       = `struct pt_regs *ctx`
)

// CompileResult holds the path to the compiled .o and cleanup to remove temp dir.
// ParsedAttach is set by Compile (template hook) for attach routing; nil for CompileRaw.
type CompileResult struct {
	ObjectPath   string
	ParsedAttach *ParsedAttach
	Cleanup      func() // call when hook is detached to remove temp dir
}

func hookVariantForPA(pa *ParsedAttach) (secLine, ctxDecl, argInit string, err error) {
	switch pa.Kind {
	case AttachKprobe:
		return `SEC("kprobe")`, ctxDeclPtRegs, ptRegsArgInit(), nil
	case AttachUprobe:
		return `SEC("uprobe")`, ctxDeclPtRegs, ptRegsArgInit(), nil
	case AttachUretprobe:
		return `SEC("uretprobe")`, ctxDeclPtRegs, ptRegsArgInit(), nil
	case AttachTracepoint:
		return fmt.Sprintf(`SEC("tracepoint/%s/%s")`, pa.TraceGroup, pa.TraceEvent), `void *ctx`, zeroArgInit(), nil
	default:
		return "", "", "", fmt.Errorf("internal: unknown attach kind")
	}
}

func ptRegsArgInit() string {
	// PT_REGS_PARM6 is missing or not CO-RE-relocatable in some libbpf/kernel pairs,
	// which breaks loading with "unsatisfied program reference". arg0–arg4 cover most
	// kprobes; users needing a 6th arg should use hook attach with custom C.
	return "\tlong arg0 = PT_REGS_PARM1(ctx);\n" +
		"\tlong arg1 = PT_REGS_PARM2(ctx);\n" +
		"\tlong arg2 = PT_REGS_PARM3(ctx);\n" +
		"\tlong arg3 = PT_REGS_PARM4(ctx);\n" +
		"\tlong arg4 = PT_REGS_PARM5(ctx);\n" +
		"\tlong arg5 = 0;\n" +
		"\tlong ret = 0;\n"
}

func zeroArgInit() string {
	return "\tlong arg0 = 0, arg1 = 0, arg2 = 0, arg3 = 0, arg4 = 0, arg5 = 0;\n\tlong ret = 0;\n"
}

// BuildTemplateSource returns the full C source that Compile would pass to clang (template + snippet), without compiling.
func BuildTemplateSource(snippet, attachPoint string) (string, error) {
	pa, err := ParseFullAttachPoint(attachPoint)
	if err != nil {
		return "", err
	}
	secLine, ctxDecl, argInit, err := hookVariantForPA(pa)
	if err != nil {
		return "", err
	}
	// Sandbox: limit snippet size (e.g. 8KB).
	const maxSnippetLen = 8192
	if len(snippet) > maxSnippetLen {
		return "", fmt.Errorf("snippet too long")
	}
	prologue := PrologueC(AttachPrologueKey(attachPoint))
	tpl := string(hookTemplate)
	tpl = strings.Replace(tpl, placeholderSecLine, secLine, 1)
	tpl = strings.Replace(tpl, placeholderCtxDecl, ctxDecl, 1)
	tpl = strings.Replace(tpl, placeholderArgInit, argInit, 1)
	tpl = strings.Replace(tpl, placeholderPrologue, prologue, 1)
	tpl = strings.Replace(tpl, placeholderSnippet, snippet, 1)
	return tpl, nil
}

// Compile compiles the C snippet into an eBPF .o file with a timeout and size limit.
func Compile(ctx context.Context, snippet, attachPoint, includeDir string) (CompileResult, error) {
	tpl, err := BuildTemplateSource(snippet, attachPoint)
	if err != nil {
		return CompileResult{}, err
	}
	pa, err := ParseFullAttachPoint(attachPoint)
	if err != nil {
		return CompileResult{}, err
	}
	dir, err := os.MkdirTemp("", "phantom-hook-")
	if err != nil {
		return CompileResult{}, err
	}
	srcPath := filepath.Join(dir, "hook.c")
	const srcMode = 0o600
	if err := os.WriteFile(srcPath, []byte(tpl), srcMode); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, err
	}
	outPath := filepath.Join(dir, "hook.o")
	args := []string{
		"-target", "bpf",
		"-O2",
		"-g", // BTF for CO-RE relocations
	}
	args = append(args, bpfTargetArchDefines()...)
	args = append(args, hostClangBPFForceIncludes()...)
	args = append(args, "-c", srcPath, "-o", outPath)
	if includeDir != "" {
		args = append(args, "-I", includeDir)
	}
	for _, inc := range hostClangBPFExtraIncludes() {
		args = append(args, "-I", inc)
	}
	const compileTimeout = 30 * time.Second
	compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()
	cmd := exec.CommandContext(compileCtx, "clang", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, &CompileFailed{Stderr: out, Err: err}
	}
	cleanup := func() { os.RemoveAll(dir) }
	paCopy := *pa
	return CompileResult{ObjectPath: outPath, ParsedAttach: &paCopy, Cleanup: cleanup}, nil
}

// MaxRawSourceLen is the maximum C source size for CompileRaw and hook attach --file.
const MaxRawSourceLen = 256 * 1024

// CompileRaw compiles a full C source file to BPF .o (CO-RE flags, same as hook template builds).
func CompileRaw(ctx context.Context, source, includeDir string) (CompileResult, error) {
	if len(source) > MaxRawSourceLen {
		return CompileResult{}, fmt.Errorf("source too long (max %d bytes)", MaxRawSourceLen)
	}
	dir, err := os.MkdirTemp("", "phantom-raw-")
	if err != nil {
		return CompileResult{}, err
	}
	srcPath := filepath.Join(dir, "program.c")
	const srcMode = 0o600
	if err := os.WriteFile(srcPath, []byte(source), srcMode); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, err
	}
	outPath := filepath.Join(dir, "program.o")
	args := []string{
		"-target", "bpf",
		"-O2",
		"-g",
	}
	args = append(args, bpfTargetArchDefines()...)
	args = append(args, hostClangBPFForceIncludes()...)
	args = append(args, "-c", srcPath, "-o", outPath)
	if includeDir != "" {
		args = append(args, "-I", includeDir)
	}
	for _, inc := range hostClangBPFExtraIncludes() {
		args = append(args, "-I", inc)
	}
	const compileTimeout = 30 * time.Second
	compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()
	cmd := exec.CommandContext(compileCtx, "clang", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, &CompileFailed{Stderr: out, Err: err}
	}
	cleanup := func() { os.RemoveAll(dir) }
	return CompileResult{ObjectPath: outPath, ParsedAttach: nil, Cleanup: cleanup}, nil
}

// bpfTargetArchDefines sets PT_REGS_* and friends for the BPF target (agent arch).
func bpfTargetArchDefines() []string {
	switch runtime.GOARCH {
	case "amd64":
		return []string{"-D__TARGET_ARCH_x86=1"}
	case "arm64":
		return []string{"-D__TARGET_ARCH_arm64=1"}
	case "ppc64le":
		return []string{"-D__TARGET_ARCH_powerpc=1"}
	case "s390x":
		return []string{"-D__TARGET_ARCH_s390=1"}
	case "riscv64":
		return []string{"-D__TARGET_ARCH_riscv=1"}
	default:
		return []string{"-D__TARGET_ARCH_x86=1"}
	}
}
