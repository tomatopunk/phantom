package probe

import (
	"strings"
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

	// --code only
	plan, err := p.PlanHook("kprobe:do_sys_open", "int x = 0;", "", 0)
	if err != nil {
		t.Fatalf("PlanHook(code): %v", err)
	}
	if plan.AttachPoint != "kprobe:do_sys_open" || plan.Code != "int x = 0;" || plan.Sec != "" || plan.Limit != 0 {
		t.Errorf("PlanHook(code): got AttachPoint=%q Code=%q Sec=%q Limit=%d", plan.AttachPoint, plan.Code, plan.Sec, plan.Limit)
	}

	// --sec only
	plan, err = p.PlanHook("kprobe:do_sys_open", "", "pid==123", 0)
	if err != nil {
		t.Fatalf("PlanHook(sec): %v", err)
	}
	if plan.AttachPoint != "kprobe:do_sys_open" || plan.Code != "" || plan.Sec != "pid==123" {
		t.Errorf("PlanHook(sec): got AttachPoint=%q Code=%q Sec=%q", plan.AttachPoint, plan.Code, plan.Sec)
	}

	// with limit
	plan, err = p.PlanHook("kprobe:tcp_sendmsg", "", "sport==22", 2)
	if err != nil {
		t.Fatalf("PlanHook(limit): %v", err)
	}
	if plan.Limit != 2 {
		t.Errorf("PlanHook(limit): got Limit=%d want 2", plan.Limit)
	}

	// both code and sec -> error
	_, err = p.PlanHook("kprobe:do_sys_open", "code", "pid==1", 0)
	if err == nil {
		t.Error("PlanHook(both): want error")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("PlanHook(both): want 'cannot use both', got %q", err.Error())
	}

	// neither -> error
	_, err = p.PlanHook("kprobe:do_sys_open", "", "", 0)
	if err == nil {
		t.Error("PlanHook(neither): want error")
	}
	if err != nil && !strings.Contains(err.Error(), "missing --code or --sec") {
		t.Errorf("PlanHook(neither): want 'missing --code or --sec', got %q", err.Error())
	}

	// negative limit -> error
	_, err = p.PlanHook("kprobe:do_sys_open", "code", "", -1)
	if err == nil {
		t.Error("PlanHook(negative limit): want error")
	}

	// empty point
	_, err = p.PlanHook("", "code", "", 0)
	if err == nil || err.Error() == "" {
		t.Error("PlanHook empty point: want error")
	}

	// invalid attach point
	_, err = p.PlanHook("invalid", "code", "", 0)
	if err == nil {
		t.Error("PlanHook invalid attach point: want error")
	}
}
