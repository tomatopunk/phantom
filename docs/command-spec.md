# Command specification

Commands are sent as a single line via `Execute(session_id, command_line)`. The executor splits on whitespace and treats the first token as the verb.

## Commands

| Command | Alias | Args | Description |
|---------|-------|------|-------------|
| `break …` | `b` | see **break / tbreak** below | Compiles **your full eBPF C** on the agent (`CompileRaw`, same as `hook attach` / `CompileAndAttach`). You define **`SEC("…")` in source**. Registers a **breakpoint** (`info break`, `disable`, `enable`, **`condition`** user-side). Consumes a **hook** quota slot + **breakpoint** slot when quotas are enabled. Bare `break <sym>` is **obsolete** — use flags below. |
| `tbreak …` | — | same as `break` | Default **`--limit 1`** (override with `--limit N`); temporary breakpoint removed after enough hook events. |
| `print <expr>` | `p` | expression | Print value (e.g. `pid`, `arg0`, `ret`) from last event context. |
| `trace <expr>` | `t` | one or more | Register expressions; after **each** qualifying probe event (legacy main kprobe ringbuf **or** any hook pump), emit `TRACE_SAMPLE` with evaluated columns. Returns trace id. **`trace` alone does not attach eBPF** — you need a probe that emits ringbuf events: **`hook attach`**, **`break`**, or a loaded legacy kprobe object. |
| `continue` | `c` | — | Continue execution. |
| `delete <id>` | — | id | Remove a **breakpoint**, **trace**, or **watch** by id. Hooks use `hook delete <id>`. |
| `disable <id>` | — | breakpoint id | Disable breakpoint. |
| `enable <id>` | — | breakpoint id | Re-enable breakpoint (recompiles saved user C for `break`-created entries). |
| `condition <id> <expr>` | — | id, expression | **User-side** filter on a breakpoint id; suppresses `BREAK_HIT` when the expression is false. |
| `info` | — | `break` \| `trace` \| `watch` \| `hook` \| `session` | List breakpoints, traces, watches, hooks, or session summary (includes hook count). |
| `list [sym]` | — | optional symbol | List source/disasm near symbol; may return "symbol not available". |
| `bt` | — | — | Backtrace; returns "not supported" if unavailable. |
| `watch <expr>` | — | expression | Register an expression; when its **string value** changes vs the previous event, emit `STATE_CHANGE` (state diff; not a hardware memory watchpoint). Same as `trace`: requires a probe producing events; the **first** event only seeds the baseline (no `STATE_CHANGE` until the value changes). |
| `help [cmd]` | — | optional command | Short help for command or global. |
| `hook …` | — | `attach` \| `list` \| `delete` | **`hook add` is removed.** Use **`hook attach`** with full C (same as `break` without breakpoint state). `hook add` in a script should be migrated to `hook attach --attach … --source …` or `--file /abs/path.c`. |
| `hook attach …` | — | see below | Same compilation path as **`break`**: full C, **`--attach`**, **`--file`** or **`--source`**, optional **`--program`**, optional **`--limit N`**. Creates a **hook** only (no breakpoint row). Same as gRPC `CompileAndAttach`. |
| `quit` / `exit` / `q` | — | — | Exit REPL. |

## break / tbreak (details)

- **Required:** `--attach` / `-a` — `kprobe:…` \| `tracepoint:sub:event` \| `uprobe:/abs:sym` \| `uretprobe:…` (same strings as `hook attach`).
- **Required (exactly one):** `--source '…'` inline C **or** `--file` / `-f` with an **absolute** path on the agent.
- **Optional:** `--program` / `-P` — BPF program function name in the ELF.
- **Optional:** `--limit N` — hook auto-detach after N events (`tbreak` defaults to **1** if you omit `--limit`).
- **Not supported:** bare `break <symbol>`, **`--sec` / `-s` DSL** on `break` (there is no template hook path; express filters in your C).

Example:

```text
break --attach kprobe:do_sys_open --source '...' --program my_entry
tbreak --attach kprobe:foo --file /tmp/x.c --program bar
```

## hook attach (details)

Compiles a **complete** C file on the agent (same pipeline as gRPC `CompileAndAttach` and **`break`**). The object must include a **ring buffer** map so events can stream to the session.

- **Required:** `--attach` / `-a` — `kprobe:…`, `tracepoint:sub:event`, `uprobe:/abs:sym`, `uretprobe:…`.
- **Required (exactly one):** `--file /absolute/path.c` or `--source '…'` (inline source; practical only for tiny programs).
- **Optional:** `--program` / `-P` — BPF program **function name** in the ELF (if omitted, the loader picks the first program of a suitable type).
- **Optional:** `--limit N` — auto-detach the hook after N ringbuf events (same semantics as `break --limit`).

Example (program on the agent filesystem):

```text
hook attach --attach kprobe:do_sys_open --file /tmp/myhook.c --program my_handler
```

Errors include `hook attach: empty source`, `hook attach: --file path must be absolute`, `hook attach: read file: …`, and compile/attach failures from clang or the loader. After successful compilation, **attach** failures are reported as `hook attach: attach failed: …` (same underlying message as gRPC `CompileAndAttach`).

## Typical scenarios (capability bounds)

- **Tcpdump-style L4:** Attach a **full C** program on `kprobe:tcp_sendmsg` / `tcp_recvmsg` that reads socket metadata (e.g. via CO-RE) and emits ringbuf events, then `trace pid tgid …` for columns. Automated e2e: `test/e2e/tcpdump_style_test.go` (set `E2E_NETWORK=1`).
- **Hook + trace on stream:** `CompileAndAttach` or `hook attach` with a tracepoint (or kprobe) program → `trace pid tgid` → gRPC `StreamEvents` yields `EVENT_TYPE_TRACE_SAMPLE`. Automated e2e: `TestE2EHookAddTraceSampleStream` in `test/e2e/scenarios_test.go` (set `E2E_SCENARIOS=1`).
- **L7 / request context:** Usually needs **user-space** uprobes (e.g. libc/TLS) via **`hook attach`** / **`break`** with custom C and maps; raw `tcp_*` kprobes often see buffer pointers, not HTTP text.
- **TC (traffic control), XDP, clsact:** **Not** supported as a dedicated attach string today; attach kinds are `kprobe`, `tracepoint`, `uprobe`, `uretprobe` only. Extending the loader would be a separate change.

## gRPC (supplementary)

- **`CompileAndAttach`** — same as `hook attach` / `break` compile path (no breakpoint unless you use the `break` REPL command). Request may include optional **`limit`** (auto-detach after N hook events; `0` = none).
- **`ValidateCompileSource`** — clang only, no attach (no quota).
- **`ListHookMaps`** / **`ReadHookMap`** — introspect maps for a loaded hook id (including hooks backing `break`).

## MCP

stdio JSON-RPC tools use the same command strings and attach semantics; see [mcp.md](mcp.md). `add_c_hook` sends **`hook attach --source …`** with full C.

## Expressions (print / trace)

Built-in names: `pid`, `tgid`, `comm`, `cpu`, `arg0` … `arg5`, `ret`. Values are read from the last event context or the probe’s `pt_regs` (kernel) / ABI (user).

## Event buffering

Events flow through **three** layers: kernel ringbuf, agent pumps, then per-client subscriber queues. Order is always **kernel → agent → client**.

### Kernel: BPF ring buffer

Each loaded program supplies its own **`BPF_MAP_TYPE_RINGBUF`** map (name and `max_entries` are defined in your C). The prebuilt legacy main kprobe object (`src/agent/bpf/core/events.c`, etc.) uses a fixed layout; **user `hook attach` / `break` objects** define whatever ringbuf size they need. If this map is overrun under extreme load, the kernel may drop or fail individual `bpf_ringbuf_output` / reserve calls — the agent does not resize maps at runtime.

### Agent: ringbuf readers (pumps)

After attach, the session runs **one goroutine per event source**: the legacy main kprobe runtime (if loaded) **plus one pump per hook**, each blocking on `ringbuf.Reader.Read()`. Decoded records become `runtime.Event` values and enter `ProcessProbeEvent` (last-event, `trace` / `watch` derivatives, then broadcast). Pumps do not queue unbounded batches in Go memory beyond what the cilium/ebpf reader holds internally.

### Agent to clients: subscriber channels (gRPC `StreamEvents`, etc.)

Each stream subscribes with a **Go channel of capacity 64** (`eventChanCap` in `lib/agent/server/debugger_server.go`). **`BroadcastEvent` uses non-blocking sends**: if a subscriber’s channel is full, **that event is dropped for that subscriber** so pumps never block on a slow or stalled client. Under burst load, **clients can miss events** even when the kernel ringbuf and agent pump kept up. For reliability under overload, consume the stream promptly or reduce probe rate / filter in your eBPF C.

Synthetic events (`TRACE_SAMPLE`, `STATE_CHANGE`) are broadcast the same way as raw probe hits and are subject to the same per-subscriber drop policy.

## Errors

- `missing session_id` — request had no session.
- `session not found` — session was closed or never created.
- `rate limited` — per-session rate limit exceeded.
- `quota: max breakpoints reached` / `quota: max hooks reached` — `break` / `tbreak` reserve **both** a breakpoint and a hook slot when quotas are enabled (and similar messages for trace-only or hook-only limits).
- `break: obsolete syntax` — bare `break <sym>`; use `--attach` and `--source` / `--file`.
- `break: missing --attach` / `missing --file or --source` — invalid `break` / `tbreak` invocation.
- `hook add is removed` — use `hook attach` with full C.
- `hook attach: missing --file or --source` — neither input source was provided.
- `hook attach: attach failed: …` — compile succeeded but loader could not attach (same situation as gRPC `CompileAndAttach` / MCP `compile_and_attach`).
- `quota: max hooks reached` — session hook quota exceeded (`CompileAndAttach` / `hook attach` / `break`).
- `unknown command: <verb>` — verb not recognized.
