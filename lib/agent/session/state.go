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

// HookState holds one C hook's attach point, detach, cancel, and optional hit limit.
type HookState struct {
	ID          string
	AttachPoint string // e.g. kprobe:do_sys_open
	Detach      func()
	Cancel      func() // cancels the hook's event pump context so reader is closed before detach
	Limit       int    // 0 = no limit; when HitCount >= Limit the hook is auto-removed
	HitCount    int    // incremented on each event; used when Limit > 0
}

// WatchState holds one watch expression and its last value for change detection.
type WatchState struct {
	ID         string
	Expression string
	LastValue  string
	HasValue   bool
}

// WatchTrigger describes a watch that fired (value changed).
type WatchTrigger struct {
	ID         string
	Expression string
	OldValue   string
	NewValue   string
}

// TraceSampleResult holds one trace's evaluated values for a single event (for TRACE_SAMPLE broadcast).
type TraceSampleResult struct {
	TraceID     string            // trace-1, trace-2, ...
	Expressions []string          // original expressions
	Values      map[string]string // expr -> evaluated value
}
