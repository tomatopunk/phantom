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
| `hook add ...` | — | see below | Inject C hook: `--point kprobe:SYM`, `--lang c`, and either `--code '...'` (custom C) or `--sec <expr>` (condition expression). Optional `--limit N` auto-detaches after N events. Hook events are merged into the session event stream. Uprobe not yet supported. |
| `quit` / `exit` / `q` | — | — | Exit REPL. |

## hook add (details)

- **Required:** `--point` / `-p` — attach point (e.g. `kprobe:do_sys_open`).
- **Required (exactly one):** `--code` / `-c` (custom C snippet) or `--sec` / `-s` (condition expression). Do not pass both.
- **Optional:** `--limit N` — non-negative integer; the hook auto-detaches after N events (default: no limit).
- **`--sec` expression:** Supports comparisons `==`, `!=`, `<`, `<=`, `>`, `>=`, and logic `and`, `or`, `not`, with parentheses. Values are decimal integers. Example: `hook add --point kprobe:do_sys_open --lang c --sec "pid==1234"`.
- **Fields for `--sec` (all attach points):** `pid`, `tgid`, `cpu`, `arg0`…`arg5`, `ret`.
- **Socket fields (only for `kprobe:tcp_sendmsg` and `kprobe:tcp_recvmsg`):** `sport`, `dport`, `saddr`, `daddr`. Using these on any other attach point returns an error. Example: `hook add --point kprobe:tcp_sendmsg --lang c --sec "sport==22 or dport==22" --limit 2`.
- Hook events are merged into the session event stream.

## Expressions (print / trace)

Built-in names: `pid`, `tgid`, `comm`, `cpu`, `arg0` … `arg5`, `ret`. Values are read from the last event context or the probe’s `pt_regs` (kernel) / ABI (user).

## Errors

- `missing session_id` — request had no session.
- `session not found` — session was closed or never created.
- `rate limited` — per-session rate limit exceeded.
- `quota: max breakpoints reached` — session breakpoint quota exceeded (and similar for trace/hook).
- `break: missing symbol` — break command had no argument.
- `hook add: missing --code or --sec` — neither `--code` nor `--sec` was given.
- `hook add: cannot use both --code and --sec (use one)` — both were given.
- `unknown command: <verb>` — verb not recognized.
