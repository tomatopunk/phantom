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
| `hook add ...` | — | see below | Template C hook: `--point` (`kprobe:`, `tracepoint:sub:event`, `uprobe:/path:sym`, `uretprobe:/path:sym`), `--lang c`, and either `--code` or `--sec`. Optional `--limit N`. |
| `hook attach ...` | — | see below | Full C program on the agent: `--attach` (same formats as `hook add --point`), **`--file /abs/path.c`** *or* **`--source '…'`**, optional **`--program`** ELF program name. Use this for custom `SEC("…")` and maps (same path as gRPC `CompileAndAttach`). |
| `quit` / `exit` / `q` | — | — | Exit REPL. |

## hook add (details)

- **Required:** `--point` / `-p` — attach point, one of:
  - `kprobe:kernel_symbol`
  - `tracepoint:subsystem:event` (e.g. `tracepoint:sched:sched_process_fork`)
  - `uprobe:/absolute/path/to/binary:symbol`
  - `uretprobe:/absolute/path/to/binary:symbol`
- **Required (exactly one):** `--code` / `-c` (custom C snippet) or `--sec` / `-s` (condition expression). Do not pass both.
- **Optional:** `--limit N` — non-negative integer; the hook auto-detaches after N events (default: no limit).
- **Note:** `--sec` here is **not** the BPF ELF `SEC("…")` macro; it is a **filter DSL** compiled into an `if` in the generated C. The template picks `SEC("kprobe")`, `SEC("tracepoint/subsys/event")`, `SEC("uprobe")`, or `SEC("uretprobe")` from `--point`. For your own section names and full programs, use **`hook attach`** or **`CompileAndAttach`**.
- **`--sec` expression:** Comparisons `==`, `!=`, `<`, `<=`, `>`, `>=`, and logic `and`, `or`, `not`, with parentheses. Values: **decimal or `0x` hex** integers. Example: `hook add --point kprobe:do_sys_open --lang c --sec "pid==1234"`.
- **Fields for `--sec` (all attach points):** `pid`, `tgid`, `cpu`, `arg0`…`arg5`, `ret`.
- **Socket fields (only for `kprobe:tcp_sendmsg` and `kprobe:tcp_recvmsg`):** `sport`, `dport`, `saddr`, `daddr`. Using these on any other attach point returns an error. Example: `hook add --point kprobe:tcp_sendmsg --lang c --sec "sport==22 or dport==22" --limit 2`.
- **Tracepoint template:** Handler receives `void *ctx`; `arg0`…`arg5` are zero unless your `--code` reads the tracepoint payload from `ctx`.
- Hook events are merged into the session event stream.

## hook attach (details)

Compiles a **complete** C file on the agent (same pipeline as gRPC `CompileAndAttach`). The object must include a **ring buffer** map (as in the built-in examples) so events can stream to the session.

- **Required:** `--attach` / `-a` — same forms as `hook add --point` (`kprobe:…`, `tracepoint:…`, `uprobe:…`, `uretprobe:…`).
- **Required (exactly one):** `--file /absolute/path.c` or `--source '…'` (inline source; practical only for tiny programs).
- **Optional:** `--program` / `-P` — BPF program **function name** in the ELF (if omitted, the loader picks the first program of a suitable type).

Example (program on the agent filesystem):

```text
hook attach --attach kprobe:do_sys_open --file /tmp/myhook.c --program my_handler
```

Errors include `hook attach: empty source`, `hook attach: --file path must be absolute`, `hook attach: read file: …`, and compile/attach failures from clang or the loader.

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
- `hook attach: missing --file or --source` — neither input source was provided.
- `unknown command: <verb>` — verb not recognized.
