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
