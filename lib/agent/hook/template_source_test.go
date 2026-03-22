// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package hook

import (
	"strings"
	"testing"
)

func TestBuildTemplateSource_kprobe(t *testing.T) {
	s, err := BuildTemplateSource("\treturn 0;\n", "kprobe:do_sys_open")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, `SEC("kprobe")`) {
		t.Fatalf("want SEC kprobe in source, got prefix %q", truncate(s, 120))
	}
	if !strings.Contains(s, "return 0") {
		t.Fatal("snippet not embedded")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
