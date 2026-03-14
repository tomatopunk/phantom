package session

import (
	"context"

	"github.com/cilium/ebpf/ringbuf"

	"github.com/tomatopunk/phantom/pkg/agent/runtime"
)

// runEventPump reads from the ring buffer, decodes events, updates last event and broadcasts to subscribers.
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
		sess.BroadcastEvent(&evCopy)
	}
}
