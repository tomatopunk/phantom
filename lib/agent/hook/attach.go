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
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
)

// HookTemplateProgramName is the BPF function name in embed/hook.c after template expansion.
const HookTemplateProgramName = "hook_handler"

// AttachKprobeFromObject loads the compiled .o, attaches the kprobe to symbol, and opens a ringbuf reader for the hook's events map.
// The caller must run a pump reading from the returned reader and broadcast events to the session; when done, call detach().
// detach does not close the reader (the pump should close it); detach closes the link, collection, and runs cleanup.
func AttachKprobeFromObject(objectPath, symbol string, cleanup func()) (detach func(), reader *ringbuf.Reader, coll *ebpf.Collection, err error) {
	pa := &ParsedAttach{Kind: AttachKprobe, KprobeSymbol: symbol}
	return AttachProbeFromObject(objectPath, pa, HookTemplateProgramName, cleanup)
}
