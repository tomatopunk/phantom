package hook

import (
	"fmt"
	"strconv"
	"strings"
)

// Allowed field names for --sec (first version: == and != only).
var secAllowedFields = map[string]string{
	"pid": "ev.pid", "tgid": "ev.tgid", "cpu": "ev.cpu",
	"arg0": "arg0", "arg1": "arg1", "arg2": "arg2", "arg3": "arg3", "arg4": "arg4", "arg5": "arg5",
	"ret": "ret",
}

// SecToSnippet converts a simple condition expression (field==value or field!=value) into a C snippet.
// Supported fields: pid, tgid, cpu, arg0..arg5, ret. Value must be a decimal integer.
// The snippet returns 0 (no event) when the condition fails, so only matching events are submitted.
func SecToSnippet(sec string) (string, error) {
	sec = strings.TrimSpace(sec)
	if sec == "" {
		return "", fmt.Errorf("--sec expression is empty")
	}
	op := ""
	if strings.Contains(sec, "==") {
		op = "=="
	} else if strings.Contains(sec, "!=") {
		op = "!="
	} else {
		return "", fmt.Errorf("--sec must be field==value or field!=value (e.g. pid==123)")
	}
	parts := strings.SplitN(sec, op, 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("--sec must be field==value or field!=value")
	}
	field := strings.TrimSpace(strings.ToLower(parts[0]))
	valueStr := strings.TrimSpace(parts[1])
	cExpr, ok := secAllowedFields[field]
	if !ok {
		return "", fmt.Errorf("--sec unknown field %q (allowed: pid, tgid, cpu, arg0..arg5, ret)", field)
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("--sec value must be decimal integer: %w", err)
	}
	// Generate: if (condition fails) return 0;
	// For "pid==123" we want to skip when pid != 123 -> if (ev.pid != 123) return 0;
	// For "pid!=123" we want to skip when pid == 123 -> if (ev.pid == 123) return 0;
	var cond string
	if op == "==" {
		cond = fmt.Sprintf("%s != %d", cExpr, value)
	} else {
		cond = fmt.Sprintf("%s == %d", cExpr, value)
	}
	return "if (" + cond + ") return 0;\n\t", nil
}
