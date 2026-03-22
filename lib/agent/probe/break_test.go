// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package probe

import "testing"

func TestValidateBreakSymbol(t *testing.T) {
	if err := ValidateBreakSymbol("do_sys_open"); err != nil {
		t.Errorf("do_sys_open: %v", err)
	}
	for _, bad := range []string{"", " ", "tracepoint:sched:foo", "uprobe:/bin/sh:main", "a:b", "a/b"} {
		if err := ValidateBreakSymbol(bad); err == nil {
			t.Errorf("ValidateBreakSymbol(%q): want error", bad)
		}
	}
}
