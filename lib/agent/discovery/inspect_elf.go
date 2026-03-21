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

package discovery

import (
	"bytes"
	"debug/elf"
	"fmt"
	"sort"
)

// InspectELFSections returns sorted section names from ELF bytes.
func InspectELFSections(elfData []byte) ([]string, error) {
	if len(elfData) == 0 {
		return nil, fmt.Errorf("empty ELF data")
	}
	f, err := elf.NewFile(bytes.NewReader(elfData))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var names []string
	for _, s := range f.Sections {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}
