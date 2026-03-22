// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateBreakSymbol ensures the token is suitable for the built-in kprobe breakpoint template only.
// Tracepoints, uprobes, and explicit attach prefixes must use hook attach (full C) instead.
func ValidateBreakSymbol(symbol string) error {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return fmt.Errorf("empty symbol")
	}
	if strings.ContainsAny(s, ":/") {
		return fmt.Errorf("break only accepts a bare kernel symbol (built-in kprobe template); use hook attach for tracepoint:, uprobe:, or paths")
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			return fmt.Errorf("break symbol must be a single token")
		}
	}
	return nil
}
