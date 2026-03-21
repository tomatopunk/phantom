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

//go:build linux

package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func tracepointRoots() []string {
	return []string{
		"/sys/kernel/tracing/events",
		"/sys/kernel/debug/tracing/events",
	}
}

// ListTracepoints returns names as "subsystem/event" from tracefs.
func ListTracepoints(prefix string, maxEntries int) ([]string, error) {
	if maxEntries <= 0 {
		maxEntries = 50000
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	var out []string
	for _, root := range tracepointRoots() {
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil || rel == "." {
				return nil
			}
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) != 2 {
				return nil
			}
			sub, ev := parts[0], parts[1]
			if sub == "" || ev == "" || strings.HasPrefix(ev, ".") {
				return nil
			}
			enable := filepath.Join(path, "enable")
			if st2, err := os.Stat(enable); err != nil || st2.IsDir() {
				return nil
			}
			name := sub + "/" + ev
			if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefix) {
				return nil
			}
			out = append(out, name)
			if len(out) >= maxEntries {
				return fs.SkipAll
			}
			return nil
		})
		if len(out) > 0 {
			break
		}
	}
	return out, nil
}
