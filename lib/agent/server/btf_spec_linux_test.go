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
	"path/filepath"
	"strings"
	"testing"
)

func TestVmlinuxBTFSearchCandidates_orderAndDedup(t *testing.T) {
	user := "/opt/linux/vmlinux"
	got := vmlinuxBTFSearchCandidates(user, "6.6.0-test")
	if len(got) < 4 {
		t.Fatalf("want at least 4 candidates, got %d: %v", len(got), got)
	}
	if got[0] != user {
		t.Fatalf("want user path first, got %q", got[0])
	}
	seen := make(map[string]int)
	for _, p := range got {
		seen[p]++
		if seen[p] > 1 {
			t.Fatalf("duplicate path %q", p)
		}
	}
	wantSub := []string{
		filepath.Join("/boot", "vmlinux-6.6.0-test"),
		filepath.Join("/usr/lib/debug/boot", "vmlinux-6.6.0-test"),
		filepath.Join("/lib/modules", "6.6.0-test", "build", "vmlinux"),
	}
	for _, w := range wantSub {
		found := false
		for _, p := range got {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected path %q in %v", w, got)
		}
	}
}

func TestKernelRelease_nonemptyOnLinux(t *testing.T) {
	r := kernelRelease()
	if r == "" {
		t.Skip("no /proc/sys/kernel/osrelease")
	}
	if strings.ContainsAny(r, "\n\r") {
		t.Errorf("release should be one line, got %q", r)
	}
}
