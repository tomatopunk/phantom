//go:build linux

package server

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// listSymbolKernel reads /proc/kallsyms and returns the line for the given symbol
// plus a few surrounding lines. Returns empty string and nil if symbol not found.
func listSymbolKernel(symbol string) (string, error) {
	f, err := os.Open("/proc/kallsyms")
	if err != nil {
		return "", fmt.Errorf("cannot read kernel symbol table: %w", err)
	}
	defer f.Close()

	var lines []string
	var matchIdx int = -1
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
	return strings.Join(lines[start:end], "\n"), nil
}
