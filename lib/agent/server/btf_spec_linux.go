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

	"github.com/cilium/ebpf/btf"
)

// loadExecutorBTF loads kernel BTF for CO-RE and optional type queries.
// Tries /sys/kernel/btf/vmlinux first, then optional vmlinux ELF path.
func loadExecutorBTF(vmlinuxPath string) *btf.Spec {
	spec, err := btf.LoadKernelSpec()
	if err == nil {
		return spec
	}
	log.Printf("phantom: kernel BTF unavailable (%v); CO-RE compile may fail on this host", err)
	if vmlinuxPath == "" {
		return nil
	}
	fallback, err := btf.LoadSpec(vmlinuxPath)
	if err != nil {
		log.Printf("phantom: load BTF from vmlinux %q: %v", vmlinuxPath, err)
		return nil
	}
	return fallback
}
