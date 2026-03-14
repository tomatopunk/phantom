# Phantom

Remote, interactive eBPF debugger with an **agent** (server) and **cli** (client). The agent injects kprobes/uprobes and runs eBPF programs; the CLI connects over gRPC and provides a GDB-style REPL.

## Build and dependency requirements

- **Go**: Version must satisfy `go.mod` (currently Go 1.25+). Ensure Go is installed and `GOROOT`/`GOPATH` are set before building.
- **Dependencies**: Managed with Go Modules. From the repo root run `go mod download` or `go build ./...` to fetch dependencies; no need to install third-party packages by hand.
- **Regenerating proto** (optional): If you change `pkg/api/proto/*.proto`, install `protoc` and the Go plugins `protoc-gen-go` and `protoc-gen-go-grpc`, then run `make proto`.
- **Building eBPF** (optional, Linux only): Requires `clang`, kernel headers, and libbpf for `make build-bpf` to produce `minikprobe.o`; you can skip this if you are not running real kprobes.

### Environment requirements (Ubuntu)

To build the project (and eBPF programs) on Ubuntu, install:

```bash
sudo apt update
sudo apt install -y build-essential
sudo apt install -y linux-headers-$(uname -r)
sudo apt install -y clang
sudo apt install -y libbpf-dev
```

- `build-essential` — compiler and base build tools.
- `linux-headers-$(uname -r)` — kernel headers for the running kernel (required for compiling eBPF and loading kprobes).
- `clang` — used to compile eBPF C sources (`make build-bpf`).
- `libbpf-dev` — libbpf headers and libraries for eBPF loaders.

## Quick start

**Build (see requirements above):**

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

## tcpdump-style observation (using existing commands)

Without using the system `tcpdump`, you can combine debugger commands to get L3/L4-style metadata: use `break tcp_sendmsg` as the trigger and inspect the event stream for `timestamp`, `pid`, `tgid`, `event_type`, `symbol`, etc.

**Prerequisites:** Linux, CAP_BPF (or root), and built agent/cli plus `bpf/probes/kernel/minikprobe.o`.

**Make targets:**

```bash
make test-e2e-tcpdump-style-cli   # CLI script e2e (break/trace/info/delete lifecycle)
make test-e2e-network            # Go e2e: HTTP/1.0, HTTP/1.1, raw TCP scenarios
make test-e2e-all                 # All of the above + test-e2e-http10-generic
```

**Example CLI flow:**

```
phantom> help
phantom> break tcp_sendmsg
breakpoint set at tcp_sendmsg (bp-1)
phantom> trace pid tgid cpu probe_id
phantom> info break
phantom> continue
# From another terminal, generate traffic: curl --http1.0 http://127.0.0.1:PORT/ or raw TCP
# Event stream will show EVENT_TYPE_BREAK_HIT and pid= / tgid= etc.
phantom> delete bp-1
phantom> info break
```

**Event fields (L3/L4 metadata):** Event type `EVENT_TYPE_BREAK_HIT`, plus `pid`, `tgid`, `cpu`, `probe_id`, `timestamp_ns`, for log-style “who hit which probe when” observation.

**Filtering by port with `hook add --sec`:** On `kprobe:tcp_sendmsg` and `kprobe:tcp_recvmsg` you can use socket fields `sport`, `dport`, `saddr`, `daddr` in `--sec`, and optional `--limit N` to auto-detach after N events. Example (port 22, stop after 2 hits):

```
phantom> hook add --point kprobe:tcp_sendmsg --lang c --sec "sport==22 or dport==22" --limit 2
phantom> hook add --point kprobe:tcp_recvmsg --lang c --sec "sport==22 or dport==22" --limit 2
```

**Go e2e (Linux + env var required):**

```bash
E2E_NETWORK=1 go test -v ./test/e2e/ -run TestTcpdumpStyle
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
