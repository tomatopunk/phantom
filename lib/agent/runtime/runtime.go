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

import "github.com/cilium/ebpf"

// Runtime loads eBPF programs, manages maps, and consumes ring buffer events.
type Runtime struct {
	collection       *ebpf.Collection // kprobe object
	uprobeCollection *ebpf.Collection // optional uprobe object
}

// New returns a new eBPF runtime (call LoadFromFile then AttachKprobe / OpenEventReader).
func New() *Runtime {
	return &Runtime{}
}
