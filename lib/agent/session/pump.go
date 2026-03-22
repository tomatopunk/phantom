// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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

// ProbeEventOpts configures ProcessProbeEvent for a specific ringbuf source.
type ProbeEventOpts struct {
	FromMainKprobePump bool   // legacy prebuilt kprobe .o pump
	HookID             string // non-empty for a template hook pump (e.g. hook-1)
}

// ProcessProbeEvent updates last-event context, emits TRACE_SAMPLE / STATE_CHANGE derivatives, and broadcasts ev.
// BREAK_HIT filtering uses ShouldReportBreakHit with a hook id for template hooks, or "" for the legacy main pump.
// Returns false if the event was suppressed when all breakpoint conditions fail.
func (s *Session) ProcessProbeEvent(ev *runtime.Event, opts ProbeEventOpts) bool {
	if ev.EventType == runtime.EventTypeBreakHit {
		sourceHookID := opts.HookID
		if opts.FromMainKprobePump {
			sourceHookID = ""
		}
		if !s.ShouldReportBreakHit(ev, sourceHookID) {
			return false
		}
		if opts.FromMainKprobePump {
			s.RemoveTemporaryBreakpointsOnHit()
		} else if opts.HookID != "" {
			s.RemoveTemporaryBreakpointsOnHitForHook(opts.HookID)
		}
	}
	s.SetLastEvent(ev)
	for _, sample := range s.EvaluateTraceSamples(ev) {
		var parts []string
		for _, expr := range sample.Expressions {
			parts = append(parts, expr+"="+sample.Values[expr])
		}
		payload := sample.TraceID
		if len(parts) > 0 {
			payload += " " + strings.Join(parts, " ")
		}
		traceEv := runtime.Event{
			TimestampNs: ev.TimestampNs,
			SessionID:   ev.SessionID,
			EventType:   eventTypeTraceSample,
			PID:         ev.PID,
			Tgid:        ev.Tgid,
			CPU:         ev.CPU,
			ProbeID:     ev.ProbeID,
			Payload:     []byte(payload),
		}
		s.BroadcastEvent(&traceEv)
	}
	for _, t := range s.EvaluateWatchChanges(ev) {
		payload := fmt.Sprintf("watch %s %s: %s -> %s", t.ID, t.Expression, t.OldValue, t.NewValue)
		watchEv := runtime.Event{
			TimestampNs: ev.TimestampNs,
			SessionID:   ev.SessionID,
			EventType:   eventTypeStateChange,
			PID:         ev.PID,
			Tgid:        ev.Tgid,
			CPU:         ev.CPU,
			ProbeID:     ev.ProbeID,
			Payload:     []byte(payload),
		}
		s.BroadcastEvent(&watchEv)
	}
	s.BroadcastEvent(ev)
	return true
}

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
		if !sess.ProcessProbeEvent(&evCopy, ProbeEventOpts{FromMainKprobePump: true}) {
			continue
		}
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
		if !sess.ProcessProbeEvent(&evCopy, ProbeEventOpts{HookID: hookID}) {
			continue
		}
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
