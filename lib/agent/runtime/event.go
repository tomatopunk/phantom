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

package runtime

// Event type constants; match bpf/include/common.h and proto.
const (
	EventTypeBreakHit    = 1
	EventTypeWatchArg    = 2
	EventTypeError       = 3
	EventTypeStateChange = 4
)

// Event is a single debug event from the eBPF ring buffer (matches event_header; optional Args/Ret/Comm from payload).
type Event struct {
	TimestampNs uint64
	SessionID   uint32
	EventType   uint32
	PID         uint32
	Tgid        uint32
	CPU         uint32
	ProbeID     uint32
	Payload     []byte
	// Optional: filled from payload when BPF sends pt_regs / ABI (arg0-arg5, ret, comm).
	Args [6]uint64
	Ret  uint64
	Comm string
	// SourceKind is break | watch | hook for UI/stream attribution.
	SourceKind string
	BreakID    string
	HookID     string
	// TemplateProbeID is the break template catalog id when applicable.
	TemplateProbeID string
}
