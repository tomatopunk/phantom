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

package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cilium/ebpf"
	"golang.org/x/sys/unix"
)

// magicKernelVersion matches github.com/cilium/ebpf/internal.MagicKernelVersion: loader must substitute
// KERNEL_VERSION; cilium does that by reading vDSO via /proc/self/mem, which fails on hardened runners.
const (
	magicKernelVersion   = 0xFFFFFFFE
	maxKernelVersionByte = 255
)

// FillKprobeKernelVersionsFromUname sets ProgramSpec.KernelVersion for Kprobes that still use the ELF
// placeholder (0 or magic). Uses uname(2) only — no /proc/self/mem — so loading works without CAP_SYS_PTRACE.
func FillKprobeKernelVersionsFromUname(spec *ebpf.CollectionSpec) error {
	if spec == nil || len(spec.Programs) == 0 {
		return nil
	}
	kv, err := kernelVersionCodeFromUname()
	if err != nil {
		return err
	}
	for _, prog := range spec.Programs {
		if prog == nil || prog.Type != ebpf.Kprobe {
			continue
		}
		if prog.KernelVersion != 0 && prog.KernelVersion != magicKernelVersion {
			continue
		}
		prog.KernelVersion = kv
	}
	return nil
}

func kernelVersionCodeFromUname() (uint32, error) {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return 0, fmt.Errorf("uname: %w", err)
	}
	rel := unix.ByteSliceToString(u.Release[:])
	return parseKernelRelease(rel)
}

// parseKernelRelease turns e.g. "6.8.0-71-generic" into the kernel's KERNEL_VERSION encoding.
func parseKernelRelease(rel string) (uint32, error) {
	rel = strings.TrimSpace(rel)
	if i := strings.IndexByte(rel, '-'); i >= 0 {
		rel = rel[:i]
	}
	parts := strings.Split(rel, ".")
	if len(parts) < 2 {
		return 0, fmt.Errorf("kernel release %q: expected major.minor", rel)
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("kernel release %q: %w", rel, err)
	}
	minorVer, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("kernel release %q: %w", rel, err)
	}
	patch := 0
	if len(parts) >= 3 {
		patch = leadingDigitsInt(parts[2])
	}
	if maj < 0 || maj > maxKernelVersionByte || minorVer < 0 || minorVer > maxKernelVersionByte || patch < 0 {
		return 0, fmt.Errorf("kernel release %q: version out of uint8 range", rel)
	}
	if patch > maxKernelVersionByte {
		patch = maxKernelVersionByte
	}
	return uint32(uint8(maj))<<16 | uint32(uint8(minorVer))<<8 | uint32(uint8(patch)), nil
}

func leadingDigitsInt(s string) int {
	if s == "" {
		return 0
	}
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	n, _ := strconv.Atoi(s[:end])
	return n
}
