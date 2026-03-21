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

import (
	"encoding/binary"
	"errors"
)

// eventHeaderSize is the fixed size of event_header in the eBPF C code.
const eventHeaderSize = 32

// DecodeEvent parses a ringbuf record into Event (matches event_header layout).
func DecodeEvent(raw []byte) (Event, error) {
	if len(raw) < eventHeaderSize {
		return Event{}, errors.New("event too short")
	}
	ev := Event{
		TimestampNs: binary.NativeEndian.Uint64(raw[0:8]),
		SessionID:   binary.NativeEndian.Uint32(raw[8:12]),
		EventType:   binary.NativeEndian.Uint32(raw[12:16]),
		PID:         binary.NativeEndian.Uint32(raw[16:20]),
		Tgid:        binary.NativeEndian.Uint32(raw[20:24]),
		CPU:         binary.NativeEndian.Uint32(raw[24:28]),
		ProbeID:     binary.NativeEndian.Uint32(raw[28:32]),
	}
	if len(raw) > eventHeaderSize {
		ev.Payload = make([]byte, len(raw)-eventHeaderSize)
		copy(ev.Payload, raw[eventHeaderSize:])
	}
	return ev, nil
}
