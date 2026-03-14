package hook

import (
	"strings"
	"testing"
)

func TestSecToSnippet(t *testing.T) {
	tests := []struct {
		sec     string
		want    string
		wantErr bool
	}{
		{"pid==123", "if (ev.pid != 123) return 0;\n\t", false},
		{"pid!=123", "if (ev.pid == 123) return 0;\n\t", false},
		{"tgid==456", "if (ev.tgid != 456) return 0;\n\t", false},
		{"cpu==0", "if (ev.cpu != 0) return 0;\n\t", false},
		{"arg0==80", "if (arg0 != 80) return 0;\n\t", false},
		{"arg1!=22", "if (arg1 == 22) return 0;\n\t", false},
		{"arg5==0", "if (arg5 != 0) return 0;\n\t", false},
		{"ret==0", "if (ret != 0) return 0;\n\t", false},
		{"  pid==1  ", "if (ev.pid != 1) return 0;\n\t", false},
		{"PID==1", "if (ev.pid != 1) return 0;\n\t", false},
		{"", "", true},
		{"pid", "", true},
		{"pid=123", "", true},
		{"pid<>123", "", true},
		{"unknown==1", "", true},
		{"pid==abc", "", true},
		{"pid==1x", "", true},
	}
	for _, tt := range tests {
		got, err := SecToSnippet(tt.sec)
		if (err != nil) != tt.wantErr {
			t.Errorf("SecToSnippet(%q): err=%v wantErr=%v", tt.sec, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("SecToSnippet(%q): got %q want %q", tt.sec, got, tt.want)
		}
	}
}

func TestSecToSnippet_AllowedFields(t *testing.T) {
	fields := []string{"pid", "tgid", "cpu", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5", "ret"}
	for _, f := range fields {
		_, err := SecToSnippet(f + "==0")
		if err != nil {
			t.Errorf("SecToSnippet(%s==0): %v", f, err)
		}
		_, err = SecToSnippet(f + "!=1")
		if err != nil {
			t.Errorf("SecToSnippet(%s!=1): %v", f, err)
		}
	}
}

func TestSecToSnippet_InvalidValue(t *testing.T) {
	_, err := SecToSnippet("pid==0x10")
	if err == nil {
		t.Error("SecToSnippet(pid==0x10): want error (hex not allowed)")
	}
	if err != nil && !strings.Contains(err.Error(), "decimal") {
		t.Errorf("SecToSnippet(pid==0x10): want 'decimal' in error, got %q", err.Error())
	}
}
