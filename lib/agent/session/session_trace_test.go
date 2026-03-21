package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestEvaluateTraceSamples(t *testing.T) {
	sess := NewSession("test", "")
	// No traces: empty result
	ev := &runtime.Event{PID: 42, Tgid: 40, CPU: 1}
	samples := sess.EvaluateTraceSamples(ev)
	if len(samples) != 0 {
		t.Errorf("no traces: want 0 samples, got %d", len(samples))
	}

	sess.AddTrace([]string{"pid", "cpu"}, nil)
	sess.AddTrace([]string{"tgid"}, nil)
	samples = sess.EvaluateTraceSamples(ev)
	if len(samples) != 2 {
		t.Fatalf("two traces: want 2 samples, got %d", len(samples))
	}
	byID := make(map[string]TraceSampleResult)
	for _, s := range samples {
		byID[s.TraceID] = s
	}
	if s1, ok := byID["trace-1"]; ok {
		if len(s1.Expressions) != 2 || s1.Values["pid"] != "42" || s1.Values["cpu"] != "1" {
			t.Errorf("trace-1: want pid=42 cpu=1, got %v", s1.Values)
		}
	}
	if s2, ok := byID["trace-2"]; ok {
		if s2.Values["tgid"] != "40" {
			t.Errorf("trace-2: want tgid=40, got %v", s2.Values)
		}
	}
}
