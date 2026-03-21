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
	"strings"
	"testing"
)

func TestRegisterGetPrologue(t *testing.T) {
	// Use a custom symbol so we don't rely on builtins
	RegisterPrologue("custom_sym", PrologueSpec{
		Prologue:    "\n\tint x = 0;\n",
		ExtraFields: []string{"foo", "bar"},
	})
	spec, ok := GetPrologue("custom_sym")
	if !ok {
		t.Fatal("GetPrologue(custom_sym): want ok true")
	}
	if !strings.Contains(spec.Prologue, "int x = 0") {
		t.Errorf("GetPrologue: prologue %q", spec.Prologue)
	}
	if len(spec.ExtraFields) != 2 || spec.ExtraFields[0] != "foo" || spec.ExtraFields[1] != "bar" {
		t.Errorf("GetPrologue: extra fields %v", spec.ExtraFields)
	}
	_, ok = GetPrologue("nonexistent")
	if ok {
		t.Error("GetPrologue(nonexistent): want ok false")
	}
}

func TestPrologueC_ExtraFieldsForSymbol_AfterBuiltin(t *testing.T) {
	RegisterBuiltinPrologues()
	c := PrologueC("tcp_sendmsg")
	if c == "" {
		t.Error("PrologueC(tcp_sendmsg): want non-empty after builtin")
	}
	if !strings.Contains(c, "sport") {
		t.Errorf("PrologueC(tcp_sendmsg): want sport in prologue, got %q", c)
	}
	fields := ExtraFieldsForSymbol("tcp_sendmsg")
	if len(fields) != 4 {
		t.Errorf("ExtraFieldsForSymbol(tcp_sendmsg): want 4 fields, got %v", fields)
	}
}
