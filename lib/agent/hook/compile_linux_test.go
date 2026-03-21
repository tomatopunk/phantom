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

//go:build linux

package hook

import (
	"context"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func TestCompile_COReHook(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed")
	}
	if goruntime.GOOS != "linux" {
		t.Skip("linux only")
	}
	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	// Repo: src/agent/bpf/include/{common.h,...} — same tree as Makefile BPF_INCLUDE (not repo-root/bpf/include).
	includeDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "src", "agent", "bpf", "include")
	abs, err := filepath.Abs(includeDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	res, err := Compile(ctx, "return 0;", "kprobe:do_sys_open", abs)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Cleanup()
	if res.ObjectPath == "" {
		t.Fatal("empty object path")
	}

	resTP, err := Compile(ctx, "return 0;", "tracepoint:sched:sched_process_fork", abs)
	if err != nil {
		t.Fatal(err)
	}
	defer resTP.Cleanup()
	if resTP.ParsedAttach == nil || resTP.ParsedAttach.Kind != AttachTracepoint {
		t.Fatalf("tracepoint compile: ParsedAttach=%v", resTP.ParsedAttach)
	}

	resUR, err := Compile(ctx, "return 0;", "uretprobe:/bin/true:main", abs)
	if err != nil {
		t.Fatal(err)
	}
	defer resUR.Cleanup()
	if resUR.ParsedAttach == nil || resUR.ParsedAttach.Kind != AttachUretprobe {
		t.Fatalf("uretprobe compile: ParsedAttach=%v", resUR.ParsedAttach)
	}
}
