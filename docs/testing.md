# Testing

Phantom uses Go tests across `lib/` and `src/`, plus a Go **e2e** package under [`test/e2e`](../test/e2e). Some tests run in-process only; others need **Linux**, **CAP_BPF** (or root), and built eBPF objects.

## Unit and integration (no kernel BPF)

From the repo root:

```bash
go test ./...
```

The e2e package includes in-process gRPC tests that always run; BPF-heavy tests **skip** unless environment variables are set (see below).

### CI parity: lint (run before push)

GitHub Actions runs **golangci-lint on both Ubuntu and macOS** without setting `GOOS`. A target that only uses `GOOS=linux` (to typecheck `//go:build linux` files) does **not** reproduce the macOS job: non-Linux stubs such as [`lib/agent/server/btf_spec_stub.go`](../lib/agent/server/btf_spec_stub.go) are analyzed only when `GOOS=darwin` (or `windows`, etc.).

From the repo root, match the `lint` + `rust-lint` workflow jobs:

```bash
make ci-lint
```

This runs `make proto`, license headers, **two** golangci passes (`linux/amd64` and `darwin` with `CI_DARWIN_ARCH`, default `arm64`), then `cargo fmt` / `cargo clippy` on `phantom-cli` and `phantom-client`. Install **golangci-lint v2.11.3** (same as CI) and **protoc** locally.

### MCP and debugger server (pure Go)

Packages [`lib/agent/mcp`](../lib/agent/mcp) and [`lib/agent/server`](../lib/agent/server) (REPL dispatch, hooks). Focused run:

```bash
go test ./lib/agent/mcp/... ./lib/agent/server/...
```

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
- **linux-libc-dev** — glibc/kernel UAPI headers (`linux/*.h`, multiarch `asm/*.h`) for CO-RE clang in tests.

On Linux, **`go test ./...`** includes [`TestCompile_COReHook`](../lib/agent/hook/compile_linux_test.go), which invokes **clang** with the same include layout as **`make build-bpf`**; without **libbpf-dev** / **linux-libc-dev**, you will see missing **`bpf/bpf_helpers.h`** or **`asm/types.h`**.

## E2E: HTTP/1.0 (generic kprobe)

Verifies eBPF hook + event stream using `break tcp_sendmsg` and HTTP/1.0 traffic:

```bash
make test-e2e-http10-generic
# Or: ./scripts/e2e_http10_generic.sh
```

Requires Linux, CAP_BPF (or root), built agent, Rust CLI (per script), and `make build-bpf`.

## E2E: CI / MR Go subset

Same Go e2e as the `e2e-bpf` job’s Go step (inside `make test-e2e-mr`): `E2E_HTTP10`, `E2E_NETWORK`, and **`E2E_SCENARIOS`** for extra BPF scenarios.

```bash
make test-e2e-ci
```

Equivalent:

```bash
E2E_HTTP10=1 E2E_NETWORK=1 E2E_SCENARIOS=1 go test -v ./test/e2e/ -run 'Test(Http10Capture|TcpdumpStyle|E2E)'
```

**Scenarios covered (Linux only; `scenarios_test.go` is `//go:build linux`):**

- **Network:** `tcp_sendmsg` (HTTP/1.0, HTTP/1.1, raw TCP) and **`tcp_recvmsg`** (client receives a response body).
- **Files:** kprobe break on **`do_sys_open` / `do_sys_openat2`** (best-effort symbol from kallsyms) plus a local `Open`.
- **Process:** **`tracepoint:sched:sched_process_fork`** via `CompileAndAttach` / full C hook (needs agent `-bpf-include`); triggers a child process.
- **User space:** **`uprobe`** on `phantom_e2e_marker` in [`test/e2e/uprobe_helper`](../test/e2e/uprobe_helper) (build with `make build-uprobe-e2e-helper`, or set `E2E_UPROBE_HELPER` to the binary path).

Requires Linux, built `phantom-agent`, and `minikprobe.o` at the default path (or set `E2E_AGENT_BIN` / `E2E_KPROBE` / `PHANTOM_KPROBE` — see [`test/e2e/helpers.go`](../test/e2e/helpers.go)). Some scenarios skip if attach/compile fails (kernel variance).

**Custom kernel without sysfs BTF:** set **`E2E_VMLINUX`** to a vmlinux ELF path so the agent gets **`-vmlinux`** (BTF fallback for hooks). See [vmlinux.md](vmlinux.md).

## E2E: full MR target (shell + Go)

Runs Rust `phantom-cli`, both shell scripts, then `test-e2e-ci`:

```bash
make test-e2e-mr
```

Requires Linux, `clang`, kernel headers, `libbpf`, `curl`, `python3`, **Rust** (`make cli`), and `make build-bpf` / `phantom-agent` (the Makefile recipe builds the uprobe helper on Linux automatically).

**Hardened / CI hosts:** [`scripts/e2e_linux_bpf_env.sh`](../scripts/e2e_linux_bpf_env.sh) raises **memlock** where possible and applies **`setcap`** (**cap_sys_resource**, **cap_bpf**) when **sudo** is available. On **GitHub Actions**, the shell e2e scripts start the agent with **`phantom_e2e_run_agent_sudo`** (**`sudo -n -E bash -c 'ulimit -l unlimited; exec …'`**). **Go** e2e matches that on **Linux** when **`GITHUB_ACTIONS`** is set: it starts **`phantom-agent`** under the same **sudo + bash + ulimit + exec** pattern (file **`setcap`** is unreliable after **`go build -o phantom-agent`** overwrites the binary during shell e2e). **`make test-e2e-mr`** also runs **`phantom-e2e-reapply-caps`** (**`setcap`** + **`getcap`**) after the shell scripts so a local run without **`GITHUB_ACTIONS`** still gets file caps before **`go test`**. When not on GHA, **Go** e2e uses **`bash` + `ulimit` + `exec`** without **sudo**; set **`E2E_AGENT_USE_SUDO=1`** to force **sudo**, or **`E2E_AGENT_USE_SUDO=0`** on GHA to force the non-sudo path (e.g. debugging **setcap**).

If **`setcap`** fails (wrong filesystem, missing **libcap**), fix the environment rather than ignoring the error. If you still see **MEMLOCK** / **operation not permitted**, check **cgroup v2** **`memory.memlock.max`** on the host (including self-hosted runners): the kernel caps locked memory even for root; raising that limit may require a different runner image or service configuration.

Set **`PHANTOM_STRICT_MEMLOCK=1`** (as in the **`e2e-bpf`** workflow) so the agent **exits at startup** when **`rlimit.RemoveMemlock()`** fails and **`-kprobe`** is set, instead of failing later on breakpoint/hook load. Kprobe **`KERNEL_VERSION`** is filled from **uname** in the agent so **cilium/ebpf** does not need **`/proc/self/mem`**.

## Tcpdump-style observation (commands only)

Without the system `tcpdump`, you can treat `break tcp_sendmsg` as a trigger and read the event stream for `timestamp`, `pid`, `tgid`, `event_type`, `symbol`, and related fields.

**Prerequisites:** Linux, CAP_BPF (or root), agent + CLI built, and `minikprobe.o`.

**Make targets:**

```bash
make test-e2e-tcpdump-style-cli   # CLI script: break / trace / info / delete lifecycle
make test-e2e-network             # Go e2e: HTTP/1.0, HTTP/1.1, raw TCP
make test-e2e-all                 # Scripts + test-e2e-http10-generic + network Go e2e (no E2E_SCENARIOS)
make test-e2e-mr                  # CLI + scripts + full Go e2e (matches CI e2e-bpf)
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

## Port filtering on TCP kprobes

Filter by L4 in your **full C** (CO-RE reads on `struct sock`, etc.), then `trace` for columns. `hook attach --limit N` auto-detaches after N events.

Example shape (source on the agent as a `.c` file):

```
phantom> hook attach --attach kprobe:tcp_sendmsg --file /tmp/tcp_hook.c --limit 2
```

## Go e2e: tcpdump-style tests only

```bash
E2E_NETWORK=1 go test -v ./test/e2e/ -run TestTcpdumpStyle
```
