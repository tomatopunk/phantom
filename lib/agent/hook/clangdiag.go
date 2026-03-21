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
