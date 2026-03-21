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
