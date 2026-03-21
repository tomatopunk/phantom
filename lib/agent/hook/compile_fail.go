package hook

import (
	"errors"
	"fmt"
)

// CompileFailed is returned when clang exits non-zero. Stderr holds combined compiler output.
type CompileFailed struct {
	Stderr []byte
	Err    error
}

func (e *CompileFailed) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("compile: %v\n%s", e.Err, e.Stderr)
}

func (e *CompileFailed) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// AsCompileFailed returns (*CompileFailed, true) if err wraps CompileFailed.
func AsCompileFailed(err error) (*CompileFailed, bool) {
	var cf *CompileFailed
	if errors.As(err, &cf) {
		return cf, true
	}
	return nil, false
}
