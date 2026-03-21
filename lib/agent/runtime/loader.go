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
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
)

const (
	mapNameEvents  = "events"
	progNameKprobe = "kprobe_handler"
	progNameUprobe = "uprobe_handler"
)

// LoadFromFile loads the eBPF collection from a compiled .o file (Linux, built with clang).
func (r *Runtime) LoadFromFile(path string) error {
	if r.collection != nil {
		return fmt.Errorf("runtime: already loaded")
	}
	spec, err := ebpf.LoadCollectionSpec(path)
	if err != nil {
		return fmt.Errorf("load spec %s: %w", path, err)
	}
	if kerr := FillKprobeKernelVersionsFromUname(spec); kerr != nil {
		return fmt.Errorf("kprobe kernel version: %w", kerr)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("new collection: %w", err)
	}
	r.collection = coll
	return nil
}

// OpenEventReader opens a ring buffer reader for the events map (call after LoadFromFile).
func (r *Runtime) OpenEventReader() (*ringbuf.Reader, error) {
	if r.collection == nil {
		return nil, fmt.Errorf("runtime: load first")
	}
	m, ok := r.collection.Maps[mapNameEvents]
	if !ok {
		return nil, fmt.Errorf("runtime: map %q not found", mapNameEvents)
	}
	return ringbuf.NewReader(m)
}

// AttachKprobe attaches the loaded kprobe program to the given kernel symbol.
func (r *Runtime) AttachKprobe(symbol string) (detach func(), err error) {
	if r.collection == nil {
		return nil, fmt.Errorf("runtime: load first")
	}
	prog, ok := r.collection.Programs[progNameKprobe]
	if !ok {
		return nil, fmt.Errorf("runtime: program %q not found", progNameKprobe)
	}
	link, err := attachKprobe(prog, symbol)
	if err != nil {
		return nil, err
	}
	return func() { link.Close() }, nil
}

// AttachUprobe attaches the loaded uprobe program to the given user binary and symbol.
func (r *Runtime) AttachUprobe(binaryPath, symbol string) (detach func(), err error) {
	if r.uprobeCollection == nil {
		return nil, fmt.Errorf("runtime: load uprobe first (LoadUprobeFromFile)")
	}
	prog, ok := r.uprobeCollection.Programs[progNameUprobe]
	if !ok {
		return nil, fmt.Errorf("runtime: program %q not found", progNameUprobe)
	}
	link, err := attachUprobe(prog, binaryPath, symbol)
	if err != nil {
		return nil, err
	}
	return func() { link.Close() }, nil
}

// AttachUretprobe attaches the loaded uprobe program as a return probe.
func (r *Runtime) AttachUretprobe(binaryPath, symbol string) (detach func(), err error) {
	if r.uprobeCollection == nil {
		return nil, fmt.Errorf("runtime: load uprobe first (LoadUprobeFromFile)")
	}
	prog, ok := r.uprobeCollection.Programs[progNameUprobe]
	if !ok {
		return nil, fmt.Errorf("runtime: program %q not found", progNameUprobe)
	}
	link, err := attachUretprobe(prog, binaryPath, symbol)
	if err != nil {
		return nil, err
	}
	return func() { link.Close() }, nil
}

// LoadUprobeFromFile loads the uprobe eBPF collection from a compiled .o file.
func (r *Runtime) LoadUprobeFromFile(path string) error {
	if r.uprobeCollection != nil {
		return fmt.Errorf("runtime: uprobe already loaded")
	}
	spec, err := ebpf.LoadCollectionSpec(path)
	if err != nil {
		return fmt.Errorf("load uprobe spec %s: %w", path, err)
	}
	if kerr := FillKprobeKernelVersionsFromUname(spec); kerr != nil {
		return fmt.Errorf("kprobe kernel version: %w", kerr)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("new uprobe collection: %w", err)
	}
	r.uprobeCollection = coll
	return nil
}

// Close releases the eBPF collections and any readers.
func (r *Runtime) Close() error {
	if r.uprobeCollection != nil {
		r.uprobeCollection.Close()
		r.uprobeCollection = nil
	}
	if r.collection != nil {
		r.collection.Close()
		r.collection = nil
	}
	return nil
}
