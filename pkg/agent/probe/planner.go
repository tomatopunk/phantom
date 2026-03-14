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
type HookPlan struct {
	AttachPoint string
	Code        string
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

// PlanHook returns a plan to compile and attach a C hook; validates attach point format.
func (p *Planner) PlanHook(attachPoint, code string) (HookPlan, error) {
	if attachPoint == "" {
		return HookPlan{}, fmt.Errorf("missing --point (e.g. kprobe:do_sys_open)")
	}
	if code == "" {
		return HookPlan{}, fmt.Errorf("missing --code")
	}
	parts := splitAttachPoint(attachPoint)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return HookPlan{}, fmt.Errorf("attach point must be type:symbol (e.g. kprobe:do_sys_open)")
	}
	if parts[0] != "kprobe" {
		return HookPlan{}, fmt.Errorf("only kprobe supported for C hook")
	}
	return HookPlan{AttachPoint: attachPoint, Code: code}, nil
}

func splitAttachPoint(point string) []string {
	for i := 0; i < len(point); i++ {
		if point[i] == ':' {
			return []string{point[:i], point[i+1:]}
		}
	}
	return nil
}
