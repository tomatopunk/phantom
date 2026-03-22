// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestRemoveTemporaryBreakpointsOnHit(t *testing.T) {
	sess := NewSession("test", "", nil)
	idPerm := sess.AddTemplateBreakpoint("kprobe.a", false, "h1", "", 0)
	idTemp := sess.AddTemplateBreakpoint("kprobe.b", true, "h2", "", 0)

	list := sess.ListBreakpoints()
	if len(list) != 2 {
		t.Fatalf("before hit: want 2 breakpoints, got %d", len(list))
	}

	sess.RemoveTemporaryBreakpointsOnHit()

	list = sess.ListBreakpoints()
	if len(list) != 1 {
		t.Fatalf("after hit: want 1 breakpoint, got %d", len(list))
	}
	if list[0].ID != idPerm || list[0].ProbeID != "kprobe.a" {
		t.Errorf("remaining breakpoint want %s kprobe.a, got %s %s", idPerm, list[0].ID, list[0].ProbeID)
	}
	if sess.GetBreakpoint(idTemp) != nil {
		t.Error("temporary breakpoint should be removed")
	}
}

func TestShouldReportBreakHit(t *testing.T) {
	ev := &runtime.Event{PID: 1, Tgid: 1, CPU: 0}

	sess := NewSession("test", "", nil)
	if !sess.ShouldReportBreakHit(ev, "hook-a") {
		t.Error("no breakpoints: should report")
	}

	sess.AddTemplateBreakpoint("kprobe.x", false, "hook-a", "", 0)
	if !sess.ShouldReportBreakHit(ev, "hook-a") {
		t.Error("one bp no condition: should report")
	}

	sess2 := NewSession("test2", "", nil)
	id2 := sess2.AddTemplateBreakpoint("kprobe.x", false, "hook-b", "", 0)
	sess2.SetBreakpointCondition(id2, "pid")
	if !sess2.ShouldReportBreakHit(ev, "hook-b") {
		t.Error("condition pid with ev.PID=1: should report")
	}

	sess3 := NewSession("test3", "", nil)
	id3 := sess3.AddTemplateBreakpoint("kprobe.x", false, "hook-c", "", 0)
	sess3.SetBreakpointCondition(id3, "cpu")
	if sess3.ShouldReportBreakHit(ev, "hook-c") {
		t.Error("condition cpu=0: should not report")
	}
}

func TestDisableEnableBreakpointTemplate(t *testing.T) {
	sess := NewSession("test", "", nil)
	id := sess.AddTemplateBreakpoint("kprobe.x", false, "hook-1", "", 0)

	if !sess.DisableBreakpoint(id) {
		t.Fatal("disable should succeed")
	}
	bp := sess.GetBreakpoint(id)
	if bp == nil || bp.Enabled || bp.HookID != "" {
		t.Errorf("after disable: enabled=%v hookID=%q", bp.Enabled, bp.HookID)
	}
	if sess.EnableBreakpoint(id) {
		t.Error("enable without recompile should fail (executor must reattach)")
	}
}
