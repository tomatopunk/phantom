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

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

// readKernelStack tries to read /proc/<tid>/stack for the thread from the last event.
// Prefers pid (thread id) then falls back to tgid. Returns stack text or an error message.
func readKernelStack(ev *runtime.Event) string {
	if ev == nil {
		return "bt: no event yet (hit a breakpoint first)"
	}
	tid := ev.PID
	if tid == 0 {
		tid = ev.Tgid
	}
	if tid == 0 {
		return "bt: no pid/tgid in event"
	}
	// Build /proc/<tid>/stack without literal path separator in Join (gocritic filepathJoin).
	path := string(filepath.Separator) + filepath.Join("proc", strconv.FormatUint(uint64(tid), 10), "stack")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("bt: thread %d not found (may have exited)", tid)
		}
		if os.IsPermission(err) {
			return fmt.Sprintf("bt: cannot read %s (permission denied; try root)", path)
		}
		return fmt.Sprintf("bt: %v", err)
	}
	return "bt:\n" + string(b)
}
