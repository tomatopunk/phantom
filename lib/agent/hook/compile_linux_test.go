//go:build linux

package hook

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompile_COReHook(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed")
	}
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	root := filepath.Join("..", "..", "..", "bpf", "include")
	abs, err := filepath.Abs(root)
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
