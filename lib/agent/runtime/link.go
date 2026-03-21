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
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// attachKprobe attaches prog to the kernel symbol and returns a link that can be closed to detach.
func attachKprobe(prog *ebpf.Program, symbol string) (link.Link, error) {
	return link.Kprobe(symbol, prog, nil)
}

// attachUprobe attaches prog to the user binary at the given symbol (resolved by link.OpenExecutable).
func attachUprobe(prog *ebpf.Program, binaryPath, symbol string) (link.Link, error) {
	ex, err := link.OpenExecutable(binaryPath)
	if err != nil {
		return nil, err
	}
	return ex.Uprobe(symbol, prog, nil)
}

// attachUretprobe attaches prog as a uretprobe at the given symbol.
func attachUretprobe(prog *ebpf.Program, binaryPath, symbol string) (link.Link, error) {
	ex, err := link.OpenExecutable(binaryPath)
	if err != nil {
		return nil, err
	}
	return ex.Uretprobe(symbol, prog, nil)
}
