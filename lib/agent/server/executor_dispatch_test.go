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

package server

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/session"
)

func TestReplVerbAliases(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "alias-test")
	ctx := context.Background()

	for _, line := range []string{"b kprobe.do_sys_open", "p pid", "t kprobe.do_sys_open", "c"} {
		resp, err := exec.execute(ctx, sess, line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if !resp.GetOk() && (strings.HasPrefix(line, "b ") || strings.HasPrefix(line, "t ")) {
			if strings.Contains(resp.GetErrorMessage(), "unknown command") {
				t.Fatalf("%q: should not be unknown command: %s", line, resp.GetErrorMessage())
			}
		}
		if strings.HasPrefix(line, "p ") || line == "c" {
			if !resp.GetOk() {
				t.Fatalf("%q: want ok, got %s", line, resp.GetErrorMessage())
			}
		}
	}
}
