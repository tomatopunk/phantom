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
	"sort"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

func findRingBufMap(coll *ebpf.Collection) (*ebpf.Map, error) {
	for _, m := range coll.Maps {
		if m.Type() == ebpf.RingBuf {
			return m, nil
		}
	}
	return nil, fmt.Errorf("no BPF_MAP_TYPE_RINGBUF map in object (need an events ringbuf map)")
}

func pickProgram(coll *ebpf.Collection, programName string, preferType ebpf.ProgramType) (*ebpf.Program, error) {
	if programName != "" {
		p, ok := coll.Programs[programName]
		if !ok {
			return nil, fmt.Errorf("program %q not found in object", programName)
		}
		return p, nil
	}
	var names []string
	for n := range coll.Programs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := coll.Programs[n]
		info, err := p.Info()
		if err != nil {
			continue
		}
		if info.Type == preferType {
			return p, nil
		}
	}
	if len(names) > 0 {
		return coll.Programs[names[0]], nil
	}
	return nil, fmt.Errorf("no BPF programs in object")
}

func preferredProgramType(pa *ParsedAttach) ebpf.ProgramType {
	switch pa.Kind {
	case AttachTracepoint:
		return ebpf.TracePoint
	case AttachUprobe, AttachUretprobe:
		return ebpf.Kprobe // many uprobe SECs load as Kprobe-type programs
	default:
		return ebpf.Kprobe
	}
}

// AttachProbeFromObject loads an ELF .o, picks a program, attaches per ParsedAttach, opens the first ringbuf map.
func AttachProbeFromObject(objectPath string, pa *ParsedAttach, programName string, cleanup func()) (detach func(), reader *ringbuf.Reader, err error) {
	spec, err := ebpf.LoadCollectionSpec(objectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load spec: %w", err)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("new collection: %w", err)
	}
	prog, err := pickProgram(coll, programName, preferredProgramType(pa))
	if err != nil {
		coll.Close()
		return nil, nil, err
	}
	var lk link.Link
	switch pa.Kind {
	case AttachKprobe:
		lk, err = link.Kprobe(pa.KprobeSymbol, prog, nil)
	case AttachTracepoint:
		lk, err = link.Tracepoint(pa.TraceGroup, pa.TraceEvent, prog, nil)
	case AttachUprobe:
		var ex *link.Executable
		ex, err = link.OpenExecutable(pa.UprobePath)
		if err == nil {
			lk, err = ex.Uprobe(pa.UprobeSymbol, prog, nil)
		}
	case AttachUretprobe:
		var ex *link.Executable
		ex, err = link.OpenExecutable(pa.UprobePath)
		if err == nil {
			lk, err = ex.Uretprobe(pa.UprobeSymbol, prog, nil)
		}
	default:
		err = fmt.Errorf("internal: unknown attach kind")
	}
	if err != nil {
		coll.Close()
		return nil, nil, fmt.Errorf("attach: %w", err)
	}
	m, err := findRingBufMap(coll)
	if err != nil {
		_ = lk.Close()
		coll.Close()
		return nil, nil, err
	}
	rd, err := ringbuf.NewReader(m)
	if err != nil {
		_ = lk.Close()
		coll.Close()
		return nil, nil, fmt.Errorf("ringbuf reader: %w", err)
	}
	detachFn := func() {
		_ = lk.Close()
		coll.Close()
		if cleanup != nil {
			cleanup()
		}
	}
	return detachFn, rd, nil
}
