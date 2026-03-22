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

package hook

import (
	"fmt"
	"strings"
	"unicode"
)

// AttachKind identifies where to attach a BPF program.
type AttachKind int

const (
	AttachKprobe AttachKind = iota
	AttachTracepoint
	AttachUprobe
	AttachUretprobe
)

// ParsedAttach is the result of parsing an attach string.
type ParsedAttach struct {
	Kind         AttachKind
	KprobeSymbol string
	TraceGroup   string
	TraceEvent   string
	UprobePath   string
	UprobeSymbol string
}

// ParseFullAttachPoint parses:
//   - kprobe:symbol
//   - tracepoint:subsystem:event
//   - uprobe:/path/to/bin:symbol  (path is everything before the last ':')
//   - uretprobe:/path:to:symbol (same path rule)
//
//nolint:gocyclo // one function covers all attach kinds and validation branches
func ParseFullAttachPoint(attach string) (*ParsedAttach, error) {
	attach = strings.TrimSpace(attach)
	parts := strings.SplitN(attach, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("attach point must be type:… (e.g. kprobe:do_sys_open)")
	}
	typ := strings.ToLower(strings.TrimSpace(parts[0]))
	rest := strings.TrimSpace(parts[1])
	if rest == "" {
		return nil, fmt.Errorf("empty attach target")
	}
	switch typ {
	case "kprobe":
		return &ParsedAttach{Kind: AttachKprobe, KprobeSymbol: rest}, nil
	case "tracepoint":
		sub := strings.SplitN(rest, ":", 2)
		if len(sub) != 2 || sub[0] == "" || sub[1] == "" {
			return nil, fmt.Errorf("tracepoint must be tracepoint:subsystem:event")
		}
		g, ev := strings.TrimSpace(sub[0]), strings.TrimSpace(sub[1])
		if err := validateBPFTraceIdent(g, "subsystem"); err != nil {
			return nil, err
		}
		if err := validateBPFTraceIdent(ev, "event"); err != nil {
			return nil, err
		}
		return &ParsedAttach{Kind: AttachTracepoint, TraceGroup: g, TraceEvent: ev}, nil
	case "uprobe", "uretprobe":
		i := strings.LastIndex(rest, ":")
		if i <= 0 || i == len(rest)-1 {
			return nil, fmt.Errorf("uprobe must be uprobe:/path/to/binary:symbol")
		}
		path := strings.TrimSpace(rest[:i])
		sym := strings.TrimSpace(rest[i+1:])
		if path == "" || sym == "" {
			return nil, fmt.Errorf("uprobe: empty path or symbol")
		}
		k := AttachUprobe
		if typ == "uretprobe" {
			k = AttachUretprobe
		}
		return &ParsedAttach{Kind: k, UprobePath: path, UprobeSymbol: sym}, nil
	default:
		return nil, fmt.Errorf("unsupported attach type %q (use kprobe, tracepoint, uprobe, uretprobe)", typ)
	}
}

func validateBPFTraceIdent(s, field string) error {
	if s == "" {
		return fmt.Errorf("empty tracepoint %s", field)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return fmt.Errorf("invalid tracepoint %s %q (only letters, digits, underscore)", field, s)
	}
	return nil
}
