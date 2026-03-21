package expression

import (
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

// Evaluate returns the value of expr in the context of ev (e.g. pid, tgid, cpu).
// Normalizes expr (lowercase, trim). Returns "(no event yet)" if ev is nil,
// "(unknown expression)" for unsupported expr.
func Evaluate(ev *runtime.Event, expr string) string {
	expr = strings.ToLower(strings.TrimSpace(expr))
	if ev == nil {
		return "(no event yet)"
	}
	switch expr {
	case "pid":
		return fmt.Sprintf("%d", ev.PID)
	case "tgid":
		return fmt.Sprintf("%d", ev.Tgid)
	case "comm":
		if ev.Comm != "" {
			return ev.Comm
		}
		return "(not in event)"
	case "cpu":
		return fmt.Sprintf("%d", ev.CPU)
	case "probe_id":
		return fmt.Sprintf("%d", ev.ProbeID)
	case "event_type":
		return fmt.Sprintf("%d", ev.EventType)
	case "timestamp_ns":
		return fmt.Sprintf("%d", ev.TimestampNs)
	case "arg0":
		return fmt.Sprintf("%d", ev.Args[0])
	case "arg1":
		return fmt.Sprintf("%d", ev.Args[1])
	case "arg2":
		return fmt.Sprintf("%d", ev.Args[2])
	case "arg3":
		return fmt.Sprintf("%d", ev.Args[3])
	case "arg4":
		return fmt.Sprintf("%d", ev.Args[4])
	case "arg5":
		return fmt.Sprintf("%d", ev.Args[5])
	case "ret":
		return fmt.Sprintf("%d", ev.Ret)
	default:
		return "(unknown expression)"
	}
}

// ConditionPasses returns true if the condition expression evaluates to a truthy value in the event context.
// Used for breakpoint conditions: "1", "true", and non-zero numeric strings are truthy; "0", "false", empty, or errors are false.
func ConditionPasses(ev *runtime.Event, cond string) bool {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return true
	}
	if cond == "1" || strings.ToLower(cond) == "true" || strings.ToLower(cond) == "yes" {
		return true
	}
	if cond == "0" || strings.ToLower(cond) == "false" || strings.ToLower(cond) == "no" {
		return false
	}
	val := Evaluate(ev, cond)
	switch strings.ToLower(val) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no", "":
		return false
	}
	if val == "(no event yet)" || val == "(unknown expression)" {
		return false
	}
	// Non-zero numeric or other non-empty value is truthy
	return len(val) > 0 && val != "0"
}
