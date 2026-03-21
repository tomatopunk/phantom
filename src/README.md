# `src` — Product entrypoints and UI

| Directory | Description |
|-----------|-------------|
| [`agent/`](agent/) | Go `main` for the gRPC/MCP agent; eBPF C sources live in `agent/bpf/`. |
| [`cli/`](cli/) | Rust `phantom-cli`: REPL and `discover` subcommands. |
| [`desktop/`](desktop/) | Tauri + Vite/React desktop client. |

Shared libraries live under the repo root [`lib/`](../lib/README.md).
