package server

import (
	"context"
	"strings"
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/session"
)

func TestExecuteBreakPrintTrace(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
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

func TestExecuteList(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	// no symbol
	resp, err := exec.execute(ctx, sess, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("list: want ok true got false")
	}
	if !strings.Contains(resp.GetOutput(), "specify a symbol") {
		t.Errorf("list: output should ask for symbol, got %q", resp.GetOutput())
	}

	// with symbol (platform-dependent: kallsyms or stub message)
	resp, err = exec.execute(ctx, sess, "list do_sys_open")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("list do_sys_open: want ok true")
	}
	if resp.GetOutput() == "" {
		t.Error("list do_sys_open: expected some output")
	}
}

func TestExecuteBt(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "bt")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Errorf("bt: want ok true")
	}
	out := resp.GetOutput()
	if !strings.Contains(out, "bt") {
		t.Errorf("bt: output should contain bt, got %q", out)
	}
	// No event yet -> "no event yet" or platform "not supported"
	if !strings.Contains(out, "event") && !strings.Contains(out, "supported") && !strings.Contains(out, "stack") {
		t.Errorf("bt: expected event/supported/stack in output, got %q", out)
	}
}

//nolint:gocyclo // table-driven style test with many cases
func TestExecuteWatchAndDeleteAndInfoWatch(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	// watch missing expression
	resp, err := exec.execute(ctx, sess, "watch")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("watch: want ok false when missing expression")
	}
	if !strings.Contains(resp.GetErrorMessage(), "missing") {
		t.Errorf("watch: want missing expression error, got %q", resp.GetErrorMessage())
	}

	// watch pid -> success, output contains id
	resp, err = exec.execute(ctx, sess, "watch pid")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Fatalf("watch pid: %s", resp.GetErrorMessage())
	}
	out := resp.GetOutput()
	if !strings.Contains(out, "watch") || !strings.Contains(out, "pid") {
		t.Errorf("watch pid: output should contain watch and pid, got %q", out)
	}
	if !strings.Contains(out, "watch-") {
		t.Errorf("watch pid: output should contain watch id (watch-N), got %q", out)
	}

	// info watch -> list the watch
	resp, err = exec.execute(ctx, sess, "info watch")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Fatalf("info watch: %s", resp.GetErrorMessage())
	}
	if !strings.Contains(resp.GetOutput(), "watches") {
		t.Errorf("info watch: output should contain watches, got %q", resp.GetOutput())
	}

	// delete watch-1 (id from first watch)
	list := sess.ListWatches()
	if len(list) != 1 {
		t.Fatalf("expected 1 watch, got %d", len(list))
	}
	id := list[0].ID
	resp, err = exec.execute(ctx, sess, "delete "+id)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetOk() {
		t.Fatalf("delete %s: %s", id, resp.GetErrorMessage())
	}
	if !strings.Contains(resp.GetOutput(), "deleted") {
		t.Errorf("delete: output should contain deleted, got %q", resp.GetOutput())
	}
	if len(sess.ListWatches()) != 0 {
		t.Error("after delete watch, ListWatches should be empty")
	}

	// delete nonexistent watch
	resp, err = exec.execute(ctx, sess, "delete watch-999")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("delete watch-999: want ok false")
	}
}

func TestExecuteHookAdd(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil) // no bpf include dir
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	// missing --code and --sec
	resp, err := exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add (no code/sec): want ok false")
	}
	if !strings.Contains(resp.GetErrorMessage(), "missing --code or --sec") {
		t.Errorf("hook add: want 'missing --code or --sec', got %q", resp.GetErrorMessage())
	}

	// both --code and --sec
	resp, err = exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c --code x --sec pid==1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add (both): want ok false")
	}
	if !strings.Contains(resp.GetErrorMessage(), "cannot use both") {
		t.Errorf("hook add: want 'cannot use both', got %q", resp.GetErrorMessage())
	}

	// --sec only: fails with no bpf include dir (we don't compile in test)
	resp, err = exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c --sec pid==1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add --sec: want ok false (no bpf include dir in test)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "no bpf include dir") {
		t.Errorf("hook add --sec: want 'no bpf include dir', got %q", resp.GetErrorMessage())
	}

	// --sec with --limit: parsing succeeds, still fails without bpf include dir
	resp, err = exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c --sec pid==1 --limit 2")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add --sec --limit: want ok false (no bpf include dir in test)")
	}
	if !strings.Contains(resp.GetErrorMessage(), "no bpf include dir") {
		t.Errorf("hook add --sec --limit: want 'no bpf include dir', got %q", resp.GetErrorMessage())
	}

	// socket field (sport) on non-tcp point: must fail with allowed-field error
	resp, err = exec.execute(ctx, sess, "hook add --point kprobe:do_sys_open --lang c --sec sport==22")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() {
		t.Error("hook add --sec sport==22 on do_sys_open: want ok false")
	}
	if !strings.Contains(resp.GetErrorMessage(), "allowed") && !strings.Contains(resp.GetErrorMessage(), "sport") {
		t.Errorf("hook add sport on do_sys_open: want 'allowed' or 'sport' in error, got %q", resp.GetErrorMessage())
	}
}

func TestExecuteHookAttach(t *testing.T) {
	exec := newCommandExecutor("", "", nil, nil, nil)
	mgr := session.NewManager("")
	sess, _ := mgr.GetOrCreate(context.Background(), "test-session")
	ctx := context.Background()

	resp, err := exec.execute(ctx, sess, "hook attach --attach kprobe:x")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "missing --file or --source") {
		t.Errorf("hook attach missing source: got ok=%v err=%q", resp.GetOk(), resp.GetErrorMessage())
	}

	resp, err = exec.execute(ctx, sess, "hook attach --source 'int x;' --file /tmp/a.c --attach kprobe:x")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "cannot use both") {
		t.Errorf("hook attach both file and source: want error, got %q", resp.GetErrorMessage())
	}

	resp, err = exec.execute(ctx, sess, "hook attach --attach kprobe:do_sys_open --source int x=0;")
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetOk() || !strings.Contains(resp.GetErrorMessage(), "no bpf include dir") {
		t.Errorf("hook attach compile path: want no bpf include dir, got %q", resp.GetErrorMessage())
	}
}
