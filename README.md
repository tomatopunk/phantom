# Phantom

Remote, interactive **eBPF debugger**: a Go **agent** (gRPC server) loads kprobes/uprobes and streams events; a Rust **CLI** (and optional **Tauri desktop** client) sends GDB-style commands over the network.

## Features

- **gRPC API** — Sessions, `Execute` / `StreamEvents`, discovery and compile-and-attach RPCs ([architecture](docs/architecture.md)).
- **REPL commands** — Break, trace, continue, hooks, watch, and more ([command reference](docs/command-spec.md)).
- **eBPF** — Ring-buffer events from kernel and user-space probes ([`src/agent/bpf`](src/agent/bpf)). REPL `hook add` uses a small C **template** (fixed handler name, `SEC(...)` chosen from `--point`). CLI **`--sec`** is a **filter expression** (converted to an `if` in C), not the ELF `SEC("…")` macro. For arbitrary `SEC` names, tracepoint layouts, and maps, use **`hook attach`** (full C from `--file` / `--source`) or the gRPC **`CompileAndAttach`** RPC — see [docs/command-spec.md](docs/command-spec.md) and [docs/ebpf-parameters.md](docs/ebpf-parameters.md).
- **Hardening** — Optional Bearer token, per-session rate limits and quotas ([architecture](docs/architecture.md#security)).
- **Desktop** — Tauri UI sharing the Rust [`phantom-client`](lib/phantom-client) crate ([`src/desktop/README.md`](src/desktop/README.md)).

## Quick start

```bash
make build                    # Go agent → ./phantom-agent
make cli                      # Rust REPL → target/release/phantom-cli
./phantom-agent -listen :9090
./target/release/phantom-cli --agent localhost:9090
```

Optional token: `PHANTOM_TOKEN=secret ./phantom-agent` and `--token secret` on the CLI.

**Desktop:** `cd src/desktop`, then `npm install` and `npm run tauri dev` — see [`src/desktop/README.md`](src/desktop/README.md).

## Requirements

| Component | Notes |
|-----------|--------|
| **Go** | Version in [`go.mod`](go.mod) (currently 1.25+). |
| **Rust** | Stable toolchain for `phantom-cli` / `phantom-client` / desktop. |
| **eBPF (real probes)** | **Linux** only: `clang`, kernel headers, `libbpf` — see [docs/testing.md](docs/testing.md#ubuntu-packages-reference) and [docs/ops.md](docs/ops.md). |
| **Protos** | To regenerate Go stubs after editing `lib/proto/*.proto`: install `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, then `make proto`. |

```bash
make build-bpf                # Linux: compile .o files under src/agent/bpf/
```

## Documentation

- **[docs/README.md](docs/README.md)** — Index of all technical docs.
- **[docs/architecture.md](docs/architecture.md)** — Design and data flow.
- **[docs/roadmap.md](docs/roadmap.md)** — Planned directions and ideas.

## Testing

```bash
go test ./...                 # Default; e2e BPF tests skip unless env is set
make test-e2e-ci              # Linux + BPF: same subset as CI
```

Full matrix, scripts, and environment variables: **[docs/testing.md](docs/testing.md)**.

## Contributing

PRs should use [Conventional Commits](https://www.conventionalcommits.org/) titles. CI runs Go and Rust lint/tests/coverage, eBPF build checks on Linux, and BPF-oriented e2e. Details: **[CONTRIBUTING.md](CONTRIBUTING.md)**.

## Project layout

| Path | Role |
|------|------|
| [`src/agent`](src/agent) | Agent `main`; eBPF C under `src/agent/bpf/`. |
| [`src/cli`](src/cli) | Rust `phantom-cli` (REPL, `discover`). |
| [`src/desktop`](src/desktop) | Tauri + frontend. |
| [`lib/proto`](lib/proto) | `debugger.proto` and generated Go code. |
| [`lib/agent`](lib/agent) | Agent libraries (server, session, runtime, hook, MCP, discovery, …). |
| [`lib/phantom-client`](lib/phantom-client) | Shared Rust gRPC client. |
| [`test/e2e`](test/e2e) | Go end-to-end tests (incl. `grpcclient`). |

More detail: [`src/README.md`](src/README.md), [`lib/README.md`](lib/README.md).

## Deployment

- **systemd:** [deploy/systemd/phantom-agent.service](deploy/systemd/phantom-agent.service)
- **Operations:** [docs/ops.md](docs/ops.md)

## License

Use under the same terms as the project or repository.
