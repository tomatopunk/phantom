# eBPF hooks: parameters and extension strategies

How **user-visible parameters** reach BPF programs. REPL syntax: [command-spec.md](command-spec.md).

## BPF `SEC("…")` (ELF section)

**`SEC("…")` in your C source** is the **ELF section** that libbpf-style loaders use to classify programs (`kprobe/…`, `tracepoint/…`, `uprobe`, etc.). You set it explicitly in every program loaded via **`break`**, **`hook attach`**, or gRPC **`CompileAndAttach`**.

There is **no** separate REPL “filter DSL” for hooks; express conditions in C (maps, CO-RE reads, early `return 0`, etc.).

## Full C: `break` / `hook attach` / `CompileAndAttach`

- You write **`SEC("…")`**, maps, CO-RE reads, and constants in C.
- **REPL:** `break --attach <point> --file /abs/path.c [--program name] [--limit N]` or `--source '…'` (breakpoint ids); `hook attach` with the same flags except no breakpoint row.
- **gRPC:** `CompileAndAttach` with `source`, `attach`, optional `program_name`, optional `limit`.
- **MCP:** `compile_and_attach` — same pipeline as gRPC ([mcp.md](mcp.md)). `add_c_hook` runs **`hook attach --source …`** with full C.

## Supported attach kinds

The loader accepts:

- `kprobe:symbol`
- `tracepoint:subsystem:event` (subsystem and event names: letters, digits, underscore only)
- `uprobe:/absolute/path:symbol`
- `uretprobe:/absolute/path:symbol`

## Runtime tunables (future / optional)

A **BPF map** (e.g. `config`) filled by the agent before attach could allow changing behavior per session without recompiling. This is **not** implemented in the agent today; the extension point would be new CLI/proto fields plus `Map.Update` after `NewCollection`.
