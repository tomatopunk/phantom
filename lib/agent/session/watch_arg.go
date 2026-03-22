// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/breaktpl"
	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func formatWatchColumn(entry *breaktpl.Entry, paramIndex int, ev *runtime.Event) string {
	if entry == nil || paramIndex < 0 || paramIndex >= len(entry.Params) {
		return "?=?"
	}
	name := entry.Params[paramIndex]
	switch name {
	case "pid":
		return fmt.Sprintf("%s=%d", name, ev.PID)
	case "tgid":
		return fmt.Sprintf("%s=%d", name, ev.Tgid)
	default:
		if strings.HasPrefix(name, "arg") && len(name) == 4 {
			n := name[3] - '0'
			if n <= 5 {
				return fmt.Sprintf("%s=%d", name, ev.Args[n])
			}
		}
	}
	return name + "=?"
}

// EvaluateWatchArgEvents returns WATCH_ARG runtime events for watches registered on probeID (caller holds no locks).
func (s *Session) EvaluateWatchArgEvents(hit *runtime.Event, probeID string) []*runtime.Event {
	entry, ok := breaktpl.Lookup(probeID)
	if !ok {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*runtime.Event
	for _, w := range s.watches {
		if w.ProbeID != probeID {
			continue
		}
		indices := w.ArgParamIndices
		if len(indices) == 0 {
			indices = append([]int(nil), entry.DefaultArgIndices...)
		}
		var parts []string
		for _, pi := range indices {
			parts = append(parts, formatWatchColumn(entry, pi, hit))
		}
		payload := w.ID + " " + strings.Join(parts, " ")
		out = append(out, &runtime.Event{
			TimestampNs:     hit.TimestampNs,
			SessionID:         hit.SessionID,
			EventType:         runtime.EventTypeWatchArg,
			PID:               hit.PID,
			Tgid:              hit.Tgid,
			CPU:               hit.CPU,
			ProbeID:           hit.ProbeID,
			Payload:           []byte(payload),
			SourceKind:        "watch",
			TemplateProbeID:   probeID,
			Args:              hit.Args,
			Ret:               hit.Ret,
			Comm:              hit.Comm,
		})
	}
	return out
}
