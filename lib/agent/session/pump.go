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
	"strings"

	"github.com/cilium/ebpf/ringbuf"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

// ProbeEventOpts configures ProcessProbeEvent for a specific ringbuf source.
type ProbeEventOpts struct {
	FromMainKprobePump bool   // legacy prebuilt kprobe .o pump
	HookID             string // non-empty for a hook pump (e.g. hook-1)
}

// ProcessProbeEvent updates last-event context, emits WATCH_ARG side events on BREAK_HIT, and broadcasts ev.
func (s *Session) ProcessProbeEvent(ev *runtime.Event, opts ProbeEventOpts) bool {
	if opts.HookID != "" {
		if h := s.GetHook(opts.HookID); h != nil {
			switch strings.TrimSpace(h.Note) {
			case "hook attach", "CompileAndAttach":
				ev.SourceKind = "hook"
				ev.HookID = opts.HookID
			}
		}
	}

	if ev.EventType == runtime.EventTypeBreakHit {
		sourceHookID := opts.HookID
		if opts.FromMainKprobePump {
			sourceHookID = ""
		}
		if !s.ShouldReportBreakHit(ev, sourceHookID) {
			return false
		}
		var bp *BreakpointState
		if sourceHookID != "" {
			for _, b := range s.ListBreakpoints() {
				if b.Enabled && b.HookID == sourceHookID {
					bp = b
					break
				}
			}
		} else {
			for _, b := range s.ListBreakpoints() {
				if b.Enabled && !b.KprobeHook {
					bp = b
					break
				}
			}
		}
		if bp != nil && bp.KprobeHook {
			ev.SourceKind = "break"
			ev.BreakID = bp.ID
			ev.TemplateProbeID = bp.ProbeID
			for _, wev := range s.EvaluateWatchArgEvents(ev, bp.ProbeID) {
				s.BroadcastEvent(wev)
			}
		}
		if opts.FromMainKprobePump {
			s.RemoveTemporaryBreakpointsOnHit()
		} else if opts.HookID != "" {
			s.RemoveTemporaryBreakpointsOnHitForHook(opts.HookID)
		}
	}
	s.SetLastEvent(ev)
	s.BroadcastEvent(ev)
	return true
}

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
		if hookID != "" {
			count, limit, ok := sess.IncrementHookHitCount(hookID)
			if ok && limit > 0 && count >= limit {
				sess.RemoveHook(hookID)
				return
			}
		}
	}
}
