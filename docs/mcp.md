# MCP (stdio JSON-RPC)

MCP over **stdin/stdout** uses the same session model as gRPC. Tools that run commands share **`Execute`** semantics: `ok: false` surfaces as a JSON-RPC error.

## Transport

- One JSON-RPC 2.0 object per line on stdin; responses on stdout.
- Supported method: **`tools/call`** with `params.name` and `params.arguments` (object).

## Tools

| Tool | Arguments | Returns |
|------|-----------|---------|
| `set_breakpoint` | `session_id`, `symbol` | Runs `break <symbol>` (legacy one-arg form); **obsolete** — expect `break: obsolete syntax`. Prefer **`run_command`** with `break --attach kprobe:<sym> --source '…'` / `--file …`, or **`compile_and_attach`**. |
| `run_command` | `session_id`, `command_line` | Text output from `Execute`. |
| `add_c_hook` | `session_id`, `attach_point`, `code` (full eBPF C) | Runs `hook attach --source …` via `Execute` (text output). |
| `compile_and_attach` | `session_id`, `source`, `attach`, optional `program_name`, optional `limit` | On success: **JSON** string (protojson of `CompileAndAttachResponse`, same path as gRPC). On logical failure: JSON-RPC **error** with agent message. |
| `list_sessions` | — | Session ids, one per line. |
| `list_breakpoints` | `session_id` | Text listing. |
| `list_hooks` | `session_id` | Text listing. |
| `list_tracepoints` | `prefix`, optional `max_entries` (default 5000) | Tracepoint names, one per line (same discovery as gRPC `ListTracepoints`). |
| `list_kprobe_symbols` | `prefix`, optional `max_entries` (default 5000) | Kprobe symbol names, one per line (same as gRPC `ListKprobeSymbols`). |

Numeric arguments may be JSON numbers (e.g. `max_entries`).

REPL equivalents for `run_command` / `add_c_hook`: [command-spec.md](command-spec.md). Hook parameters: [ebpf-parameters.md](ebpf-parameters.md).

`trace` / `watch` apply to events from **`break`**, **`hook attach`**, and the legacy main kprobe runtime when loaded: each probe event updates the session’s last-event context and can emit `TRACE_SAMPLE` / `STATE_CHANGE` derivatives. Use `info session` for counts (`hooks=`, etc.).

**Event buffering:** same pipeline as gRPC — kernel BPF ringbuf (size is defined in **your** C or the legacy agent object), then agent pumps, then per-stream **64-slot** subscriber channels with **drop-on-full** delivery. Details: [command-spec.md — Event buffering](command-spec.md#event-buffering).
