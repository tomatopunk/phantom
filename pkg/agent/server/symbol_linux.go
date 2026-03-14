//go:build linux

package server

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// symbolAddressFromKallsyms returns the address of the given symbol from /proc/kallsyms, or error if not found.
func symbolAddressFromKallsyms(symbol string) (uint64, error) {
	f, err := os.Open("/proc/kallsyms")
	if err != nil {
		return 0, fmt.Errorf("cannot read kernel symbol table: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[2] == symbol {
			addr, err := strconv.ParseUint(fields[0], 16, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid address for %s: %w", symbol, err)
			}
			return addr, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("symbol %s not found", symbol)
}

// disasmSymbol runs objdump -d on vmlinux for the given address range and returns the disassembly (best-effort).
// Returns empty string on any error (e.g. objdump not found, file missing); caller keeps kallsyms output only.
func disasmSymbol(vmlinuxPath string, addr uint64, size uint64) string {
	if size == 0 {
		size = 0x80 // default window for kernel symbol
	}
	if _, err := os.Stat(vmlinuxPath); err != nil {
		return ""
	}
	objdump, err := exec.LookPath("objdump")
	if err != nil {
		objdump, err = exec.LookPath("llvm-objdump")
		if err != nil {
			return ""
		}
	}
	cmd := exec.Command(objdump, "-d",
		"--start-address="+strconv.FormatUint(addr, 16),
		"--stop-address="+strconv.FormatUint(addr+size, 16),
		vmlinuxPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// listSymbolKernel reads /proc/kallsyms and returns the line for the given symbol
// plus a few surrounding lines. Returns empty string and nil if symbol not found.
func listSymbolKernel(symbol string) (string, error) {
	f, err := os.Open("/proc/kallsyms")
	if err != nil {
		return "", fmt.Errorf("cannot read kernel symbol table: %w", err)
	}
	defer f.Close()

	var lines []string
	var matchIdx = -1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		// Format: address type name [module]
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == symbol {
			matchIdx = len(lines) - 1
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if matchIdx < 0 {
		return "", nil
	}
	// Show a few lines before and after (e.g. 2 before, 2 after)
	const context = 2
	start := matchIdx - context
	if start < 0 {
		start = 0
	}
	end := matchIdx + context + 1
	if end > len(lines) {
		end = len(lines)
	}
	kallsymsOut := strings.Join(lines[start:end], "\n")
	return kallsymsOut, nil
}

// listSymbolKernelAndDisasm returns kallsyms context for the symbol; if vmlinuxPath is set, appends disassembly.
func listSymbolKernelAndDisasm(symbol, vmlinuxPath string) (string, error) {
	out, err := listSymbolKernel(symbol)
	if err != nil || out == "" {
		return out, err
	}
	if vmlinuxPath == "" {
		return out, nil
	}
	addr, err := symbolAddressFromKallsyms(symbol)
	if err != nil {
		return out, nil // keep kallsyms only on address lookup failure
	}
	disasm := disasmSymbol(vmlinuxPath, addr, 0x80)
	if disasm != "" {
		out += "\n\nDisassembly:\n" + disasm
	}
	return out, nil
}
