package hook

import (
	"strings"
	"testing"
)

const attachGeneric = "kprobe:do_sys_open"
const attachTcpSend = "kprobe:tcp_sendmsg"
const attachTcpRecv = "kprobe:tcp_recvmsg"

func TestSecToSnippet_BackwardCompat(t *testing.T) {
	tests := []struct {
		sec     string
		point   string
		want    string
		wantErr bool
	}{
		{"pid==123", attachGeneric, "if (!(ev.pid == 123)) return 0;\n\t", false},
		{"pid!=123", attachGeneric, "if (!(ev.pid != 123)) return 0;\n\t", false},
		{"tgid==456", attachGeneric, "if (!(ev.tgid == 456)) return 0;\n\t", false},
		{"cpu==0", attachGeneric, "if (!(ev.cpu == 0)) return 0;\n\t", false},
		{"arg0==80", attachGeneric, "if (!(arg0 == 80)) return 0;\n\t", false},
		{"arg1!=22", attachGeneric, "if (!(arg1 != 22)) return 0;\n\t", false},
		{"arg5==0", attachGeneric, "if (!(arg5 == 0)) return 0;\n\t", false},
		{"ret==0", attachGeneric, "if (!(ret == 0)) return 0;\n\t", false},
		{"  pid==1  ", attachGeneric, "if (!(ev.pid == 1)) return 0;\n\t", false},
		{"PID==1", attachGeneric, "if (!(ev.pid == 1)) return 0;\n\t", false},
		{"", attachGeneric, "", true},
		{"pid", attachGeneric, "", true},
		{"pid=123", attachGeneric, "", true},
		{"unknown==1", attachGeneric, "", true},
		{"pid==abc", attachGeneric, "", true},
	}
	for _, tt := range tests {
		got, err := SecToSnippet(tt.sec, tt.point)
		if (err != nil) != tt.wantErr {
			t.Errorf("SecToSnippet(%q, %q): err=%v wantErr=%v", tt.sec, tt.point, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("SecToSnippet(%q, %q): got %q want %q", tt.sec, tt.point, got, tt.want)
		}
	}
}

func TestSecToSnippet_LogicalExpr(t *testing.T) {
	tests := []struct {
		sec     string
		point   string
		contain string
		wantErr bool
	}{
		{"pid==1 and tgid==2", attachGeneric, "ev.pid == 1", false},
		{"pid==1 and tgid==2", attachGeneric, "ev.tgid == 2", false},
		{"pid==1 or cpu==0", attachGeneric, "||", false},
		{"not pid==0", attachGeneric, "!(", false},
		{"(pid==1)", attachGeneric, "ev.pid == 1", false},
		{"(pid==1 or pid==2) and cpu==0", attachGeneric, "&&", false},
		{"arg1 >= 22", attachGeneric, "arg1 >= 22", false},
		{"arg1 <= 22", attachGeneric, "arg1 <= 22", false},
		{"arg1 > 21", attachGeneric, "arg1 > 21", false},
		{"arg1 < 23", attachGeneric, "arg1 < 23", false},
	}
	for _, tt := range tests {
		got, err := SecToSnippet(tt.sec, tt.point)
		if (err != nil) != tt.wantErr {
			t.Errorf("SecToSnippet(%q): err=%v wantErr=%v", tt.sec, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !strings.Contains(got, tt.contain) {
			t.Errorf("SecToSnippet(%q): got %q should contain %q", tt.sec, got, tt.contain)
		}
	}
}

func TestSecToSnippet_SocketFieldsOnlyTcp(t *testing.T) {
	// sport/dport allowed only for tcp_sendmsg and tcp_recvmsg
	_, err := SecToSnippet("sport==22", attachTcpSend)
	if err != nil {
		t.Errorf("SecToSnippet(sport==22, tcp_sendmsg): %v", err)
	}
	_, err = SecToSnippet("dport==22", attachTcpRecv)
	if err != nil {
		t.Errorf("SecToSnippet(dport==22, tcp_recvmsg): %v", err)
	}
	_, err = SecToSnippet("sport==22 or dport==22", attachTcpSend)
	if err != nil {
		t.Errorf("SecToSnippet(sport==22 or dport==22, tcp_sendmsg): %v", err)
	}
	// socket fields on generic point must fail
	_, err = SecToSnippet("sport==22", attachGeneric)
	if err == nil {
		t.Error("SecToSnippet(sport==22, do_sys_open): want error")
	}
	if err != nil && !strings.Contains(err.Error(), "allowed") {
		t.Errorf("SecToSnippet(sport==22, do_sys_open): want 'allowed' in error, got %q", err.Error())
	}
}

func TestSecToSnippet_AllowedFields(t *testing.T) {
	fields := []string{"pid", "tgid", "cpu", "arg0", "arg1", "arg2", "arg3", "arg4", "arg5", "ret"}
	for _, f := range fields {
		_, err := SecToSnippet(f+"==0", attachGeneric)
		if err != nil {
			t.Errorf("SecToSnippet(%s==0): %v", f, err)
		}
		_, err = SecToSnippet(f+"!=1", attachGeneric)
		if err != nil {
			t.Errorf("SecToSnippet(%s!=1): %v", f, err)
		}
	}
}

func TestSecToSnippet_InvalidValue(t *testing.T) {
	_, err := SecToSnippet("pid==0x10", attachGeneric)
	if err == nil {
		t.Error("SecToSnippet(pid==0x10): want error (hex not allowed)")
	}
	// Parser may report unexpected token or invalid number
	if err != nil && !strings.Contains(err.Error(), "number") && !strings.Contains(err.Error(), "integer") && !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("SecToSnippet(pid==0x10): want error about value, got %q", err.Error())
	}
}
