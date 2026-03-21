package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/cilium/ebpf/ringbuf"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

// EventType constants; match proto.EventType.
const (
	eventTypeTraceSample = 2 // EVENT_TYPE_TRACE_SAMPLE
	eventTypeStateChange = 4 // EVENT_TYPE_STATE_CHANGE
)

// runEventPump reads from the ring buffer, decodes events, updates last event and broadcasts to subscribers.
// After each event it evaluates trace expressions (TRACE_SAMPLE) and watch expressions (STATE_CHANGE), then broadcasts the raw event.
func runEventPump(ctx context.Context, sess *Session, reader *ringbuf.Reader) {
	defer reader.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		record, err := reader.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		ev, err := runtime.DecodeEvent(record.RawSample)
		if err != nil {
			continue
		}
		evCopy := ev
		if evCopy.EventType == runtime.EventTypeBreakHit && !sess.ShouldReportBreakHit(&evCopy) {
			// Condition not satisfied for any breakpoint; suppress this BREAK_HIT.
			continue
		}
		sess.SetLastEvent(&evCopy)
		// On BREAK_HIT, auto-remove temporary breakpoints (tbreak).
		if evCopy.EventType == runtime.EventTypeBreakHit {
			sess.RemoveTemporaryBreakpointsOnHit()
		}
		// Broadcast TRACE_SAMPLE for each registered trace.
		for _, sample := range sess.EvaluateTraceSamples(&evCopy) {
			var parts []string
			for _, expr := range sample.Expressions {
				parts = append(parts, expr+"="+sample.Values[expr])
			}
			payload := sample.TraceID
			if len(parts) > 0 {
				payload += " " + strings.Join(parts, " ")
			}
			traceEv := runtime.Event{
				TimestampNs: evCopy.TimestampNs,
				SessionID:   evCopy.SessionID,
				EventType:   eventTypeTraceSample,
				PID:         evCopy.PID,
				Tgid:        evCopy.Tgid,
				CPU:         evCopy.CPU,
				ProbeID:     evCopy.ProbeID,
				Payload:     []byte(payload),
			}
			sess.BroadcastEvent(&traceEv)
		}
		triggers := sess.EvaluateWatchChanges(&evCopy)
		for _, t := range triggers {
			payload := fmt.Sprintf("watch %s %s: %s -> %s", t.ID, t.Expression, t.OldValue, t.NewValue)
			watchEv := runtime.Event{
				TimestampNs: evCopy.TimestampNs,
				SessionID:   evCopy.SessionID,
				EventType:   eventTypeStateChange,
				PID:         evCopy.PID,
				Tgid:        evCopy.Tgid,
				CPU:         evCopy.CPU,
				ProbeID:     evCopy.ProbeID,
				Payload:     []byte(payload),
			}
			sess.BroadcastEvent(&watchEv)
		}
		sess.BroadcastEvent(&evCopy)
	}
}

// runHookEventPump is like runEventPump but for a single hook's reader; when the hook has a limit,
// it auto-removes the hook after that many events.
func runHookEventPump(ctx context.Context, sess *Session, reader *ringbuf.Reader, hookID string) {
	defer reader.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		record, err := reader.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		ev, err := runtime.DecodeEvent(record.RawSample)
		if err != nil {
			continue
		}
		evCopy := ev
		sess.SetLastEvent(&evCopy)
		sess.BroadcastEvent(&evCopy)
		// Auto-remove when limit reached
		if hookID != "" {
			count, limit, ok := sess.IncrementHookHitCount(hookID)
			if ok && limit > 0 && count >= limit {
				sess.RemoveHook(hookID)
				return
			}
		}
	}
}
