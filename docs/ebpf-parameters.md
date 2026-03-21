# eBPF hooks: parameters and extension strategies

How **user-visible parameters** reach BPF programs, and how **`hook add --sec`** differs from BPF **`SEC("…")`**. REPL syntax: [command-spec.md](command-spec.md).

## Two meanings of “sec”

| Term | What it is |
|------|------------|
| **`hook add --sec`** | A small **condition DSL** (e.g. `pid==123 and arg0==0xff`). The agent turns it into a C `if (!cond) return 0;` in front of your `--code` snippet (or alone as the snippet body). It does **not** set the ELF section name. |
| **BPF `SEC("…")`** | The **ELF section** that libbpf-style loaders use to classify programs (`kprobe`, `tracepoint/...`, `uprobe`, etc.). In the template path this is **generated for you** from `--point`. For full control, use **`hook attach`** or **`CompileAndAttach`** with your own C source. |

## Strategies for “parameters”

### 1. Template hook + `--sec` DSL (narrow, safe)

- **Fields:** `pid`, `tgid`, `cpu`, `arg0`…`arg5`, `ret`, plus symbol-specific extras registered via **`hook.RegisterPrologue`** (see [`lib/agent/hook/prologue.go`](../lib/agent/hook/prologue.go)).
- **Literals:** Decimal integers and **`0x` / `0X` hex** (e.g. `arg0==0xff`).
- **Best for:** Fast filters without shipping a full `.c` file.

### 2. Full C: `hook attach` / `CompileAndAttach` (maximum control)

- You write **`SEC("…")`**, maps, CO-RE reads, and any constants in C.
- **REPL:** `hook attach --attach <point> --file /abs/path.c [--program name]` or `--source '…'`.
- **gRPC:** `CompileAndAttach` with `source`, `attach`, optional `program_name`.
- **MCP:** `compile_and_attach` — same pipeline as gRPC ([mcp.md](mcp.md)).
- **Best for:** Custom sections, tracepoint-specific `ctx` layout, BPF maps, and production-style programs.

### 3. Runtime tunables (future / optional)

- A **BPF map** (e.g. `config`) filled by the agent before attach would allow changing behavior per session without recompiling. This is **not** implemented in the agent today; the extension point would be new CLI/proto fields plus `Map.Update` after `NewCollection`.

## `hook add --point` attach kinds

Template compilation now supports:

- `kprobe:symbol`
- `tracepoint:subsystem:event` (subsystem and event names: letters, digits, underscore only)
- `uprobe:/absolute/path:symbol`
- `uretprobe:/absolute/path:symbol`

Tracepoint templates use `void *ctx` and zero `arg0`…`arg5` unless your `--code` reads from `ctx`. Kprobe/uprobe templates use `struct pt_regs *ctx` and `PT_REGS_PARM*`.
