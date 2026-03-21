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

//go:build linux

package server

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cilium/ebpf/btf"
)

// kernelRelease returns the running kernel release (e.g. "6.6.0-amd64"), or empty on error.
func kernelRelease() string {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// vmlinuxBTFSearchCandidates returns ordered paths to try for BTF embedded in a vmlinux ELF.
// userPath is tried first (from -vmlinux / PHANTOM_VMLINUX); then common distro / self-build locations.
func vmlinuxBTFSearchCandidates(userPath, release string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if p == "" || p == "." {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(userPath)
	if release == "" {
		release = kernelRelease()
	}
	if release != "" {
		add("/boot/vmlinux-" + release)
		add("/usr/lib/debug/boot/vmlinux-" + release)
		add("/lib/modules/" + release + "/build/vmlinux")
	}
	return out
}

// loadExecutorBTF loads kernel BTF for CO-RE (hook compile) and optional type queries.
// Order: (1) kernel BTF via btf.LoadKernelSpec() (/sys/kernel/btf/vmlinux), (2) -vmlinux path if set,
// (3) well-known vmlinux ELF paths for the running release (self-built / debug packages).
//
// On success loading from an ELF file, the second return value is that path so the agent can use it
// for `list` disassembly when the user did not pass -vmlinux explicitly.
func loadExecutorBTF(vmlinuxPath string) (spec *btf.Spec, elfPath string) {
	spec, err := btf.LoadKernelSpec()
	if err == nil {
		return spec, ""
	}
	log.Printf("phantom: kernel BTF unavailable (%v); trying vmlinux ELF BTF fallback (see docs/vmlinux.md)", err)

	for _, p := range vmlinuxBTFSearchCandidates(vmlinuxPath, kernelRelease()) {
		if _, stErr := os.Stat(p); stErr != nil {
			continue
		}
		fallback, lerr := btf.LoadSpec(p)
		if lerr == nil {
			log.Printf("phantom: loaded kernel BTF from %q", p)
			return fallback, p
		}
		log.Printf("phantom: BTF load %q: %v", p, lerr)
	}
	if vmlinuxPath != "" {
		log.Printf("phantom: could not load BTF from -vmlinux %q or standard paths; hook CO-RE may fail", vmlinuxPath)
	}
	return nil, ""
}
