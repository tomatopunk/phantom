// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
)

// MapDesc describes a BPF map in a loaded hook collection.
type MapDesc struct {
	Name       string
	TypeName   string
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
}

// MapEntryBytes is one key/value pair from a map read.
type MapEntryBytes struct {
	Key   []byte
	Value []byte
}

var (
	errHookNotFound   = errors.New("hook not found")
	errNoCollection   = errors.New("hook has no live collection")
	errMapNotFound    = errors.New("map not found")
	errMapUnsupported = errors.New("map type not supported for read")
)

// ListHookMapDescriptors returns metadata for all maps in the hook's collection (caller must not mutate maps).
func (s *Session) ListHookMapDescriptors(hookID string) ([]MapDesc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.hooks[hookID]
	if !ok {
		return nil, errHookNotFound
	}
	if h.Coll == nil {
		return nil, errNoCollection
	}
	var out []MapDesc
	for n, m := range h.Coll.Maps {
		info, err := m.Info()
		if err != nil {
			continue
		}
		out = append(out, MapDesc{
			Name:       n,
			TypeName:   m.Type().String(),
			KeySize:    uint32(info.KeySize),
			ValueSize:  uint32(info.ValueSize),
			MaxEntries: info.MaxEntries,
		})
	}
	return out, nil
}

const defaultMaxMapReadEntries = 256

// ReadHookMapEntries reads up to maxEntries key/value pairs from a map (hash-like or array).
func (s *Session) ReadHookMapEntries(hookID, mapName string, maxEntries uint32) ([]MapEntryBytes, error) {
	if maxEntries == 0 {
		maxEntries = defaultMaxMapReadEntries
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.hooks[hookID]
	if !ok {
		return nil, errHookNotFound
	}
	if h.Coll == nil {
		return nil, errNoCollection
	}
	m, ok := h.Coll.Maps[mapName]
	if !ok {
		return nil, errMapNotFound
	}
	switch m.Type() {
	case ebpf.Hash, ebpf.LRUHash, ebpf.LRUCPUHash:
		return readHashMapEntries(m, maxEntries)
	case ebpf.Array, ebpf.PerCPUArray:
		return readArrayMapEntries(m, maxEntries)
	default:
		return nil, fmt.Errorf("%w: %s", errMapUnsupported, m.Type().String())
	}
}

func readHashMapEntries(m *ebpf.Map, maxEntries uint32) ([]MapEntryBytes, error) {
	var out []MapEntryBytes
	iter := m.Iterate()
	var key, val []byte
	for iter.Next(&key, &val) {
		if uint32(len(out)) >= maxEntries {
			break
		}
		kc := append([]byte(nil), key...)
		vc := append([]byte(nil), val...)
		out = append(out, MapEntryBytes{Key: kc, Value: vc})
	}
	if err := iter.Err(); err != nil {
		return out, err
	}
	return out, nil
}

func readArrayMapEntries(m *ebpf.Map, maxEntries uint32) ([]MapEntryBytes, error) {
	info, err := m.Info()
	if err != nil {
		return nil, err
	}
	valBuf := make([]byte, info.ValueSize)
	keyBuf := make([]byte, info.KeySize)
	var out []MapEntryBytes
	for i := uint32(0); i < info.MaxEntries && uint32(len(out)) < maxEntries; i++ {
		if err := m.Lookup(i, valBuf); err != nil {
			continue
		}
		for j := range keyBuf {
			keyBuf[j] = 0
		}
		if info.KeySize >= 4 {
			binary.LittleEndian.PutUint32(keyBuf[:4], i)
		}
		vc := append([]byte(nil), valBuf...)
		kc := append([]byte(nil), keyBuf...)
		out = append(out, MapEntryBytes{Key: kc, Value: vc})
	}
	return out, nil
}
