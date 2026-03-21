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

// Package hook: prologue registry for hook compile.
//
// Prologues are C fragments injected before the user --sec/--code snippet per symbol.
// To extend: call RegisterPrologue(symbol, PrologueSpec{Prologue: "…", ExtraFields: []string{…}})
// after built-ins load (built-ins register on first GetPrologue/RegisterPrologue). To add a new built-in, add a func like
// registerSocketPrologue() and call it inside RegisterBuiltinPrologues. To override or
// edit the socket prologue, change registerSocketPrologue in this file.
package hook

import (
	"strings"
	"sync"
)

// PrologueSpec describes optional C code injected before the user snippet for a given symbol,
// and the extra --sec field names that become available (e.g. sport, dport for TCP).
type PrologueSpec struct {
	// Prologue is the C fragment injected after arg0..arg5 are set (no leading/trailing newline required).
	Prologue string
	// ExtraFields are the additional --sec field names this prologue provides (e.g. "sport", "dport", "saddr", "daddr").
	// They are only allowed for this symbol when parsing --sec.
	ExtraFields []string
}

var (
	prologueMu  sync.RWMutex
	prologueReg = make(map[string]PrologueSpec)
	builtinOnce sync.Once
)

// storePrologue stores spec under symbol (normalized); acquires prologueMu.
func storePrologue(symbol string, spec PrologueSpec) {
	symbol = strings.TrimSpace(strings.ToLower(symbol))
	if symbol == "" {
		return
	}
	prologueMu.Lock()
	defer prologueMu.Unlock()
	prologueReg[symbol] = spec
}

// RegisterPrologue registers a prologue for the given kprobe symbol (e.g. "tcp_sendmsg").
// Symbol is normalized to lowercase. Re-registering the same symbol overwrites.
func RegisterPrologue(symbol string, spec PrologueSpec) {
	RegisterBuiltinPrologues()
	storePrologue(symbol, spec)
}

// GetPrologue returns the prologue spec for the symbol, if any.
func GetPrologue(symbol string) (PrologueSpec, bool) {
	RegisterBuiltinPrologues()
	prologueMu.RLock()
	defer prologueMu.RUnlock()
	symbol = strings.TrimSpace(strings.ToLower(symbol))
	spec, ok := prologueReg[symbol]
	return spec, ok
}

// PrologueC returns the C code to inject for the symbol, or empty string if none.
func PrologueC(symbol string) string {
	spec, ok := GetPrologue(symbol)
	if !ok {
		return ""
	}
	return spec.Prologue
}

// ExtraFieldsForSymbol returns the extra --sec field names for the symbol, if any.
func ExtraFieldsForSymbol(symbol string) []string {
	spec, ok := GetPrologue(symbol)
	if !ok {
		return nil
	}
	return spec.ExtraFields
}

// RegisterBuiltinPrologues registers built-in prologues (e.g. socket four-tuple for tcp_sendmsg/tcp_recvmsg).
// Safe to call multiple times; runs only once.
func RegisterBuiltinPrologues() {
	builtinOnce.Do(func() {
		registerSocketPrologue()
	})
}

func registerSocketPrologue() {
	const socketPrologue = `
	/* CO-RE: sport/dport/saddr/daddr for --sec (no fixed struct offsets) */
	struct sock *sk = (void *)arg0;
	__u16 sport = 0, dport = 0;
	__u32 saddr = 0, daddr = 0;
	if (sk) {
		__be32 saddr_be = 0, daddr_be = 0;
		__u64 skc_num_raw = BPF_CORE_READ_BITFIELD_PROBED(&sk->__sk_common, skc_num);
		__u64 skc_dport_raw = BPF_CORE_READ_BITFIELD_PROBED(&sk->__sk_common, skc_dport);
		BPF_CORE_READ_INTO(&saddr_be, sk, __sk_common.skc_rcv_saddr);
		BPF_CORE_READ_INTO(&daddr_be, sk, __sk_common.skc_daddr);
		sport = __builtin_bswap16((__u16)skc_num_raw);
		dport = __builtin_bswap16((__u16)skc_dport_raw);
		saddr = __builtin_bswap32(saddr_be);
		daddr = __builtin_bswap32(daddr_be);
	}
`
	spec := PrologueSpec{
		Prologue:    socketPrologue,
		ExtraFields: []string{"sport", "dport", "saddr", "daddr"},
	}
	// Use storePrologue only: RegisterPrologue would call RegisterBuiltinPrologues and deadlock inside Once.
	storePrologue("tcp_sendmsg", spec)
	storePrologue("tcp_recvmsg", spec)
}
