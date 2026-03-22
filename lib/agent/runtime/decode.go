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

// phantomRingRecordSize is PHANTOM_RING_RECORD_SIZE (header + 6 x u64 args).
const phantomRingRecordSize = 80

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
	if len(raw) >= phantomRingRecordSize {
		for i := range ev.Args {
			off := 32 + i*8
			ev.Args[i] = binary.NativeEndian.Uint64(raw[off : off+8])
		}
		if len(raw) > phantomRingRecordSize {
			ev.Payload = make([]byte, len(raw)-phantomRingRecordSize)
			copy(ev.Payload, raw[phantomRingRecordSize:])
		}
	} else if len(raw) > eventHeaderSize {
		ev.Payload = make([]byte, len(raw)-eventHeaderSize)
		copy(ev.Payload, raw[eventHeaderSize:])
	}
	return ev, nil
}
