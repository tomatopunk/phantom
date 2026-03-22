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

// BreakPlan describes attaching a kprobe at a kernel symbol.
type BreakPlan struct {
	Symbol string
}

// TracePlan describes registering trace expressions (evaluated on each event; no separate eBPF attach).
type TracePlan struct {
	Expressions []string
}

// Planner turns high-level commands (break, trace, hook) into attach plans.
type Planner struct{}

// NewPlanner returns a new probe planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// PlanBreak returns a plan to attach a kprobe at the given symbol.
func (*Planner) PlanBreak(symbol string) BreakPlan {
	return BreakPlan{Symbol: symbol}
}

// PlanTrace returns a plan to register trace expressions (evaluated in event pump).
func (*Planner) PlanTrace(expressions []string) TracePlan {
	return TracePlan{Expressions: expressions}
}
