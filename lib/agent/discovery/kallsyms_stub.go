//go:build !linux

package discovery

import "fmt"

// ListKprobeSymbols is unavailable on this platform.
func ListKprobeSymbols(_ string, _ int) ([]string, error) {
	return nil, fmt.Errorf("kallsyms: not available on this platform")
}
