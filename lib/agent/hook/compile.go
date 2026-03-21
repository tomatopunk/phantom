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
func Compile(ctx context.Context, snippet, attachPoint, includeDir string) (CompileResult, error) {
	_, symbol, err := ParseAttachPoint(attachPoint)
	if err != nil {
		return CompileResult{}, err
	}
	// Sandbox: limit snippet size (e.g. 8KB).
	const maxSnippetLen = 8192
	if len(snippet) > maxSnippetLen {
		return CompileResult{}, fmt.Errorf("snippet too long")
	}
	prologue := PrologueC(symbol)
	tpl := strings.Replace(string(hookTemplate), placeholderPrologue, prologue, 1)
	tpl = strings.Replace(tpl, placeholderSnippet, snippet, 1)
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
	args = append(args, "-c", srcPath, "-o", outPath)
	if includeDir != "" {
		args = append(args, "-I", includeDir)
	}
	const compileTimeout = 30 * time.Second
	compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()
	cmd := exec.CommandContext(compileCtx, "clang", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, fmt.Errorf("compile: %w\n%s", err, out)
	}
	cleanup := func() { os.RemoveAll(dir) }
	return CompileResult{ObjectPath: outPath, Symbol: symbol, Cleanup: cleanup}, nil
}

const maxRawSourceLen = 256 * 1024

// CompileRaw compiles a full C source file to BPF .o (CO-RE flags, same as hook template builds).
func CompileRaw(ctx context.Context, source, includeDir string) (CompileResult, error) {
	if len(source) > maxRawSourceLen {
		return CompileResult{}, fmt.Errorf("source too long (max %d bytes)", maxRawSourceLen)
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
	args = append(args, "-c", srcPath, "-o", outPath)
	if includeDir != "" {
		args = append(args, "-I", includeDir)
	}
	const compileTimeout = 30 * time.Second
	compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()
	cmd := exec.CommandContext(compileCtx, "clang", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return CompileResult{}, fmt.Errorf("compile: %w\n%s", err, out)
	}
	cleanup := func() { os.RemoveAll(dir) }
	return CompileResult{ObjectPath: outPath, Symbol: "", Cleanup: cleanup}, nil
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
