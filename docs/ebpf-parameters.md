# eBPF hooks: parameters and extension strategies

How **user-visible parameters** reach BPF programs. REPL syntax: [command-spec.md](command-spec.md).

## BPF `SEC("…")` (ELF section)

**`SEC("…")` in C** is the **ELF section** libbpf-style loaders use to classify programs (`kprobe/…`, `tracepoint/…`, `uprobe`, etc.).

- **Template `break`:** the agent generates C for a **catalog `probe_id`**; you do not author `SEC` for breaks.
- **User hook:** you set **`SEC("…")`** in full C loaded via **`hook attach`** or gRPC **`CompileAndAttach`**. The loader derives the **probe point** from section names; there is **no** separate `--attach` string.

There is **no** REPL “filter DSL” for hooks; express conditions in C. Template breaks support a **small kernel predicate DSL** via `break … --filter`.

## Full C: `hook attach` / `CompileAndAttach`

- You write **`SEC("…")`**, maps, CO-RE reads, and constants in C.
- **REPL:** `hook attach --file /abs/path.c [--program name] [--limit N]` or `--source '…'`.
- **gRPC:** `CompileAndAttach` with `source`, optional `program_name`, optional `limit`.
- **MCP:** `compile_and_attach` — same pipeline as gRPC ([mcp.md](mcp.md)). `add_c_hook` runs **`hook attach --source …`**.

## Supported section / attach kinds (user hooks)

The loader accepts programs in sections such as:

- `kprobe/symbol` → kernel kprobe
- `tracepoint/subsystem/event` → tracepoint
- `uprobe/…` / `uretprobe/…` → user probes (when enabled)

Subsystem and event names follow libbpf naming rules (letters, digits, underscore).

## Runtime tunables (future / optional)

A **BPF map** (e.g. `config`) filled by the agent before attach could allow changing behavior per session without recompiling. This is **not** implemented in the agent today; the extension point would be new CLI/proto fields plus `Map.Update` after `NewCollection`.
