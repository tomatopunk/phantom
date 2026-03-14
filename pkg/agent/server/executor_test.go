package server

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/pkg/agent/session"
)

func TestExecuteBreakPrintTrace(t *testing.T) {
	exec := newCommandExecutor("")
	mgr := session.NewManager("") // no kprobe path: break will fail
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")

	tests := []struct {
		line   string
		wantOk bool
	}{
		{"break do_sys_open", false}, // no kprobe path in test
		{"print pid", true},
		{"trace arg0", true},
		{"continue", true},
		{"break", false},
		{"print", false},
		{"unknown_cmd", false},
	}
	for _, tt := range tests {
		resp, err := exec.execute(context.Background(), sess, tt.line)
		if err != nil {
			t.Errorf("execute %q: %v", tt.line, err)
			continue
		}
		if resp.GetOk() != tt.wantOk {
			t.Errorf("execute %q: ok=%v want %v", tt.line, resp.GetOk(), tt.wantOk)
		}
		if tt.wantOk && resp.GetOutput() == "" && !strings.Contains(tt.line, "continue") {
			if resp.GetBreakpoint() == nil && resp.GetPrint() == nil && resp.GetTrace() == nil {
				t.Errorf("execute %q: expected some output or result", tt.line)
			}
		}
	}
}
