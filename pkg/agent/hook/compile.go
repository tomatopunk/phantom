package hook

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CompileResult holds the path to the compiled .o, symbol, and cleanup to remove temp dir.
type CompileResult struct {
	ObjectPath string
	Symbol     string
	Cleanup    func() // call when hook is detached to remove temp dir
}

// AttachPoint describes where to attach (e.g. "kprobe:do_sys_open" -> symbol "do_sys_open").
func ParseAttachPoint(attachPoint string) (typ, symbol string, err error) {
	parts := strings.SplitN(attachPoint, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("attach point must be type:symbol (e.g. kprobe:do_sys_open)")
	}
	typ = strings.TrimSpace(strings.ToLower(parts[0]))
	symbol = strings.TrimSpace(parts[1])
	if symbol == "" {
		return "", "", fmt.Errorf("symbol is empty")
	}
	if typ != "kprobe" {
		return "", "", fmt.Errorf("only kprobe supported for C hook")
	}
	return typ, symbol, nil
}

// Compile compiles the C snippet into an eBPF .o file with a timeout and size limit.
func Compile(ctx context.Context, snippet string, attachPoint string, includeDir string) (CompileResult, error) {
	_, symbol, err := ParseAttachPoint(attachPoint)
	if err != nil {
		return CompileResult{}, err
	}
	// Sandbox: limit snippet size (e.g. 8KB).
	if len(snippet) > 8192 {
		return CompileResult{}, fmt.Errorf("snippet too long")
	}
	// Build minimal kprobe C: user snippet runs with ctx and ev; we submit ev to ringbuf.
	tpl := `
#define __BPF_TRACING__
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include "common.h"
struct { __uint(type, BPF_MAP_TYPE_RINGBUF); __uint(max_entries, 256*1024); } events SEC(".maps");
SEC("kprobe")
int hook_handler(struct pt_regs *ctx) {
	__u64 ts = bpf_ktime_get_ns();
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	struct event_header ev = {
		.timestamp_ns = ts,
		.pid = (__u32)(pid_tgid >> 32),
		.tgid = (__u32)pid_tgid,
		.event_type = PHANTOM_EVENT_TYPE_BREAK_HIT,
	};
	long arg0 = PT_REGS_PARM1(ctx);
	long arg1 = PT_REGS_PARM2(ctx);
	(void)arg0; (void)arg1;
	/* user snippet can use ctx, arg0, arg1, ev */
` + snippet + `
	bpf_ringbuf_output(&events, &ev, sizeof(ev), 0);
	return 0;
}
char _license[] SEC("license") = "GPL";
`
	dir, err := os.MkdirTemp("", "phantom-hook-")
	if err != nil {
		return CompileResult{}, err
	}
	srcPath := filepath.Join(dir, "hook.c")
	if err := os.WriteFile(srcPath, []byte(tpl), 0600); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, err
	}
	outPath := filepath.Join(dir, "hook.o")
	args := []string{"-target", "bpf", "-O2", "-c", srcPath, "-o", outPath}
	if includeDir != "" {
		args = append(args, "-I", includeDir)
	}
	compileCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(compileCtx, "clang", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, fmt.Errorf("compile: %w\n%s", err, out)
	}
	cleanup := func() { os.RemoveAll(dir) }
	return CompileResult{ObjectPath: outPath, Symbol: symbol, Cleanup: cleanup}, nil
}
