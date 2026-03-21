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

import "testing"

func TestParseClangDiagnostics(t *testing.T) {
	stderr := `program.c:10:5: error: unknown type name 'oops'
program.c:9:1: note: to match this '('
`
	diags := ParseClangDiagnostics(stderr)
	if len(diags) != 2 {
		t.Fatalf("got %d diags, want 2", len(diags))
	}
	if diags[0].Path != "program.c" || diags[0].Line != 10 || diags[0].Column != 5 || diags[0].Severity != "error" {
		t.Fatalf("first diag: %+v", diags[0])
	}
	if diags[1].Severity != "note" {
		t.Fatalf("second severity: %q", diags[1].Severity)
	}
}
