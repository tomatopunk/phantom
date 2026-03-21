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
