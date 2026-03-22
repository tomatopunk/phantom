# Architecture

## Overview

**Phantom** is a remote eBPF debugger.

- **Agent**: gRPC server managing sessions, executing commands (break / print / trace / hooks), loading eBPF, streaming ring-buffer events.
- **CLI**: Rust (`cargo build -p phantom-cli` in `src/cli`); talks gRPC to the agent.
- **Desktop**: Tauri app under `src/desktop` using shared `lib/phantom-client`.

## Data flow

1. Client connects with agent address and optional token.
2. **`OpenSession`** returns a session id.
3. User commands go through **`Execute(session_id, command_line)`**.
4. Agent applies rate limit and quota, runs the executor, returns **`ExecuteResponse`**.
5. eBPF loads attach kprobe/uprobe; **`StreamEvents`** delivers **`DebugEvent`**.
6. Other RPCs include **`CompileAndAttach`**, **`ListTracepoints`**, **`ListKprobeSymbols`**, **`ListUprobeSymbols`**, **`InspectELF`** (discovery and full-C hooks).

## Components

| Layer | Role |
|-------|------|
| CLI | REPL and `discover` in `src/cli` |
| Agent API | Auth, sessions, Execute, streams, discovery, compile/attach |
| Discovery | `lib/agent/discovery`: tracefs, kallsyms, ELF symbols |
| Hook compile | `lib/agent/hook`: clang CO-RE, `CompileRaw` + attach for `break` / `hook attach` / `CompileAndAttach` |
| Executor | Parse line, dispatch verbs, return proto result |
| Session | Per-session state; quota and rate limiter |
| Probe | User-space ELF resolution for uprobes |
| Runtime | Load `.o`, attach probes, ring buffer, decode events |

## Security

- Optional **Bearer** token on gRPC metadata.
- **Rate limit** and **quota** per session (breakpoints, traces, hooks).
- Optional **audit** log of each Execute.
- Optional **HTTP health** endpoint for load balancers.

## eBPF

- **Kprobe** — `src/agent/bpf/probes/kernel/minikprobe.c`.
- **Uprobe** — `src/agent/bpf/probes/user/uprobe.c`.
- **Events** — Ring buffer + shared `event_header`; decoded in user space via `runtime.DecodeEvent`.

## Roadmap

Larger themes (persistence, packaging, etc.) are in [roadmap.md](roadmap.md). Code style: [coding-standards.md](coding-standards.md).
