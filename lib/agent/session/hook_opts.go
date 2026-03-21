package session

// HookOpts optional metadata for hooks (UI / info listing).
type HookOpts struct {
	FilterExpr string // REPL hook add --sec DSL when used
	Note       string // e.g. "hook add", "CompileAndAttach"
}
