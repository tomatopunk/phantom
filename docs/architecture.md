# Architecture

## Overview

**Phantom** is a remote eBPF debugger.

- **Agent**: gRPC server that manages debug sessions, executes commands (break/print/trace), and (when wired) loads eBPF programs and streams events from a ring buffer.
- **CLI**: Connects to the agent, opens a session, and runs a REPL that sends command lines and prints responses.

## Data flow

1. CLI starts with `-agent <addr>` and `-token` (optional).
2. CLI calls `Connect` to get or create a session ID.
3. User types commands; CLI sends `Execute(session_id, command_line)`.
4. Agent resolves the session, applies rate limit and quota, runs the command executor (break/print/trace), and returns `ExecuteResponse`.
5. Agent loads eBPF, attaches kprobe/uprobe, and streams `DebugEvent` via `StreamEvents`.

## Components

| Layer        | Responsibility |
|-------------|----------------|
| CLI         | Flags, gRPC client, REPL loop, script mode (`-x file`) |
| Agent API   | Auth (Bearer token), session manager, Execute/StreamEvents/ListSessions/CloseSession |
| Executor    | Parse command line, dispatch break/print/trace/continue, return proto result |
| Session     | Per-session state; quota and rate limiter keyed by session ID |
| Probe       | User-space symbol resolution (ELF) for uprobe |
| Runtime     | Load eBPF from .o file, attach kprobe/uprobe, ring buffer reader, decode events |

## Security

- **Auth**: Optional Bearer token in gRPC metadata; interceptor validates on every RPC.
- **Rate limit**: Per-session requests per second (configurable).
- **Quota**: Max breakpoints, traces, and hooks per session.
- **Audit**: Optional log of each Execute (session, command, ok/err).
- **Health**: Optional HTTP endpoint (e.g. `:8080/health`) for load balancers.

## eBPF

- **Kprobe**: One program in `bpf/probes/kernel/minikprobe.c`; runtime attaches it to a kernel symbol (e.g. `do_sys_open`).
- **Uprobe**: One program in `bpf/probes/user/uprobe.c`; runtime attaches via cilium/ebpf `OpenExecutable` + `Uprobe(symbol)`.
- **Events**: Both use a ring buffer map and `event_header` (timestamp, session_id, event_type, pid, tgid, cpu, probe_id). User space decodes with `runtime.DecodeEvent`.

## Coding standards

See [docs/coding-standards.md](coding-standards.md): English comments at key points, one function one responsibility, clear naming, layer boundaries.

## Further improvements

Possible next steps: condition/watchpoint refinements, session persistence, additional MCP tools. See the codebase and architecture above for extension points.
