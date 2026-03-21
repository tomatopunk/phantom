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

package session

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestAddWatchRemoveWatchListWatches(t *testing.T) {
	sess := NewSession("test", "")
	id1 := sess.AddWatch("pid")
	if id1 == "" || id1 != "watch-1" {
		t.Errorf("first watch id want watch-1 got %q", id1)
	}
	id2 := sess.AddWatch("cpu")
	if id2 != "watch-2" {
		t.Errorf("second watch id want watch-2 got %q", id2)
	}
	list := sess.ListWatches()
	if len(list) != 2 {
		t.Fatalf("ListWatches want 2 got %d", len(list))
	}
	if !sess.RemoveWatch("watch-1") {
		t.Error("RemoveWatch watch-1 should return true")
	}
	if sess.RemoveWatch("watch-1") {
		t.Error("RemoveWatch watch-1 again should return false")
	}
	list = sess.ListWatches()
	if len(list) != 1 || list[0].ID != "watch-2" {
		t.Errorf("after remove watch-1, ListWatches want [watch-2] got %v", list)
	}
}

func TestEvaluateWatchChanges(t *testing.T) {
	sess := NewSession("test", "")
	sess.AddWatch("pid")
	sess.AddWatch("cpu")

	// First event: establishes baseline, no trigger (HasValue was false)
	ev1 := &runtime.Event{PID: 100, Tgid: 100, CPU: 0}
	triggers := sess.EvaluateWatchChanges(ev1)
	if len(triggers) != 0 {
		t.Errorf("first event: want 0 triggers (baseline only), got %d", len(triggers))
	}

	// Same values: no trigger
	ev2 := &runtime.Event{PID: 100, Tgid: 100, CPU: 0}
	triggers = sess.EvaluateWatchChanges(ev2)
	if len(triggers) != 0 {
		t.Errorf("same values: want 0 triggers, got %d", len(triggers))
	}

	// pid changed: one trigger for pid watch
	ev3 := &runtime.Event{PID: 200, Tgid: 100, CPU: 0}
	triggers = sess.EvaluateWatchChanges(ev3)
	if len(triggers) != 1 {
		t.Fatalf("pid change: want 1 trigger, got %d", len(triggers))
	}
	if triggers[0].Expression != "pid" || triggers[0].OldValue != "100" || triggers[0].NewValue != "200" {
		t.Errorf("trigger want pid 100->200 got %q %q->%q", triggers[0].Expression, triggers[0].OldValue, triggers[0].NewValue)
	}

	// Both pid and cpu change: two triggers
	ev4 := &runtime.Event{PID: 300, Tgid: 100, CPU: 1}
	triggers = sess.EvaluateWatchChanges(ev4)
	if len(triggers) != 2 {
		t.Errorf("pid and cpu change: want 2 triggers, got %d", len(triggers))
	}

	// Remove pid watch; only cpu watch remains
	sess.RemoveWatch("watch-1")
	ev5 := &runtime.Event{PID: 400, Tgid: 100, CPU: 1}
	triggers = sess.EvaluateWatchChanges(ev5)
	if len(triggers) != 0 {
		t.Errorf("after remove watch-1: only cpu watch left, cpu 1->1 so want 0 triggers, got %d", len(triggers))
	}
	// pid changed but that watch was removed; cpu unchanged so no trigger from watch-2
	ev6 := &runtime.Event{PID: 500, Tgid: 100, CPU: 1}
	triggers = sess.EvaluateWatchChanges(ev6)
	if len(triggers) != 0 {
		t.Errorf("only pid changed but pid watch removed, want 0 triggers got %d", len(triggers))
	}
}
