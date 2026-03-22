// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

// Package breaktpl defines the built-in template library for break/watch.
package breaktpl

// Kind is the probe class (no uprobes in the catalog).
type Kind int

const (
	KindKprobe Kind = iota
	KindTracepoint
)

// Entry is one agent-defined probe point (single meaning for probe_id).
type Entry struct {
	ProbeID string
	Kind    Kind
	// KprobeSymbol is the kernel symbol for KindKprobe (SEC kprobe/<symbol>).
	KprobeSymbol string
	// TraceGroup / TraceEvent for KindTracepoint (SEC tracepoint/<g>/<e>).
	TraceGroup string
	TraceEvent string
	// Params lists names allowed in watch output and (where applicable) DSL.
	Params []string
	// DefaultArgIndices are 0-based indices into Params that are "arg-like" (arg0..) for default watch columns.
	DefaultArgIndices []int
	// FilterParams lists which identifiers may appear in break filter DSL for this entry.
	FilterParams []string
}

// Catalog is the fixed template library (extend by appending entries here).
var Catalog = []Entry{
	{
		ProbeID: "kprobe.do_sys_open", Kind: KindKprobe, KprobeSymbol: "do_sys_open",
		Params:            []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
		DefaultArgIndices: []int{2, 3, 4, 5, 6, 7},
		FilterParams:      []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
	},
	{
		ProbeID: "kprobe.do_nanosleep", Kind: KindKprobe, KprobeSymbol: "do_nanosleep",
		Params:            []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
		DefaultArgIndices: []int{2, 3, 4, 5, 6, 7},
		FilterParams:      []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
	},
	{
		ProbeID: "kprobe.tcp_sendmsg", Kind: KindKprobe, KprobeSymbol: "tcp_sendmsg",
		Params:            []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
		DefaultArgIndices: []int{2, 3, 4, 5, 6, 7},
		FilterParams:      []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
	},
	{
		ProbeID: "tp.sched.sched_process_fork", Kind: KindTracepoint,
		TraceGroup: "sched", TraceEvent: "sched_process_fork",
		Params:            []string{"pid", "tgid", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5"},
		DefaultArgIndices: []int{2, 3}, // parent/child pid packed in first two arg slots in generated BPF
		FilterParams:      []string{"pid", "tgid"},
	},
}

// Lookup returns the catalog entry for probe_id.
func Lookup(probeID string) (*Entry, bool) {
	for i := range Catalog {
		if Catalog[i].ProbeID == probeID {
			return &Catalog[i], true
		}
	}
	return nil, false
}

// List returns a copy of all entries.
func List() []Entry {
	out := make([]Entry, len(Catalog))
	copy(out, Catalog)
	return out
}

// ParsedAttachPoint returns the hook attach string (probe_point) for loader parsing.
func (e *Entry) ParsedAttachPoint() string {
	switch e.Kind {
	case KindKprobe:
		return "kprobe:" + e.KprobeSymbol
	case KindTracepoint:
		return "tracepoint:" + e.TraceGroup + ":" + e.TraceEvent
	default:
		return ""
	}
}
