package mcp

import (
	"fmt"
	"strconv"
)

// uint32FromArgs reads an optional unsigned int from JSON-RPC arguments (numbers may be float64).
func uint32FromArgs(args map[string]any, key string, def uint32) (uint32, error) {
	v, ok := args[key]
	if !ok {
		return def, nil
	}
	switch x := v.(type) {
	case float64:
		if x < 0 || x > float64(^uint32(0)) {
			return 0, fmt.Errorf("%s must be a non-negative integer", key)
		}
		return uint32(x), nil
	case int:
		if x < 0 {
			return 0, fmt.Errorf("%s must be a non-negative integer", key)
		}
		return uint32(x), nil
	case int64:
		if x < 0 {
			return 0, fmt.Errorf("%s must be a non-negative integer", key)
		}
		return uint32(x), nil
	case string:
		if x == "" {
			return def, nil
		}
		n, err := strconv.ParseUint(x, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s: invalid integer", key)
		}
		return uint32(n), nil
	default:
		return 0, fmt.Errorf("%s: unsupported type", key)
	}
}
