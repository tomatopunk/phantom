package probe

import "fmt"

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
	parts := splitAttachPoint(attachPoint)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return HookPlan{}, fmt.Errorf("attach point must be type:symbol (e.g. kprobe:do_sys_open)")
	}
	if parts[0] != "kprobe" {
		return HookPlan{}, fmt.Errorf("only kprobe supported for C hook")
	}
	return HookPlan{AttachPoint: attachPoint, Code: code, Sec: sec, Limit: limit}, nil
}

func splitAttachPoint(point string) []string {
	for i := 0; i < len(point); i++ {
		if point[i] == ':' {
			return []string{point[:i], point[i+1:]}
		}
	}
	return nil
}
