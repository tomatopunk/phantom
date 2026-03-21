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

import "testing"

// kernelVersionEncoding is the same layout as Linux KERNEL_VERSION / parseKernelRelease output.
func kernelVersionEncoding(major, minor, patch uint8) uint32 {
	return uint32(major)<<16 | uint32(minor)<<8 | uint32(patch)
}

func TestParseKernelRelease(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rel          string
		major, minor uint8
		patch        uint8
	}{
		{"6.8.0-71-generic", 6, 8, 0},
		{"5.15-1-amd64", 5, 15, 0},
		{"5.4.250", 5, 4, 250},
		{"4.19.0", 4, 19, 0},
	}
	for _, tc := range cases {
		want := kernelVersionEncoding(tc.major, tc.minor, tc.patch)
		got, err := parseKernelRelease(tc.rel)
		if err != nil {
			t.Fatalf("parseKernelRelease(%q): %v", tc.rel, err)
		}
		if got != want {
			t.Fatalf("parseKernelRelease(%q) = %#x want %#x (%d.%d.%d)",
				tc.rel, got, want, tc.major, tc.minor, tc.patch)
		}
	}
}

func TestParseKernelRelease_clampsPatchTo255(t *testing.T) {
	t.Parallel()
	got, err := parseKernelRelease("3.10.999")
	if err != nil {
		t.Fatal(err)
	}
	want := kernelVersionEncoding(3, 10, 255)
	if got != want {
		t.Fatalf("parseKernelRelease(%q) = %#x want %#x (patch clamp)", "3.10.999", got, want)
	}
}
