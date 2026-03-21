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
	"fmt"

	"github.com/tomatopunk/phantom/lib/agent/hook"
)

// BreakPlan describes attaching a kprobe at a kernel symbol.
type BreakPlan struct {
	Symbol string
}

// TracePlan describes registering trace expressions (evaluated on each event; no separate eBPF attach).
type TracePlan struct {
	Expressions []string
}

// HookPlan describes compiling and attaching a C hook at an attach point.
// Exactly one of Code or Sec is set: Code for custom --code, Sec for --sec (auto-generated snippet).
// Limit is optional: 0 means no limit; when set, the hook auto-detaches after that many events.
type HookPlan struct {
	AttachPoint string
	Code        string // user-provided C snippet when --code
	Sec         string // condition expression when --sec (e.g. pid==123)
	Limit       int    // 0 = no limit; auto-detach after Limit events
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

// PlanHook returns a plan to compile and attach a C hook; validates attach point and code/sec mutual exclusion.
// limit is optional: 0 means no limit; when > 0 the hook auto-detaches after that many events.
func (p *Planner) PlanHook(attachPoint, code, sec string, limit int) (HookPlan, error) {
	if attachPoint == "" {
		return HookPlan{}, fmt.Errorf("missing --point (e.g. kprobe:do_sys_open)")
	}
	if code != "" && sec != "" {
		return HookPlan{}, fmt.Errorf("cannot use both --code and --sec (use one)")
	}
	if code == "" && sec == "" {
		return HookPlan{}, fmt.Errorf("missing --code or --sec")
	}
	if limit < 0 {
		return HookPlan{}, fmt.Errorf("--limit must be >= 0")
	}
	if _, err := hook.ParseFullAttachPoint(attachPoint); err != nil {
		return HookPlan{}, err
	}
	return HookPlan{AttachPoint: attachPoint, Code: code, Sec: sec, Limit: limit}, nil
}
