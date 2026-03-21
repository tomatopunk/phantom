package hook

import (
	"fmt"
	"strings"
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
		return &ParsedAttach{Kind: AttachTracepoint, TraceGroup: sub[0], TraceEvent: sub[1]}, nil
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
