# Command specification

Commands are sent as a single line via `Execute(session_id, command_line)`. The executor splits on whitespace and treats the first token as the verb.

## Commands

| Command | Alias | Args | Description |
|---------|-------|------|-------------|
| `break <sym>` | `b` | symbol | Set breakpoint (kprobe). Returns breakpoint id and symbol. Uprobe not yet supported. |
| `tbreak <sym>` | — | symbol | Temporary breakpoint (auto-delete on first hit). |
| `print <expr>` | `p` | expression | Print value (e.g. `pid`, `arg0`, `ret`) from last event context. |
| `trace <expr>` | `t` | one or more | Start tracing expressions. Returns trace id. |
| `continue` | `c` | — | Continue execution. |
| `delete <id>` | — | breakpoint id | Remove breakpoint by id. |
| `disable <id>` | — | breakpoint id | Disable breakpoint. |
| `enable <id>` | — | breakpoint id | Re-enable breakpoint. |
| `condition <id> <expr>` | — | id, expression | Breakpoint condition; BREAK_HIT only when condition passes. |
| `info` (break, trace, hook, session) | — | subcommand | List breakpoints, traces, hooks, or session info. |
| `list [sym]` | — | optional symbol | List source/disasm near symbol; may return "symbol not available". |
| `bt` | — | — | Backtrace; returns "not supported" if unavailable. |
| `watch <expr>` | — | expression | Watchpoint (emit when value changes). |
| `help [cmd]` | — | optional command | Short help for command or global. |
| `hook add ...` | — | see below | Inject C hook: `--point kprobe:SYM`, `--lang c`, `--code '...'`. Uprobe attach point not yet supported. |
| `quit` / `exit` / `q` | — | — | Exit REPL. |

## Expressions (print / trace)

Built-in names: `pid`, `tgid`, `comm`, `cpu`, `arg0` … `arg5`, `ret`. Values are read from the last event context or the probe’s `pt_regs` (kernel) / ABI (user).

## Errors

- `missing session_id` — request had no session.
- `session not found` — session was closed or never created.
- `rate limited` — per-session rate limit exceeded.
- `quota: max breakpoints reached` — session breakpoint quota exceeded (and similar for trace/hook).
- `break: missing symbol` — break command had no argument.
- `unknown command: <verb>` — verb not recognized.
