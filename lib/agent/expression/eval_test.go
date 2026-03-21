package expression

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestEvaluate(t *testing.T) {
	ev := &runtime.Event{PID: 42, Tgid: 40, CPU: 2, ProbeID: 1, EventType: 1, TimestampNs: 1000}

	if got := Evaluate(nil, "pid"); got != "(no event yet)" {
		t.Errorf("Evaluate(nil, pid) = %q want (no event yet)", got)
	}
	if got := Evaluate(ev, "pid"); got != "42" {
		t.Errorf("Evaluate(ev, pid) = %q want 42", got)
	}
	if got := Evaluate(ev, "tgid"); got != "40" {
		t.Errorf("Evaluate(ev, tgid) = %q want 40", got)
	}
	if got := Evaluate(ev, "cpu"); got != "2" {
		t.Errorf("Evaluate(ev, cpu) = %q want 2", got)
	}
	if got := Evaluate(ev, "probe_id"); got != "1" {
		t.Errorf("Evaluate(ev, probe_id) = %q want 1", got)
	}
	if got := Evaluate(ev, "event_type"); got != "1" {
		t.Errorf("Evaluate(ev, event_type) = %q want 1", got)
	}
	if got := Evaluate(ev, "timestamp_ns"); got != "1000" {
		t.Errorf("Evaluate(ev, timestamp_ns) = %q want 1000", got)
	}
	if got := Evaluate(ev, "unknown"); got != "(unknown expression)" {
		t.Errorf("Evaluate(ev, unknown) = %q want (unknown expression)", got)
	}
	// normalizes expr
	if got := Evaluate(ev, "  PID  "); got != "42" {
		t.Errorf("Evaluate(ev, '  PID  ') = %q want 42", got)
	}
	// arg0..arg5, ret (zero when not in event)
	if got := Evaluate(ev, "arg0"); got != "0" {
		t.Errorf("Evaluate(ev, arg0) = %q want 0", got)
	}
	if got := Evaluate(ev, "arg5"); got != "0" {
		t.Errorf("Evaluate(ev, arg5) = %q want 0", got)
	}
	if got := Evaluate(ev, "ret"); got != "0" {
		t.Errorf("Evaluate(ev, ret) = %q want 0", got)
	}
	// comm when empty
	if got := Evaluate(ev, "comm"); got != "(not in event)" {
		t.Errorf("Evaluate(ev, comm) = %q want (not in event)", got)
	}
}

func TestEvaluateArgRetComm(t *testing.T) {
	ev := &runtime.Event{}
	ev.Args[0] = 100
	ev.Args[5] = 200
	ev.Ret = 0
	ev.Comm = "bash"

	if got := Evaluate(ev, "arg0"); got != "100" {
		t.Errorf("Evaluate(ev, arg0) = %q want 100", got)
	}
	if got := Evaluate(ev, "arg5"); got != "200" {
		t.Errorf("Evaluate(ev, arg5) = %q want 200", got)
	}
	if got := Evaluate(ev, "ret"); got != "0" {
		t.Errorf("Evaluate(ev, ret) = %q want 0", got)
	}
	if got := Evaluate(ev, "comm"); got != "bash" {
		t.Errorf("Evaluate(ev, comm) = %q want bash", got)
	}
}

func TestConditionPasses(t *testing.T) {
	ev := &runtime.Event{PID: 42, Tgid: 40, CPU: 0}

	if !ConditionPasses(ev, "") {
		t.Error("empty condition should pass")
	}
	if !ConditionPasses(ev, "pid") {
		t.Error("pid=42 should be truthy")
	}
	if ConditionPasses(ev, "cpu") {
		t.Error("cpu=0 should be false")
	}
	if !ConditionPasses(ev, "1") {
		t.Error("1 should pass")
	}
	if ConditionPasses(ev, "0") {
		t.Error("0 should not pass")
	}
	if ConditionPasses(nil, "pid") {
		t.Error("nil event should not pass")
	}
}
