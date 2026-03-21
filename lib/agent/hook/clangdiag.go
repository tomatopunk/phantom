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
	"regexp"
	"strconv"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
)

// clangDiagLine matches: path:line:col: severity: message
// Examples: program.c:12:3: error: expected ';' after expression
var clangDiagLine = regexp.MustCompile(`^(.+):(\d+):(\d+):\s+(error|warning|note|fatal error|remark):\s+(.+)$`)

// ParseClangDiagnostics extracts structured diagnostics from clang stderr.
// Lines that do not match (caret lines, include stacks) are skipped as primary rows
// but preserved in compiler_output on the wire.
func ParseClangDiagnostics(stderr string) []*proto.CompileDiagnostic {
	var out []*proto.CompileDiagnostic
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		m := clangDiagLine.FindStringSubmatch(line)
		if len(m) != 6 {
			continue
		}
		ln, _ := strconv.ParseInt(m[2], 10, 32)
		col, _ := strconv.ParseInt(m[3], 10, 32)
		sev := strings.ToLower(strings.TrimSpace(m[4]))
		switch {
		case strings.HasPrefix(sev, "fatal"):
			sev = "fatal"
		case sev == "fatal error":
			sev = "fatal"
		}
		out = append(out, &proto.CompileDiagnostic{
			Path:     m[1],
			Line:     int32(ln), //nolint:gosec // G115
			Column:   int32(col), //nolint:gosec // G115
			Severity: sev,
			Message:  m[5],
		})
	}
	return out
}
