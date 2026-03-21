//go:build !linux

package discovery

// ListTracepoints is unavailable on this platform.
func ListTracepoints(_ string, _ int) ([]string, error) {
	return nil, nil
}
