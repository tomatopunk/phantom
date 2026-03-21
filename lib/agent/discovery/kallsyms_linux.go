//go:build linux

package discovery

import (
	"bufio"
	"os"
	"strings"
)

// ListKprobeSymbols returns kernel text symbols from /proc/kallsyms (T/t), optional prefix filter.
func ListKprobeSymbols(prefix string, maxEntries int) ([]string, error) {
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	prefix = strings.TrimSpace(prefix)
	f, err := os.Open("/proc/kallsyms")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	seen := make(map[string]struct{})
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		typ := fields[1]
		if typ != "T" && typ != "t" {
			continue
		}
		name := fields[2]
		if idx := strings.Index(name, "\t"); idx >= 0 {
			name = name[:idx]
		}
		if i := strings.Index(name, "."); i > 0 && strings.HasPrefix(name[i:], ".cold") {
			name = name[:i]
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
		if len(out) >= maxEntries {
			break
		}
	}
	return out, sc.Err()
}
