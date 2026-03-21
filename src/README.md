# src — 按产品划分的入口与 UI

| 目录 | 说明 |
|------|------|
| [`agent/`](agent/) | Go `main`：启动 gRPC/MCP agent；eBPF C 源码在 `agent/bpf/` |
| [`cli/`](cli/) | Rust `phantom-cli`：REPL 与 discover 子命令 |
| [`desktop/`](desktop/) | Tauri + Vite/React 桌面客户端 |

共享逻辑见仓库根目录的 [`lib/`](../lib/README.md)。
