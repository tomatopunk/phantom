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

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestRemoveTemporaryBreakpointsOnHit(t *testing.T) {
	sess := NewSession("test", "", nil)
	// Add one normal and one temporary breakpoint (detach can be nil for test).
	idPerm := sess.AddBreakpoint("do_sys_open", nil, false, "")
	idTemp := sess.AddBreakpoint("other_sym", nil, true, "")

	list := sess.ListBreakpoints()
	if len(list) != 2 {
		t.Fatalf("before hit: want 2 breakpoints, got %d", len(list))
	}

	sess.RemoveTemporaryBreakpointsOnHit()

	list = sess.ListBreakpoints()
	if len(list) != 1 {
		t.Fatalf("after hit: want 1 breakpoint, got %d", len(list))
	}
	if list[0].ID != idPerm || list[0].Symbol != "do_sys_open" {
		t.Errorf("remaining breakpoint want %s do_sys_open, got %s %s", idPerm, list[0].ID, list[0].Symbol)
	}
	if sess.GetBreakpoint(idTemp) != nil {
		t.Error("temporary breakpoint should be removed")
	}
}

func TestShouldReportBreakHit(t *testing.T) {
	ev := &runtime.Event{PID: 1, Tgid: 1, CPU: 0}

	// No breakpoints -> report
	sess := NewSession("test", "", nil)
	if !sess.ShouldReportBreakHit(ev, "") {
		t.Error("no breakpoints: should report")
	}

	// One breakpoint, no condition -> report
	sess.AddBreakpoint("sym", nil, false, "")
	if !sess.ShouldReportBreakHit(ev, "") {
		t.Error("one bp no condition: should report")
	}

	// One breakpoint, condition passes (pid 1)
	sess2 := NewSession("test2", "", nil)
	sess2.AddBreakpoint("sym", nil, false, "")
	sess2.SetBreakpointCondition("bp-1", "pid")
	if !sess2.ShouldReportBreakHit(ev, "") {
		t.Error("condition pid with ev.PID=1: should report")
	}

	// One breakpoint, condition fails (pid 1 but we check cpu)
	sess3 := NewSession("test3", "", nil)
	sess3.AddBreakpoint("sym", nil, false, "")
	sess3.SetBreakpointCondition("bp-1", "cpu") // cpu=0 is false
	if sess3.ShouldReportBreakHit(ev, "") {
		t.Error("condition cpu=0: should not report")
	}
}

func TestDisableEnableBreakpointReattach(t *testing.T) {
	// Session without runtime (empty kprobe path): enable after disable cannot re-attach.
	sess := NewSession("test", "", nil)
	detachCalled := false
	detach := func() { detachCalled = true }
	id := sess.AddBreakpoint("do_sys_open", detach, false, "")

	if !sess.DisableBreakpoint(id) {
		t.Fatal("disable should succeed")
	}
	if !detachCalled {
		t.Error("disable should have called detach")
	}
	bp := sess.GetBreakpoint(id)
	if bp == nil || bp.Enabled || bp.Detach != nil {
		t.Errorf("after disable: enabled=%v detach=%v", bp.Enabled, bp.Detach != nil)
	}
	// Re-enable without runtime: cannot re-attach, so EnableBreakpoint fails.
	if sess.EnableBreakpoint(id) {
		t.Error("enable without runtime should fail")
	}
}
