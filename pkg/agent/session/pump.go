package session

import (
	"context"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"

	"github.com/tomatopunk/phantom/pkg/agent/runtime"
)

// EventType STATE_CHANGE (4) for watch triggers; matches proto.EventType_EVENT_TYPE_STATE_CHANGE.
const eventTypeStateChange = 4

// runEventPump reads from the ring buffer, decodes events, updates last event and broadcasts to subscribers.
// After each event it evaluates watch expressions and broadcasts a STATE_CHANGE event for each value change.
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
		sess.SetLastEvent(&evCopy)
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
