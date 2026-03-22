# Command specification

Commands are sent as a single line via `Execute(session_id, command_line)`. The executor splits on whitespace and treats the first token as the verb.

## Commands

| Command | Alias | Args | Description |
|---------|-------|------|-------------|
| `break <sym>` | `b` | symbol + optional kernel filter | **Built-in kprobe template only:** argument must be a **bare kernel symbol** (no `kprobe:`, `tracepoint:`, `uprobe:`, paths). The agent always generates `SEC("kprobe")` for this path; you do **not** pick the ELF section. Optional **`--sec` / `-s`** uses the **same filter DSL** as `hook add --sec` for that symbol (e.g. `pid`, `tgid`, `cpu`, `arg0`…; `sport`/`dport`/… only on `tcp_sendmsg`/`tcp_recvmsg`). Default kernel filter is `pid>=0` if `--sec` is omitted. **`condition`** is still **user-side** (suppress `BREAK_HIT` in the agent after the event). Tracepoints/uprobes/custom `SEC` → **`hook add`** or **`hook attach`**. Needs `--bpf-include`; uses a **hook** quota slot + breakpoint slot when quotas are enabled. |
| `tbreak <sym>` | — | same as `break` | Temporary breakpoint (`--limit 1` on the template hook); auto-delete on first hit. |
| `print <expr>` | `p` | expression | Print value (e.g. `pid`, `arg0`, `ret`) from last event context. |
| `trace <expr>` | `t` | one or more | Register expressions; after **each** qualifying probe event (legacy main kprobe ringbuf **or** any hook pump), emit `TRACE_SAMPLE` with evaluated columns. Returns trace id. **`trace` alone does not attach eBPF** — you need an active **`break`** / **`hook add`** / **`hook attach`** (or a loaded legacy kprobe object) so events reach the session. |
| `continue` | `c` | — | Continue execution. |
| `delete <id>` | — | id | Remove a **breakpoint**, **trace**, or **watch** by id. Hooks use `hook delete <id>`. |
| `disable <id>` | — | breakpoint id | Disable breakpoint. |
| `enable <id>` | — | breakpoint id | Re-enable breakpoint. |
| `condition <id> <expr>` | — | id, expression | **User-side** filter on a breakpoint id; suppresses `BREAK_HIT` when the expression is false. Contrast **`break … --sec`** and **`hook add --sec`**, which are **kernel-side** filters in the generated C. |
| `info` | — | `break` \| `trace` \| `watch` \| `hook` \| `session` | List breakpoints, traces, watches, hooks, or session summary (includes hook count). |
| `list [sym]` | — | optional symbol | List source/disasm near symbol; may return "symbol not available". |
| `bt` | — | — | Backtrace; returns "not supported" if unavailable. |
| `watch <expr>` | — | expression | Register an expression; when its **string value** changes vs the previous event, emit `STATE_CHANGE` (state diff; not a hardware memory watchpoint). Same as `trace`: requires a probe producing events; the **first** event only seeds the baseline (no `STATE_CHANGE` until the value changes). |
| `help [cmd]` | — | optional command | Short help for command or global. |
| `hook add ...` | — | see below | **You choose the attach kind** via `--point`; the template emits the matching `SEC("kprobe")`, `SEC("tracepoint/…")`, `SEC("uprobe")`, etc. Either **`--code`** (snippet) or **`--sec`** (filter DSL). Optional `--limit N`. This is the path for **non–bare-symbol** probes (tracepoint, uprobe, explicit `kprobe:` spelling, custom snippet). |
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
- **`break` vs `hook add`:** Both use the same **C template** and clang pipeline for the probe *body*, and both get an auto-chosen `SEC(...)` from the attach kind. **`break`** is restricted to the **built-in kprobe** shape (`<symbol>` only) and is meant for debugger-style stops; **`hook add`** is the general tool for **any** `--point` and user-defined `--sec` / `--code`.
- Hook events (including those backing `break` / `tbreak`) are merged into the session event stream; they drive `print`, `trace`, and `watch` the same way as legacy prebuilt kprobe hits when that path is loaded.

## Typical scenarios (capability bounds)

- **Tcpdump-style L4:** `break tcp_sendmsg` or `hook add --point kprobe:tcp_sendmsg --lang c --sec "sport==22 or dport==22"`, plus `trace pid tgid sport dport` for per-hit columns. Socket fields `sport`/`dport`/`saddr`/`daddr` apply only to `kprobe:tcp_sendmsg` and `kprobe:tcp_recvmsg` in `--sec` (see above). Automated e2e: `test/e2e/tcpdump_style_test.go` (set `E2E_NETWORK=1`).
- **Hook + trace on stream:** Full path `hook add` (tracepoint) → `trace pid tgid` → gRPC `StreamEvents` must yield `EVENT_TYPE_TRACE_SAMPLE` with `pid=` / `tgid=` in the payload. Automated e2e: `TestE2EHookAddTraceSampleStream` in `test/e2e/scenarios_test.go` (set `E2E_SCENARIOS=1`, same agent/BPF prerequisites as other scenario tests).
- **L7 / request context:** Usually needs **user-space** uprobes (e.g. libc/TLS) via `hook add --point uprobe:…` or **`hook attach`** with custom C and maps; raw `tcp_*` kprobes often see buffer pointers, not HTTP text.
- **TC (traffic control), XDP, clsact:** **Not** supported as `hook add --point tc:…` today; attach kinds are `kprobe`, `tracepoint`, `uprobe`, `uretprobe` only. Extending the loader would be a separate change; until then use **`hook attach`** only if your program attaches to a **supported** hook type.

## hook attach (details)

Compiles a **complete** C file on the agent (same pipeline as gRPC `CompileAndAttach`). The object must include a **ring buffer** map (as in the built-in examples) so events can stream to the session.

- **Required:** `--attach` / `-a` — same forms as `hook add --point` (`kprobe:…`, `tracepoint:…`, `uprobe:…`, `uretprobe:…`).
- **Required (exactly one):** `--file /absolute/path.c` or `--source '…'` (inline source; practical only for tiny programs).
- **Optional:** `--program` / `-P` — BPF program **function name** in the ELF (if omitted, the loader picks the first program of a suitable type).

Example (program on the agent filesystem):

```text
hook attach --attach kprobe:do_sys_open --file /tmp/myhook.c --program my_handler
```

Errors include `hook attach: empty source`, `hook attach: --file path must be absolute`, `hook attach: read file: …`, and compile/attach failures from clang or the loader. After successful compilation, **attach** failures are reported as `hook attach: attach failed: …` (same underlying message as gRPC `CompileAndAttach`).

## MCP

stdio JSON-RPC tools use the same command strings and attach semantics; see [mcp.md](mcp.md).

## Expressions (print / trace)

Built-in names: `pid`, `tgid`, `comm`, `cpu`, `arg0` … `arg5`, `ret`. Values are read from the last event context or the probe’s `pt_regs` (kernel) / ABI (user).

## Event buffering

Events flow through **three** layers: kernel ringbuf, agent pumps, then per-client subscriber queues. Order is always **kernel → agent → client**.

### Kernel: BPF ring buffer

Template-generated hooks (`break` / `hook add` C template) declare a `BPF_MAP_TYPE_RINGBUF` map (typically named `events`) with **`max_entries = 256 * 1024` bytes** (see `lib/agent/hook/embed/hook.c` and `src/agent/bpf/core/events.c`). This holds raw samples until user space reads them. **`hook attach`** and other custom objects must supply their own ringbuf map; its size is whatever the program defines. If this map is overrun under extreme load, the kernel may drop or fail individual `bpf_ringbuf_output` calls according to kernel/BPF rules — the agent does not resize this map at runtime.

### Agent: ringbuf readers (pumps)

After attach, the session runs **one goroutine per event source**: the legacy main kprobe runtime (if loaded) **plus one pump per hook**, each blocking on `ringbuf.Reader.Read()`. Decoded records become `runtime.Event` values and enter `ProcessProbeEvent` (last-event, `trace` / `watch` derivatives, then broadcast). Pumps do not queue unbounded batches in Go memory beyond what the cilium/ebpf reader holds internally.

### Agent to clients: subscriber channels (gRPC `StreamEvents`, etc.)

Each stream subscribes with a **Go channel of capacity 64** (`eventChanCap` in `lib/agent/server/debugger_server.go`). **`BroadcastEvent` uses non-blocking sends**: if a subscriber’s channel is full, **that event is dropped for that subscriber** so pumps never block on a slow or stalled client. Under burst load, **clients can miss events** even when the kernel ringbuf and agent pump kept up. For reliability under overload, consume the stream promptly or reduce probe rate / use kernel-side `--sec` filters.

Synthetic events (`TRACE_SAMPLE`, `STATE_CHANGE`) are broadcast the same way as raw probe hits and are subject to the same per-subscriber drop policy.

## Errors

- `missing session_id` — request had no session.
- `session not found` — session was closed or never created.
- `rate limited` — per-session rate limit exceeded.
- `quota: max breakpoints reached` / `quota: max hooks reached` — `break` / `tbreak` reserve **both** a breakpoint and a hook slot when quotas are enabled (and similar messages for trace-only or hook-only limits).
- `break: missing symbol` — break command had no argument.
- `hook add: missing --code or --sec` — neither `--code` nor `--sec` was given.
- `hook add: cannot use both --code and --sec (use one)` — both were given.
- `hook attach: missing --file or --source` — neither input source was provided.
- `hook attach: attach failed: …` — compile succeeded but loader could not attach (same situation as gRPC `CompileAndAttach` / MCP `compile_and_attach`).
- `quota: max hooks reached` — session hook quota exceeded (`CompileAndAttach` / `hook add` / `hook attach`).
- `unknown command: <verb>` — verb not recognized.
