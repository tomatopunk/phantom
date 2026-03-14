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

func TestPlanHook(t *testing.T) {
	p := NewPlanner()
	plan, err := p.PlanHook("kprobe:do_sys_open", "int x = 0;")
	if err != nil {
		t.Fatalf("PlanHook: %v", err)
	}
	if plan.AttachPoint != "kprobe:do_sys_open" || plan.Code != "int x = 0;" {
		t.Errorf("PlanHook: want AttachPoint kprobe:do_sys_open, got %q %q", plan.AttachPoint, plan.Code)
	}

	_, err = p.PlanHook("", "code")
	if err == nil || err.Error() == "" {
		t.Error("PlanHook empty point: want error")
	}
	_, err = p.PlanHook("kprobe:do_sys_open", "")
	if err == nil {
		t.Error("PlanHook empty code: want error")
	}
	_, err = p.PlanHook("invalid", "code")
	if err == nil {
		t.Error("PlanHook invalid attach point: want error")
	}
}
