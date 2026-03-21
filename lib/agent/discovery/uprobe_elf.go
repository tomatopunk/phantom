package discovery

import (
	"debug/elf"
	"sort"
	"strings"
)

// ListUprobeSymbols lists STT_FUNC symbols from .dynsym and .symtab (best-effort).
func ListUprobeSymbols(binaryPath, prefix string, maxEntries int) ([]string, error) {
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	prefix = strings.TrimSpace(prefix)
	f, err := elf.Open(binaryPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	seen := make(map[string]struct{})
	var names []string
	addSyms := func(syms []elf.Symbol) {
		for _, s := range syms {
			if elf.ST_TYPE(s.Info) != elf.STT_FUNC {
				continue
			}
			if s.Name == "" {
				continue
			}
			if prefix != "" && !strings.HasPrefix(s.Name, prefix) {
				continue
			}
			if _, ok := seen[s.Name]; ok {
				continue
			}
			seen[s.Name] = struct{}{}
			names = append(names, s.Name)
		}
	}
	if syms, err := f.DynamicSymbols(); err == nil {
		addSyms(syms)
	}
	if syms, err := f.Symbols(); err == nil {
		addSyms(syms)
	}
	sort.Strings(names)
	if len(names) > maxEntries {
		names = names[:maxEntries]
	}
	return names, nil
}
