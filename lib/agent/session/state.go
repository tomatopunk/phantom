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

package session

import "github.com/cilium/ebpf"

// BreakpointState holds one template-backed breakpoint (catalog probe_id + optional kernel filter DSL).
type BreakpointState struct {
	ID               string
	ProbeID          string // catalog id (same as user-facing symbol)
	Symbol           string // display; equals ProbeID
	Enabled          bool
	IsTemp           bool
	Condition        string // optional user-side expression on BREAK_HIT
	HookID           string // backing hook; empty when disabled
	KprobeHook       bool   // always true for template breaks
	KernelFilterExpr string // kernel predicate DSL saved for re-enable
	HookEventLimit   int    // hook auto-remove limit (0 = none; tbreak uses 1)
}

// HookState holds one C hook's probe_point, detach, cancel, and optional hit limit.
type HookState struct {
	ID       string
	ProbePoint string // e.g. kprobe:do_sys_open
	Detach   func()
	Cancel   func() // cancels the hook's event pump context so reader is closed before detach
	Limit    int    // 0 = no limit; when HitCount >= Limit the hook is auto-removed
	HitCount int    // incremented on each event; used when Limit > 0
	Note     string // origin label e.g. CompileAndAttach
	// Coll is the live collection until Detach closes it; used for ListHookMaps / ReadHookMap.
	Coll *ebpf.Collection
}

// WatchState holds arg-column watch registration for a catalog probe_id (fires on matching BREAK_HIT).
type WatchState struct {
	ID              string
	ProbeID         string // catalog probe_id
	ArgParamIndices []int  // indices into catalog Params; empty means use catalog defaults at fire time
}
