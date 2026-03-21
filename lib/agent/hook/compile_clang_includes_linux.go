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

package hook

import (
	"os"
	"path/filepath"
	"runtime"
)

// hostClangBPFExtraIncludes returns system -I paths so <linux/*.h> → <asm/types.h> resolves under
// Debian/Ubuntu multiarch (same as Makefile CLANG_FLAGS: -I /usr/include/$(uname -m)-linux-gnu -I /usr/include).
func hostClangBPFExtraIncludes() []string {
	var out []string
	if sys := archLinuxUserIncludeDir(); sys != "" {
		asm := filepath.Join(sys, "asm")
		if isDir(asm) {
			out = append(out, sys)
		}
	}
	if isDir("/usr/include/bpf") {
		out = append(out, "/usr/include")
	}
	return out
}

func archLinuxUserIncludeDir() string {
	switch runtime.GOARCH {
	case "amd64":
		return "/usr/include/x86_64-linux-gnu"
	case "arm64":
		return "/usr/include/aarch64-linux-gnu"
	case "386":
		return "/usr/include/i386-linux-gnu"
	case "arm":
		return "/usr/include/arm-linux-gnueabihf"
	case "ppc64le":
		return "/usr/include/powerpc64le-linux-gnu"
	case "riscv64":
		return "/usr/include/riscv64-linux-gnu"
	case "s390x":
		return "/usr/include/s390x-linux-gnu"
	default:
		return ""
	}
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
