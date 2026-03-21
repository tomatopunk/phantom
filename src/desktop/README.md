# Phantom Desktop (Tauri)

Rust shell + **React + TypeScript + Vite + Tailwind**；通过共享 [`lib/phantom-client`](../../lib/phantom-client) 与 agent 走 gRPC。界面支持 **中文 / English**（顶栏语言切换，偏好保存在 `localStorage`）。

## Develop

在仓库根目录：

```bash
make desktop-install   # 首次或依赖变更后
make desktop-dev
```

或手动：

```bash
cd src/desktop
npm install
npx tauri dev
```

仓库根目录已配置 **Cargo workspace**（见根目录 `Cargo.toml`）。

需要先启动 Phantom agent。C hook 编译需要 BPF 头文件，示例：

```bash
./phantom-agent -bpf-include ./src/agent/bpf/include
```

指标 RPC 仅在 **Linux agent** 上有完整数据。

## Build (no bundled installer)

`src-tauri/tauri.conf.json` 中 `"bundle": { "active": false }`。生成 release 二进制：

```bash
make desktop-build
```

或手动：

```bash
cd src/desktop
npm install
npm run build
cargo build --release --manifest-path src-tauri/Cargo.toml
```

或在仓库根目录仅编 Rust（需已 `npm run build` 过前端）：

```bash
cargo build -p phantom-desktop --release
```

产物位于根目录 `target/release/`（workspace）或本目录 `src-tauri/target/release/`（仅构建 manifest 时）。
