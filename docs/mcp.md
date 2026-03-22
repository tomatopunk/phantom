# MCP (stdio JSON-RPC)

MCP over **stdin/stdout** uses the same session model as gRPC. Tools that run commands share **`Execute`** semantics: `ok: false` surfaces as a JSON-RPC error.

## Transport

- One JSON-RPC 2.0 object per line on stdin; responses on stdout.
- Supported method: **`tools/call`** with `params.name` and `params.arguments` (object).

## Tools

| Tool | Arguments | Returns |
|------|-----------|---------|
| `set_breakpoint` | `session_id`, `probe_id` | Runs `break <probe_id>` (catalog id from `info break-templates`). |
| `run_command` | `session_id`, `command_line` | Text output from `Execute`. |
| `add_c_hook` | `session_id`, `code` (full eBPF C) | Runs `hook attach --source …` via `Execute`. **Probe point** comes from `SEC("…")` in the object (no attach string). |
| `compile_and_attach` | `session_id`, `source`, optional `program_name`, optional `limit` | On success: **JSON** string (protojson of `CompileAndAttachResponse`). On logical failure: JSON-RPC **error** with agent message. |
| `list_sessions` | — | Session ids, one per line. |
| `list_breakpoints` | `session_id` | Text listing. |
| `list_hooks` | `session_id` | Text listing. |
| `list_tracepoints` | `prefix`, optional `max_entries` (default 5000) | Tracepoint names, one per line (same discovery as gRPC `ListTracepoints`). |
| `list_kprobe_symbols` | `prefix`, optional `max_entries` (default 5000) | Kprobe symbol names, one per line (same as gRPC `ListKprobeSymbols`). |

Numeric arguments may be JSON numbers (e.g. `max_entries`).

REPL equivalents for `run_command` / `add_c_hook`: [command-spec.md](command-spec.md). Hook `SEC` layout: [ebpf-parameters.md](ebpf-parameters.md).

**Events:** each probe hit updates the session last-event context. **Arg watches** (`watch --sec …`) emit **`EVENT_TYPE_WATCH_ARG`** on matching template break hits. Streamed **`DebugEvent`** values carry **`source_kind`** (`break` \| `watch` \| `hook`) plus structured ids where applicable. Use `info session` for counts.

**Event buffering:** kernel BPF ringbuf → agent pumps → per-stream **64-slot** subscriber channels with **drop-on-full** delivery. Details were summarized in older revisions of [command-spec.md](command-spec.md) (event pipeline section); behavior is unchanged aside from event types.
