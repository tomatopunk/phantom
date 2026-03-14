package expression

import (
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/pkg/agent/runtime"
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
	case "cpu":
		return fmt.Sprintf("%d", ev.CPU)
	case "probe_id":
		return fmt.Sprintf("%d", ev.ProbeID)
	case "event_type":
		return fmt.Sprintf("%d", ev.EventType)
	case "timestamp_ns":
		return fmt.Sprintf("%d", ev.TimestampNs)
	default:
		return "(unknown expression)"
	}
}
