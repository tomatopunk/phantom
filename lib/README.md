# lib — 跨产品共享代码

| 目录 | 说明 |
|------|------|
| [`proto/`](proto/) | gRPC `debugger.proto` 与 Go 生成代码（`go_package`: `.../lib/proto`） |
| [`agent/`](agent/) | Agent 的 Go 实现（server、session、eBPF runtime、hook、MCP 等），由 [`src/agent`](../src/agent) 入口导入 |
| [`phantom-client/`](phantom-client/) | Rust gRPC 客户端库，供 [`src/cli`](../src/cli) 与 [`src/desktop`](../src/desktop) 共用 |
