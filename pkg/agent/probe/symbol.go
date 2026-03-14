package probe

import (
	"debug/elf"
	"fmt"
	"os"
)

// ResolveUserSymbol returns the file offset for the given symbol in the binary at path.
func ResolveUserSymbol(binaryPath, symbolName string) (uint64, error) {
	f, err := os.Open(binaryPath)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", binaryPath, err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		return 0, fmt.Errorf("elf %s: %w", binaryPath, err)
	}
	syms, err := ef.Symbols()
	if err != nil {
		return 0, fmt.Errorf("symbols %s: %w", binaryPath, err)
	}
	for _, s := range syms {
		if s.Name == symbolName {
			return s.Value, nil
		}
	}
	return 0, fmt.Errorf("symbol %q not found in %s", symbolName, binaryPath)
}
