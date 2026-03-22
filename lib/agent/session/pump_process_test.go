// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestProcessProbeEventHookPathTraceSample(t *testing.T) {
	s := NewSession("s1", "", nil)
	ch := make(chan *runtime.Event, 8)
	s.SubscribeEvents(ch)

	s.AddTrace([]string{"pid"}, nil)

	ev := &runtime.Event{
		TimestampNs: 1,
		EventType:   99,
		PID:         4242,
		Tgid:        4240,
		CPU:         0,
		ProbeID:     7,
	}
	if !s.ProcessProbeEvent(ev, ProbeEventOpts{HookID: "hook-1"}) {
		t.Fatal("ProcessProbeEvent(fromHook): want true")
	}

	var sawTrace, sawRaw bool
	for len(ch) > 0 {
		e := <-ch
		if e == nil {
			continue
		}
		if e.EventType == eventTypeTraceSample {
			sawTrace = true
		}
		if e.EventType == 99 && e.PID == 4242 {
			sawRaw = true
		}
	}
	if !sawTrace {
		t.Fatal("expected TRACE_SAMPLE on hook-path ProcessProbeEvent with trace registered")
	}
	if !sawRaw {
		t.Fatal("expected raw probe event broadcast")
	}
}

func TestProcessProbeEventMainPumpSuppressesBreakHitWhenAllConditionsFail(t *testing.T) {
	s := NewSession("s2", "", nil)
	ch := make(chan *runtime.Event, 8)
	s.SubscribeEvents(ch)

	id := s.AddBreakpoint("sym", func() {}, false, "")
	if !s.SetBreakpointCondition(id, "pid==1") {
		t.Fatal("SetBreakpointCondition")
	}

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
