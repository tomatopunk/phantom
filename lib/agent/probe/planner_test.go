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

package probe

import (
	"testing"
)

func TestPlanBreak(t *testing.T) {
	p := NewPlanner()
	plan := p.PlanBreak("do_sys_open")
	if plan.Symbol != "do_sys_open" {
		t.Errorf("PlanBreak: want Symbol do_sys_open, got %q", plan.Symbol)
	}
}

func TestPlanTrace(t *testing.T) {
	p := NewPlanner()
	exprs := []string{"pid", "cpu"}
	plan := p.PlanTrace(exprs)
	if len(plan.Expressions) != 2 || plan.Expressions[0] != "pid" || plan.Expressions[1] != "cpu" {
		t.Errorf("PlanTrace: want [pid cpu], got %v", plan.Expressions)
	}
}
