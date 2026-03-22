// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package hook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cilium/ebpf"
)

// ParseProbePointFromELF loads the BPF object spec and derives the probe attachment
// from the selected program's ELF section name (e.g. kprobe/foo, tracepoint/a/b).
// When programName is empty, the first program of a suitable probe type is used.
func ParseProbePointFromELF(objectPath, programName string) (*ParsedAttach, string, error) {
	spec, err := ebpf.LoadCollectionSpec(objectPath)
	if err != nil {
		return nil, "", fmt.Errorf("load spec: %w", err)
	}
	return parseProbePointFromSpec(spec, programName)
}

func parseProbePointFromSpec(spec *ebpf.CollectionSpec, programName string) (*ParsedAttach, string, error) {
	name, sec, err := pickProgramSection(spec, programName)
	if err != nil {
		return nil, "", err
	}
	pa, err := parseSectionToAttach(sec)
	if err != nil {
		return nil, "", err
	}
	return pa, name, nil
}

func pickProgramSection(spec *ebpf.CollectionSpec, programName string) (progName, section string, err error) {
	if programName != "" {
		ps, ok := spec.Programs[programName]
		if !ok {
			return "", "", fmt.Errorf("program %q not found in object", programName)
		}
		return programName, ps.SectionName, nil
	}
	var names []string
	for n := range spec.Programs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		ps := spec.Programs[n]
		switch ps.Type {
		case ebpf.Kprobe, ebpf.TracePoint:
			return n, ps.SectionName, nil
		}
	}
	if len(names) > 0 {
		n := names[0]
		return n, spec.Programs[n].SectionName, nil
	}
	return "", "", fmt.Errorf("no BPF programs in object")
}

func parseSectionToAttach(section string) (*ParsedAttach, error) {
	section = strings.TrimSpace(section)
	// cilium uses lowercase "kprobe" in Type but section is like "kprobe/do_sys_open"
	if strings.HasPrefix(section, "kprobe/") {
		sym := strings.TrimSpace(strings.TrimPrefix(section, "kprobe/"))
		if sym == "" {
			return nil, fmt.Errorf("empty kprobe symbol in section %q", section)
		}
		return &ParsedAttach{Kind: AttachKprobe, KprobeSymbol: sym}, nil
	}
	if strings.HasPrefix(section, "tracepoint/") {
		rest := strings.TrimPrefix(section, "tracepoint/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("tracepoint section must be tracepoint/group/name, got %q", section)
		}
		return &ParsedAttach{Kind: AttachTracepoint, TraceGroup: parts[0], TraceEvent: parts[1]}, nil
	}
	if strings.HasPrefix(section, "uprobe/") || section == "uprobe" {
		// Expect uprobe/path:sym in section for libbpf-style objects.
		if section == "uprobe" {
			return nil, fmt.Errorf("ambiguous uprobe section %q: use a section like uprobe/ABS_PATH:symbol", section)
		}
		rest := strings.TrimPrefix(section, "uprobe/")
		i := strings.LastIndex(rest, ":")
		if i <= 0 || i == len(rest)-1 {
			return nil, fmt.Errorf("uprobe section must be uprobe/PATH:symbol, got %q", section)
		}
		path := strings.TrimSpace(rest[:i])
		sym := strings.TrimSpace(rest[i+1:])
		return &ParsedAttach{Kind: AttachUprobe, UprobePath: path, UprobeSymbol: sym}, nil
	}
	if strings.HasPrefix(section, "uretprobe/") || section == "uretprobe" {
		if section == "uretprobe" {
			return nil, fmt.Errorf("ambiguous uretprobe section %q: use uretprobe/PATH:symbol", section)
		}
		rest := strings.TrimPrefix(section, "uretprobe/")
		i := strings.LastIndex(rest, ":")
		if i <= 0 || i == len(rest)-1 {
			return nil, fmt.Errorf("uretprobe section must be uretprobe/PATH:symbol, got %q", section)
		}
		path := strings.TrimSpace(rest[:i])
		sym := strings.TrimSpace(rest[i+1:])
		return &ParsedAttach{Kind: AttachUretprobe, UprobePath: path, UprobeSymbol: sym}, nil
	}
	return nil, fmt.Errorf("unsupported program section %q (need kprobe/, tracepoint/, uprobe/, uretprobe/)", section)
}

// FormatProbePoint returns a stable textual probe_point for display (replaces ambiguous "attach_point").
func FormatProbePoint(pa *ParsedAttach) string {
	if pa == nil {
		return ""
	}
	switch pa.Kind {
	case AttachKprobe:
		return "kprobe:" + pa.KprobeSymbol
	case AttachTracepoint:
		return "tracepoint:" + pa.TraceGroup + ":" + pa.TraceEvent
	case AttachUprobe:
		return "uprobe:" + pa.UprobePath + ":" + pa.UprobeSymbol
	case AttachUretprobe:
		return "uretprobe:" + pa.UprobePath + ":" + pa.UprobeSymbol
	default:
		return ""
	}
}
