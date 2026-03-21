// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"fmt"
	"math"
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
		if x < 0 || x > math.MaxUint32 {
			return 0, fmt.Errorf("%s must be a non-negative integer", key)
		}
		return uint32(x), nil
	case int64:
		if x < 0 || x > int64(math.MaxUint32) {
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
