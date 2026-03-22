// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestProcessProbeEventHookPathWatchArg(t *testing.T) {
	s := NewSession("s1", "", nil)
	ch := make(chan *runtime.Event, 16)
	s.SubscribeEvents(ch)

	s.AddTemplateBreakpoint("kprobe.do_sys_open", false, "hook-1", "", 0)
	s.AddArgWatch("kprobe.do_sys_open", []int{2})

	ev := &runtime.Event{
		TimestampNs: 1,
		EventType:   runtime.EventTypeBreakHit,
		PID:         4242,
		Tgid:        4240,
		CPU:         0,
		ProbeID:     7,
		Args:        [6]uint64{99, 0, 0, 0, 0, 0},
	}
	if !s.ProcessProbeEvent(ev, ProbeEventOpts{HookID: "hook-1"}) {
		t.Fatal("ProcessProbeEvent(fromHook): want true")
	}

	var sawWatch, sawBreak bool
	for len(ch) > 0 {
		e := <-ch
		if e == nil {
			continue
		}
		switch e.EventType {
		case runtime.EventTypeWatchArg:
			sawWatch = true
		case runtime.EventTypeBreakHit:
			sawBreak = true
		}
	}
	if !sawWatch {
		t.Fatal("expected WATCH_ARG when watch registered on probe")
	}
	if !sawBreak {
		t.Fatal("expected BREAK_HIT broadcast")
	}
}

func TestProcessProbeEventMainPumpSuppressesBreakHitWhenAllConditionsFail(t *testing.T) {
	s := NewSession("s2", "", nil)
	ch := make(chan *runtime.Event, 8)
	s.SubscribeEvents(ch)

	s.mu.Lock()
	s.nextBPID++
	id := fmtID("bp", s.nextBPID)
	s.breakpoints[id] = &BreakpointState{
		ID: id, Symbol: "sym", ProbeID: "sym", Enabled: true,
		Condition: "pid==1", KprobeHook: false, HookID: "",
	}
	s.mu.Unlock()

	ev := &runtime.Event{
		EventType: runtime.EventTypeBreakHit,
		PID:       2,
	}
	if s.ProcessProbeEvent(ev, ProbeEventOpts{FromMainKprobePump: true}) {
		t.Fatal("expected suppression when condition fails")
	}
	select {
	case e := <-ch:
		t.Fatalf("unexpected broadcast: %+v", e)
	default:
	}
}
