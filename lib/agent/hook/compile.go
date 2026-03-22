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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// CompileResult holds the path to the compiled .o and cleanup to remove temp dir.
// ParsedAttach is nil for CompileRaw (attach point is supplied separately).
type CompileResult struct {
	ObjectPath   string
	ParsedAttach *ParsedAttach
	Cleanup      func() // call when hook is detached to remove temp dir
}

// MaxRawSourceLen is the maximum C source size for CompileRaw and hook attach --file.
const MaxRawSourceLen = 256 * 1024

// CompileRaw compiles a full C source file to BPF .o (CO-RE flags).
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
