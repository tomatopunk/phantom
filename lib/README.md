# `lib` — Shared libraries

| Directory | Description |
|-----------|-------------|
| [`proto/`](proto/) | gRPC `debugger.proto` and generated Go code (`go_package`: `.../lib/proto`). |
| [`agent/`](agent/) | Go agent implementation (server, session, eBPF runtime, hook, MCP, …), imported by [`src/agent`](../src/agent). |
| [`phantom-client/`](phantom-client/) | Rust gRPC client crate used by [`src/cli`](../src/cli) and [`src/desktop`](../src/desktop). |
