package hook

import "testing"

func TestParseFullAttachPoint_TracepointIdent(t *testing.T) {
	_, err := ParseFullAttachPoint("tracepoint:bad/name:event")
	if err == nil {
		t.Fatal("want error for subsystem with slash")
	}
	_, err = ParseFullAttachPoint("tracepoint:sched:sched_process_fork")
	if err != nil {
		t.Fatal(err)
	}
}
