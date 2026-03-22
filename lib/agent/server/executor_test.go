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

func TestExecuteBreakPrintContinue(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")

	tests := []struct {
		line   string
		wantOk bool
	}{
		{"break kprobe.do_sys_open", false}, // no bpf include
		{"print pid", true},
		{"continue", true},
		{"break", false},
		{"print", false},
		{"unknown_cmd", false},
	}
	for _, tt := range tests {
		resp, err := exec.execute(context.Background(), sess, tt.line)
		if err != nil {
			t.Errorf("execute %q: %v", tt.line, err)
			continue
		}
		if resp.GetOk() != tt.wantOk {
			t.Errorf("execute %q: ok=%v want %v", tt.line, resp.GetOk(), tt.wantOk)
		}
		if tt.wantOk && resp.GetOutput() == "" && !strings.Contains(tt.line, "continue") {
			if resp.GetBreakpoint() == nil && resp.GetPrint() == nil && resp.GetWatch() == nil {
				t.Errorf("execute %q: expected some output or result", tt.line)
			}
		}
	}
}

func TestExecuteList(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("list: want ok true got false")
	}
	if !strings.Contains(resp.GetOutput(), "specify a symbol") {
		t.Errorf("list: output should ask for symbol, got %q", resp.GetOutput())
	}

	resp, err = exec.execute(ctx, sess, "list do_sys_open")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("list do_sys_open: want ok true")
	}
	if resp.GetOutput() == "" {
		t.Error("list do_sys_open: expected some output")
	}
}

func TestExecuteBt(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "bt")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("bt: want ok true")
	}
	out := resp.GetOutput()
	if !strings.Contains(out, "bt") {
		t.Errorf("bt: output should contain bt, got %q", out)
	}
	if !strings.Contains(out, "event") && !strings.Contains(out, "supported") && !strings.Contains(out, "stack") {
		t.Errorf("bt: expected event/supported/stack in output, got %q", out)
	}
}

func TestExecuteWatchRequiresBreak(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "watch")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("watch: want ok false when missing args")
	}

	resp, err = exec.execute(ctx, sess, "watch --sec kprobe.do_sys_open")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "no enabled break") {
		t.Fatalf("watch without break: got ok=%v err=%q", resp.GetOk(), resp.GetErrorMessage())
	}
}

func TestExecuteWatchAndDeleteAndInfoWatch(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	sess.AddTemplateBreakpoint("kprobe.do_sys_open", false, "hook-1", "", 0)

	resp, err := exec.execute(ctx, sess, "watch --sec kprobe.do_sys_open --args 2,3")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Fatalf("watch: %s", resp.GetErrorMessage())
	}
	if resp.GetWatch() == nil {
		t.Fatal("expected Watch result")
	}

	resp, err = exec.execute(ctx, sess, "info watch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.GetOutput(), "kprobe.do_sys_open") {
		t.Errorf("info watch: got %q", resp.GetOutput())
	}

	list := sess.ListWatches()
	if len(list) != 1 {
		t.Fatalf("expected 1 watch, got %d", len(list))
	}
	id := list[0].ID
	resp, err = exec.execute(ctx, sess, "delete "+id)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Fatal(resp.GetErrorMessage())
	}
	if len(sess.ListWatches()) != 0 {
		t.Error("watch should be removed")
	}
}

func TestExecuteHookAddRemoved(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c --sec pid==1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add: want ok false (removed)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "removed") {
		t.Errorf("hook add: want 'removed' in error, got %q", resp.GetErrorMessage())
	}
}

func TestExecuteHookAttach(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "hook attach --source 'int x;'")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "no bpf include dir") {
		t.Errorf("hook attach: want no bpf include dir, got %q", resp.GetErrorMessage())
	}

	resp, err = exec.execute(ctx, sess, "hook attach --source 'int a;' --file /tmp/x.c")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "cannot use both") {
		t.Errorf("hook attach both file and source: want error, got %q", resp.GetErrorMessage())
	}
}

func TestParseTemplateBreakArgs(t *testing.T) {
	_, _, _, msg := parseTemplateBreakArgs("break", []string{"--filter", "pid==1"}, false)
	if msg == "" || !strings.Contains(msg, "missing probe_id") {
		t.Fatalf("want missing probe_id, got %q", msg)
	}
	pid, flt, limit, msg := parseTemplateBreakArgs("break", []string{
		"kprobe.do_sys_open", "--filter", "pid==1",
	}, false)
	if msg != "" || pid != "kprobe.do_sys_open" || flt != "pid==1" || limit != 0 {
		t.Fatalf("got pid=%q flt=%q limit=%d msg=%q", pid, flt, limit, msg)
	}
	_, _, limit, msg = parseTemplateBreakArgs("tbreak", []string{
		"kprobe.foo", "--limit", "3",
	}, true)
	if msg != "" || limit != 3 {
		t.Fatalf("tbreak limit: msg=%q limit=%d", msg, limit)
	}
}

func TestExecuteBreakUnknownProbe(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil)
	mgr := session.NewManager("", nil)
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()
	resp, err := exec.execute(ctx, sess, "break not_a_real_probe_id")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "unknown probe_id") {
		t.Fatalf("want unknown probe_id, got ok=%v err=%q", resp.GetOk(), resp.GetErrorMessage())
	}
}
