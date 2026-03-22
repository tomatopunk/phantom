# Testing

Phantom uses Go tests across `lib/` and `src/`. **End-to-end Go tests under `test/e2e` were removed** for this refactor cycle; that directory holds only a short [README](../test/e2e/README.md). Makefile targets such as `make test-e2e-*` are **no-ops** until a new e2e suite is added.

## Unit and integration (default)

From the repo root:

```bash
go test ./...
```

On **Linux**, with **clang**, **libbpf**, and kernel/UAPI headers installed, this also runs CO-RE compile tests (e.g. [`lib/agent/hook/compile_linux_test.go`](../lib/agent/hook/compile_linux_test.go)). On other OSes, Linux-only packages build with stubs.

### CI parity: lint (run before push)

GitHub Actions runs **golangci-lint on both Ubuntu and macOS** without forcing `GOOS=linux`. A target that only uses `GOOS=linux` does **not** reproduce the macOS job: non-Linux stubs such as [`lib/agent/server/btf_spec_stub.go`](../lib/agent/server/btf_spec_stub.go) are analyzed only when `GOOS=darwin` (or `windows`, etc.).

From the repo root:

```bash
make ci-lint
```

This runs `make proto`, license headers, **two** golangci passes (`linux/amd64` and `darwin` with `CI_DARWIN_ARCH`, default `arm64`), then `cargo fmt` / `cargo clippy` on `phantom-cli` and `phantom-client`. Install **golangci-lint** (same major as CI) and **protoc** locally.

### MCP and debugger server (pure Go)

```bash
go test ./lib/agent/mcp/... ./lib/agent/server/...
```

## Building eBPF objects (Linux)

For agents that load checked-in `.o` probes:

```bash
make build-bpf
```

Produces objects such as `src/agent/bpf/probes/kernel/minikprobe.o` and `src/agent/bpf/probes/user/uprobe.o`.

### Ubuntu packages (reference)

```bash
sudo apt update
sudo apt install -y build-essential
sudo apt install -y linux-headers-$(uname -r)
sudo apt install -y clang
sudo apt install -y libbpf-dev
```

Without **libbpf-dev** / **linux-libc-dev**, CO-RE clang tests may fail with missing **`bpf/bpf_helpers.h`** or **`asm/types.h`**.

## Desktop

```bash
cd src/desktop && npm ci && npm run build
```

## Contract-focused tests (recommended when extending)

When reintroducing coverage, prefer small tests that do **not** require a full agent+BPF stack:

- Template catalog / `probe_id` validation (`lib/agent/breaktpl/…`)
- DSL parser rejections (`lib/agent/breakdsl/…`)
- Executor parsing for `break` / `watch` / `hook attach`
- Ringbuf / `DebugEvent` decoding assumptions
