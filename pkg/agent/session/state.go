package session

// BreakpointState holds one breakpoint's runtime state and detach.
type BreakpointState struct {
	ID        string
	Symbol    string
	Detach    func()
	Enabled   bool
	IsTemp    bool
	Condition string // optional expr; when set, event is only reported if condition passes (evaluated later)
}

// TraceState holds one trace's expressions and optional detach.
type TraceState struct {
	ID          string
	Expressions []string
	Detach      func()
}

// HookState holds one C hook's attach point and detach.
type HookState struct {
	ID          string
	AttachPoint string // e.g. kprobe:do_sys_open
	Detach      func()
}
