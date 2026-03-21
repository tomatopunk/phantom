package server

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/session"
)

func TestReplVerbAliases(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "alias-test")
	ctx := context.Background()

	for _, line := range []string{"b do_sys_open", "p pid", "t arg0", "c"} {
		resp, err := exec.execute(ctx, sess, line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if !resp.GetOk() && strings.HasPrefix(line, "b ") {
			// break may fail without kprobe path — still must parse as break, not unknown
			if strings.Contains(resp.GetErrorMessage(), "unknown command") {
				t.Fatalf("%q: should not be unknown command: %s", line, resp.GetErrorMessage())
			}
		}
		if strings.HasPrefix(line, "p ") || strings.HasPrefix(line, "t ") || line == "c" {
			if !resp.GetOk() {
				t.Fatalf("%q: want ok, got %s", line, resp.GetErrorMessage())
			}
		}
	}
}
