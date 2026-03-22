// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestAddArgWatchRemoveWatchListWatches(t *testing.T) {
	sess := NewSession("s1", "", nil)
	id1 := sess.AddArgWatch("kprobe.do_sys_open", []int{2, 3})
	id2 := sess.AddArgWatch("kprobe.tcp_sendmsg", nil)
	list := sess.ListWatches()
	if len(list) != 2 {
		t.Fatalf("want 2 watches, got %d", len(list))
	}
	if !sess.RemoveWatch(id1) {
		t.Fatal("RemoveWatch id1")
	}
	if len(sess.ListWatches()) != 1 {
		t.Fatalf("want 1 watch left")
	}
	if !sess.RemoveWatch(id2) {
		t.Fatal("RemoveWatch id2")
	}
	if len(sess.ListWatches()) != 0 {
		t.Fatal("want empty")
	}
}

func TestEvaluateWatchArgEvents(t *testing.T) {
	sess := NewSession("s1", "", nil)
	sess.AddArgWatch("kprobe.do_sys_open", []int{2}) // arg0 column
	hit := &runtime.Event{PID: 4, Tgid: 4, Args: [6]uint64{42, 0, 0, 0, 0, 0}}
	evs := sess.EvaluateWatchArgEvents(hit, "kprobe.do_sys_open")
	if len(evs) != 1 {
		t.Fatalf("want 1 watch event, got %d", len(evs))
	}
	if evs[0].EventType != runtime.EventTypeWatchArg {
		t.Fatalf("want WATCH_ARG type")
	}
	if evs[0].SourceKind != "watch" {
		t.Fatalf("want source_kind watch")
	}
}
