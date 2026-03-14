package expression

import (
	"testing"

	"github.com/tomatopunk/phantom/pkg/agent/runtime"
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
}
