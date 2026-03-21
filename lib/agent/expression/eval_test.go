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

package expression

import (
	"testing"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

func TestEvaluate(t *testing.T) {
	ev := &runtime.Event{PID: 42, Tgid: 40, CPU: 2, ProbeID: 1, EventType: 1, TimestampNs: 1000}

	if got := Evaluate(nil, "pid"); got != msgNoEventYet {
		t.Errorf("Evaluate(nil, pid) = %q want %s", got, msgNoEventYet)
	}
	if got := Evaluate(ev, "pid"); got != "42" {
		t.Errorf("Evaluate(ev, pid) = %q want 42", got)
	}
	if got := Evaluate(ev, "tgid"); got != "40" {
		t.Errorf("Evaluate(ev, tgid) = %q want 40", got)
	}
	if got := Evaluate(ev, "cpu"); got != "2" {
		t.Errorf("Evaluate(ev, cpu) = %q want 2", got)
	}
	if got := Evaluate(ev, "probe_id"); got != "1" {
		t.Errorf("Evaluate(ev, probe_id) = %q want 1", got)
	}
	if got := Evaluate(ev, "event_type"); got != "1" {
		t.Errorf("Evaluate(ev, event_type) = %q want 1", got)
	}
	if got := Evaluate(ev, "timestamp_ns"); got != "1000" {
		t.Errorf("Evaluate(ev, timestamp_ns) = %q want 1000", got)
	}
	if got := Evaluate(ev, "unknown"); got != msgUnknownExpr {
		t.Errorf("Evaluate(ev, unknown) = %q want %s", got, msgUnknownExpr)
	}
	// normalizes expr
	if got := Evaluate(ev, "  PID  "); got != "42" {
		t.Errorf("Evaluate(ev, '  PID  ') = %q want 42", got)
	}
	// arg0..arg5, ret (zero when not in event)
	if got := Evaluate(ev, "arg0"); got != "0" {
		t.Errorf("Evaluate(ev, arg0) = %q want 0", got)
	}
	if got := Evaluate(ev, "arg5"); got != "0" {
		t.Errorf("Evaluate(ev, arg5) = %q want 0", got)
	}
	if got := Evaluate(ev, "ret"); got != "0" {
		t.Errorf("Evaluate(ev, ret) = %q want 0", got)
	}
	// comm when empty
	if got := Evaluate(ev, "comm"); got != "(not in event)" {
		t.Errorf("Evaluate(ev, comm) = %q want (not in event)", got)
	}
}

func TestEvaluateArgRetComm(t *testing.T) {
	ev := &runtime.Event{}
	ev.Args[0] = 100
	ev.Args[5] = 200
	ev.Ret = 0
	ev.Comm = "bash"

	if got := Evaluate(ev, "arg0"); got != "100" {
		t.Errorf("Evaluate(ev, arg0) = %q want 100", got)
	}
	if got := Evaluate(ev, "arg5"); got != "200" {
		t.Errorf("Evaluate(ev, arg5) = %q want 200", got)
	}
	if got := Evaluate(ev, "ret"); got != "0" {
		t.Errorf("Evaluate(ev, ret) = %q want 0", got)
	}
	if got := Evaluate(ev, "comm"); got != "bash" {
		t.Errorf("Evaluate(ev, comm) = %q want bash", got)
	}
}

func TestConditionPasses(t *testing.T) {
	ev := &runtime.Event{PID: 42, Tgid: 40, CPU: 0}

	if !ConditionPasses(ev, "") {
		t.Error("empty condition should pass")
	}
	if !ConditionPasses(ev, "pid") {
		t.Error("pid=42 should be truthy")
	}
	if ConditionPasses(ev, "cpu") {
		t.Error("cpu=0 should be false")
	}
	if !ConditionPasses(ev, "1") {
		t.Error("1 should pass")
	}
	if ConditionPasses(ev, "0") {
		t.Error("0 should not pass")
	}
	if ConditionPasses(nil, "pid") {
		t.Error("nil event should not pass")
	}
}
