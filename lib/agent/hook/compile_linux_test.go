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
}
