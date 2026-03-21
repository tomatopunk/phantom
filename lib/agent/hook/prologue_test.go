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
