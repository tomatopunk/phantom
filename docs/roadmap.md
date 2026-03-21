# Roadmap

This describes **directions we care about**, not a fixed schedule. Priorities change with contributors and use cases.

## What exists today

- **Agent** — gRPC: sessions, `Execute`, optional Bearer auth, rate limits, quotas, audit hooks; discovery and `CompileAndAttach`; event stream.
- **Rust CLI** — `phantom-cli`: same command lines as the agent REPL, `discover` helpers, shared `lib/phantom-client`.
- **Desktop** — Tauri app using the same Rust client (shared success/failure semantics with CLI).
- **eBPF** — Kprobe/uprobe objects under `src/agent/bpf`; load/attach and ring-buffer events.
- **MCP** — Debugger over stdio JSON-RPC (`tools/call`); see [mcp.md](mcp.md).
- **CI** — Lint, Go/Rust tests and coverage, eBPF build checks, Linux-oriented Go e2e; tagged releases.

## Near-term baseline (delivered)

Goal: **one consistent story** for command behavior and errors across REPL, gRPC, MCP, and clients—not a pile of one-off paths.

**Included in this baseline:**

| Area | What shipped |
|------|----------------|
| **Executor / REPL** | Table-driven dispatch, shared hook quota, help aligned with [command-spec.md](command-spec.md). |
| **REPL ↔ gRPC** | Shared compile/attach path for hooks; attach failures use an `attach failed:` style message (see command-spec and [ebpf-parameters.md](ebpf-parameters.md)). |
| **MCP** | `ExecuteCommandLine` parity with `Execute` (`ok: false` → tool error); tools `compile_and_attach`, `list_tracepoints`, `list_kprobe_symbols`, plus session/command helpers in [mcp.md](mcp.md). |
| **Tests & docs** | e.g. `go test ./lib/agent/mcp/...`; [testing.md](testing.md) covers MCP-related packages. |

**Maintenance:** Keep docs and tests in step when behavior changes. New MCP tools or REPL edge cases belong here unless they are clearly **Mid-term**-sized features.

**Residual (optional, non-blocking):** further refactors inside the hook compile/attach path; on non-Linux, `list` may return an informational message with `ok: true` instead of failing—documented behavior, not a near-term gap.

## Mid-term

- **Session persistence** — Optional recovery or export across agent restarts (design TBD).
- **Conditions & watchpoints** — Richer `condition` / `watch` behavior and validation.
- **Packaging** — Distro packages, container images beyond ad-hoc binaries.

## Long-term / exploratory

- Broader probe models and safer defaults per environment.
- Multi-node / federation (early concept).
- Deeper metrics and tracing around RPC and eBPF attach.

Component layout: [architecture.md](architecture.md).
