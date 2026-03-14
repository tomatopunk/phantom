# Phantom

Remote, interactive eBPF debugger with an **agent** (server) and **cli** (client). The agent injects kprobes/uprobes and runs eBPF programs; the CLI connects over gRPC and provides a GDB-style REPL.

## Quick start

**Build (requires Go 1.21+):**

```bash
make build
# Or: go build -o phantom-agent ./cmd/agent && go build -o phantom-cli ./cmd/cli
```

**Run agent (Linux; optional token):**

```bash
./phantom-agent
# With token: PHANTOM_TOKEN=secret ./phantom-agent
# Listen on custom port: ./phantom-agent -listen :9090
```

**Run CLI:**

```bash
./phantom-cli -agent localhost:9090
# With token: ./phantom-cli -agent localhost:9090 -token secret
# Script mode: ./phantom-cli -agent localhost:9090 -x script.txt  (exits non-zero on first command failure)
```

**Example REPL:**

```
phantom> break do_sys_open
breakpoint set at do_sys_open
phantom> print pid
$pid = (stub)
phantom> trace arg0
tracing arg0
phantom> continue
continue
phantom> quit
```

## Building eBPF programs (Linux)

To load real kprobes/uprobes, build the eBPF objects on a Linux host with clang and kernel headers:

```bash
make build-bpf
```

This produces `bpf/probes/kernel/minikprobe.o` and `bpf/probes/user/uprobe.o`. The agent can then load them via the runtime API.

## E2E test (HTTP/1.0 traffic)

Use the built-in kprobe and `break` command to verify eBPF hook + event stream on HTTP traffic:

```bash
make test-e2e-http10-generic
# Or: ./scripts/e2e_http10_generic.sh
```

This starts the agent with `-kprobe` (minikprobe.o), sets `break tcp_sendmsg`, sends `curl --http1.0`, and asserts that at least one break hit event is received. Requires Linux, CAP_BPF (or root), and `make build-bpf`.

**Go e2e (for CI):**

```bash
E2E_HTTP10=1 go test -v ./test/e2e/ -run TestHttp10CaptureE2E
```

## Project layout

- `cmd/agent` — gRPC server (sessions, execute, stream events)
- `cmd/cli` — REPL client
- `pkg/agent/server` — RPC handlers, auth, rate limit, quota, audit, health
- `pkg/agent/session` — Session lifecycle
- `pkg/agent/probe` — Symbol resolution (user-space)
- `pkg/agent/runtime` — eBPF load/attach, ring buffer events
- `pkg/api/proto` — gRPC protocol
- `pkg/cli/client` — gRPC client
- `pkg/cli/repl` — REPL, flags, script mode, background event stream
- `pkg/agent/hook` — C hook compile and attach
- `pkg/agent/mcp` — MCP server (stdio) for AI/IDE tools
- `bpf/` — eBPF C sources and includes

See [docs/architecture.md](docs/architecture.md) and [docs/command-spec.md](docs/command-spec.md).

## Deployment

- Systemd unit: [deploy/systemd/phantom-agent.service](deploy/systemd/phantom-agent.service)
- Ops and troubleshooting: [docs/ops.md](docs/ops.md)

## License

Use under the same terms as the project or repository.
