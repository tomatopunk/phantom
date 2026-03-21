# Roadmap

This document describes **directions we consider valuable**, not a fixed delivery schedule. Priorities change with contributors and use cases.

**Maturity:** Many Near-term capabilities already exist in tree, but **presence is not the same as production-ready usability**—interactive flows, error clarity, parity across REPL / gRPC / MCP, and tests still need active iteration. See [near-maturity-inventory.md](near-maturity-inventory.md) for a prioritized view of those gaps.

## Current (what exists today)

- **Agent** — Go gRPC server: sessions, command execution, optional Bearer auth, rate limits, quotas, audit hooks.
- **Rust CLI** — `phantom-cli`: REPL-style commands, `discover` helpers, shared [`lib/phantom-client`](../lib/phantom-client).
- **Desktop** — Tauri + web frontend using the same Rust client crate (early UX; shares client semantics with CLI).
- **eBPF** — Kprobe and uprobe object files under [`src/agent/bpf`](../src/agent/bpf); runtime load/attach and ring-buffer events.
- **MCP** — Debugger backend exposed for tool-style integrations (contract aligned with `Execute` / sessions).
- **CI** — Lint, tests, coverage (Go + Rust), eBPF build checks, Linux BPF-oriented Go e2e; release workflow for tagged versions.

## Near-term (baseline maturity & usability)

Focus: **stabilize and clarify** what already ships—predictable command behavior, useful errors, and one story across REPL, gRPC, MCP, and clients—not only new features.

- **Executor / REPL** — Table-driven dispatch and shared hook-quota lifecycle; continue tightening errors, help text, and edge cases around `break`, `trace`, `hook`, and session lifecycle.
- **REPL ↔ gRPC parity** — `hook attach` and `CompileAndAttach` share compile/attach paths; keep docs explicit on `--sec` DSL vs BPF `SEC("…")` ([ebpf-parameters.md](ebpf-parameters.md)).
- **MCP** — Same success/failure semantics as `Execute` for command-style tools; then additional tools aligned with common agent workflows.
- **Tests & docs** — Expand coverage on brittle paths (executor, hooks, sessions); keep [testing.md](testing.md) and [command-spec.md](command-spec.md) in sync with behavior.

## Mid-term (larger pieces)

- **Session persistence** — Optional recovery or export of session state across agent restarts (design TBD).
- **Conditions & watchpoints** — Richer `condition` / `watch` behavior and validation.
- **Packaging** — First-class install paths (distro packages, container images) beyond ad-hoc binaries.

## Long-term / ideas (exploratory)

- **Broader probe models** — More attach types, safer defaults, and clearer resource limits per environment.
- **Multi-node / federation** — Optional coordination across agents (very early concept only).
- **Observability** — Deeper metrics and tracing around RPC and eBPF attach paths.

For how components fit together, see [architecture.md](architecture.md).
