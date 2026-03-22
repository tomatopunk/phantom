# Command specification

Commands are sent as a single line via `Execute(session_id, command_line)`. The executor splits on whitespace (with quoted tokens) and treats the first token as the verb.

## Commands

| Command | Alias | Args | Description |
|---------|-------|------|-------------|
| `break …` | `b` | see **break / tbreak** | **Catalog template only** (`info break-templates`). Compiles agent-generated BPF for the given **`probe_id`**, optional **`--filter`** kernel predicate DSL. Registers a breakpoint; consumes hook + breakpoint quota when enabled. |
| `tbreak …` | `t` | same as `break` | Default **`--limit 1`** unless overridden. |
| `print <expr>` | `p` | expression | Evaluate on last event (`pid`, `arg0`, …). |
| `continue` | `c` | — | No-op placeholder (REPL). |
| `delete <id>` | — | id | Remove breakpoint or **arg watch** by id. Hooks: `hook delete <id>`. |
| `disable <id>` / `enable <id>` | — | breakpoint id | Detach or recompile template break. |
| `condition <id> <expr>` | — | id, expr | User-side filter on `BREAK_HIT`. |
| `info` | — | `break` \| `break-templates` \| `watch` \| `hook` \| `session` | List state. |
| `list [sym]` | — | optional | Kernel symbol listing / disasm. |
| `bt` | — | — | Kernel stack for last event thread. |
| `watch …` | — | `--sec` + optional `--args` | **Arg-column watch**: requires an **enabled break** on the same catalog `probe_id`. Emits **`EVENT_TYPE_WATCH_ARG`** on break hits. |
| `help [cmd]` | — | optional | Help text. |
| `hook …` | — | `attach` \| `list` \| `delete` | **`hook add` removed.** `hook attach` loads full C; **probe_point** is derived from ELF `SEC` (no `--attach`). |
| `quit` / `exit` / `q` | — | — | Exit REPL. |

## break / tbreak (details)

- **Required:** first token **`probe_id`** from the built-in catalog (e.g. `kprobe.do_sys_open`, `tp.sched.sched_process_fork`).
- **Optional:** `--filter '<dsl>'` — predicate only (identifiers allowed per template; see agent).
- **Optional:** `--limit N` — hook auto-detach after N events (`tbreak` defaults limit **1**).

Example:

```text
break kprobe.do_sys_open --filter "pid==1"
tbreak tp.sched.sched_process_fork --limit 3
```

## hook attach (details)

- **Required:** `--file /abs/path.c` **or** `--source '…'` (inline C).
- **Optional:** `--program` / `-P` — BPF program name; if empty, first suitable kprobe/tracepoint program in the object.
- **Optional:** `--limit N` — auto-detach after N ringbuf events.
- **No** `--attach`: attachment is inferred from section names (`kprobe/…`, `tracepoint/…`, etc.).

## gRPC

- **`CompileAndAttach`** — compile full C and attach from ELF SEC; fields: `session_id`, `source`, `program_name`, `limit`. Response includes **`probe_point`**.
- **`StreamEvents`** — `DebugEvent` includes **`source_kind`** (`break` \| `watch` \| `hook`), **`break_id`**, **`hook_id`**, **`template_probe_id`** where applicable. `EVENT_TYPE_WATCH_ARG` replaces the old trace sample type.

## MCP

`compile_and_attach` takes `source` and optional `program_name` / `limit` (no attach string). `add_c_hook` runs `hook attach --source …`.

## Expressions (`print` / `condition`)

Built-in: `pid`, `tgid`, `comm`, `cpu`, `arg0`–`arg5`, `ret` from last decoded event.

## Event flow

Template **break** and **hook** programs should use a **ringbuf** map. The agent decodes the template layout (extended header + args) and sets **`source_kind`** on streamed events. **Watch** rows are synthetic `WATCH_ARG` events emitted when a matching break hits.

## Errors (selected)

- `unknown probe_id` — not in `info break-templates`.
- `watch: no enabled break for …` — register a break on that `probe_id` first.
- `quota: max breakpoints reached` / `quota: max hooks reached` — per-session limits.
- `no bpf include dir configured` — agent needs BPF headers for compile.
