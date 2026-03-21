# Roadmap

This document describes **directions we consider valuable**, not a fixed delivery schedule. Priorities change with contributors and use cases.

## Current (what exists today)

- **Agent** — Go gRPC server: sessions, command execution, optional Bearer auth, rate limits, quotas, audit hooks.
- **Rust CLI** — `phantom-cli`: REPL-style commands, `discover` helpers, shared [`lib/phantom-client`](../lib/phantom-client).
- **Desktop** — Tauri + web frontend using the same Rust client crate.
- **eBPF** — Kprobe and uprobe object files under [`src/agent/bpf`](../src/agent/bpf); runtime load/attach and ring-buffer events.
- **MCP** — Debugger backend exposed for tool-style integrations.
- **CI** — Lint, tests, coverage (Go + Rust), eBPF build checks, Linux BPF-oriented Go e2e; release workflow for tagged versions.

## Near-term (concrete follow-ups)

- **Executor / REPL polish** — Clearer errors, help text, and edge cases around `break`, `trace`, `hook`, and session lifecycle.
- **REPL ↔ gRPC parity** — `hook attach` aligns interactive use with `CompileAndAttach`; docs spell out `--sec` DSL vs BPF `SEC("…")` (see [ebpf-parameters.md](ebpf-parameters.md)).
- **MCP** — Additional tools and tighter alignment with common agent workflows.
- **Tests & docs** — Expand unit/integration coverage where brittle; keep [testing.md](testing.md) and command docs in sync with behavior.

## Mid-term (larger pieces)

- **Session persistence** — Optional recovery or export of session state across agent restarts (design TBD).
- **Conditions & watchpoints** — Richer `condition` / `watch` behavior and validation.
- **Packaging** — First-class install paths (distro packages, container images) beyond ad-hoc binaries.

## Long-term / ideas (exploratory)

- **Broader probe models** — More attach types, safer defaults, and clearer resource limits per environment.
- **Multi-node / federation** — Optional coordination across agents (very early concept only).
- **Observability** — Deeper metrics and tracing around RPC and eBPF attach paths.

For how components fit together, see [architecture.md](architecture.md).
