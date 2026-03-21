# Testing

Phantom uses Go tests across `lib/` and `src/`, plus a Go **e2e** package under [`test/e2e`](../test/e2e). Some tests run in-process only; others need **Linux**, **CAP_BPF** (or root), and built eBPF objects.

## Unit and integration (no kernel BPF)

From the repo root:

```bash
go test ./...
```

The e2e package includes in-process gRPC tests that always run; BPF-heavy tests **skip** unless environment variables are set (see below).

## Building eBPF objects (Linux)

Required for real kprobe tests and scripted e2e:

```bash
make build-bpf
```

Produces objects such as `src/agent/bpf/probes/kernel/minikprobe.o` and `src/agent/bpf/probes/user/uprobe.o`. The agent loads them via the runtime API.

### Ubuntu packages (reference)

```bash
sudo apt update
sudo apt install -y build-essential
sudo apt install -y linux-headers-$(uname -r)
sudo apt install -y clang
sudo apt install -y libbpf-dev
```

- **build-essential** — Base toolchain.
- **linux-headers-$(uname -r)** — Headers for the running kernel (compile + load kprobes).
- **clang** — Compiles eBPF C (`make build-bpf`).
- **libbpf-dev** — libbpf headers for loaders.

## E2E: HTTP/1.0 (generic kprobe)

Verifies eBPF hook + event stream using `break tcp_sendmsg` and HTTP/1.0 traffic:

```bash
make test-e2e-http10-generic
# Or: ./scripts/e2e_http10_generic.sh
```

Requires Linux, CAP_BPF (or root), built agent, Rust CLI (per script), and `make build-bpf`.

## E2E: CI subset (Go)

Same Go test subset as GitHub Actions (`E2E_HTTP10` + `E2E_NETWORK`):

```bash
make test-e2e-ci
```

Equivalent:

```bash
E2E_HTTP10=1 E2E_NETWORK=1 go test -v ./test/e2e/ -run 'Test(Http10Capture|TcpdumpStyle)'
```

Requires Linux, built `phantom-agent`, and `minikprobe.o` at the default path (or set `E2E_AGENT_BIN` / `E2E_KPROBE` / `PHANTOM_KPROBE` — see [`test/e2e/helpers.go`](../test/e2e/helpers.go)).

## Tcpdump-style observation (commands only)

Without the system `tcpdump`, you can treat `break tcp_sendmsg` as a trigger and read the event stream for `timestamp`, `pid`, `tgid`, `event_type`, `symbol`, and related fields.

**Prerequisites:** Linux, CAP_BPF (or root), agent + CLI built, and `minikprobe.o`.

**Make targets:**

```bash
make test-e2e-tcpdump-style-cli   # CLI script: break / trace / info / delete lifecycle
make test-e2e-network             # Go e2e: HTTP/1.0, HTTP/1.1, raw TCP
make test-e2e-all                 # Scripts + test-e2e-http10-generic + network Go e2e
```

**Example REPL flow:**

```
phantom> help
phantom> break tcp_sendmsg
breakpoint set at tcp_sendmsg (bp-1)
phantom> trace pid tgid cpu probe_id
phantom> info break
phantom> continue
# From another terminal: curl --http1.0 http://127.0.0.1:PORT/ or raw TCP
phantom> delete bp-1
phantom> info break
```

**Useful event fields:** `EVENT_TYPE_BREAK_HIT`, `pid`, `tgid`, `cpu`, `probe_id`, `timestamp_ns`.

## Port filtering with `hook add --sec`

On `kprobe:tcp_sendmsg` and `kprobe:tcp_recvmsg`, socket fields `sport`, `dport`, `saddr`, and `daddr` are available in `--sec`. Optional `--limit N` detaches after N events.

Example (SSH port, stop after two hits):

```
phantom> hook add --point kprobe:tcp_sendmsg --lang c --sec "sport==22 or dport==22" --limit 2
phantom> hook add --point kprobe:tcp_recvmsg --lang c --sec "sport==22 or dport==22" --limit 2
```

## Go e2e: tcpdump-style tests only

```bash
E2E_NETWORK=1 go test -v ./test/e2e/ -run TestTcpdumpStyle
```

See [command-spec.md](command-spec.md) for full command syntax.
