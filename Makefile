# Phantom — build and CI
.PHONY: all build proto test fmt vet lint clean agent cli rust-cli rust-workspace build-bpf test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network test-e2e-ci test-e2e-all desktop-install desktop-dev desktop-build license-add license-check

BINARY_AGENT := phantom-agent
DESKTOP_DIR  := src/desktop
NPM          := npm
BINARY_RUST_CLI := target/release/phantom-cli
GO           := go
PROTO_DIR    := lib/proto
PROTO_SRC    := $(PROTO_DIR)/debugger.proto
BPF_INCLUDE  := $(CURDIR)/src/agent/bpf/include
BPF_SYSINC   := /usr/include/$(shell uname -m)-linux-gnu
# /usr/include for libbpf headers (bpf/bpf_helpers.h, bpf/bpf_tracing.h) on Linux
BPF_LIBBPF_INC := /usr/include
BPF_KPROBE   := src/agent/bpf/probes/kernel/minikprobe
BPF_UPROBE   := src/agent/bpf/probes/user/uprobe
BPF_EVENTS   := src/agent/bpf/core/events
BPF_OUT      := $(BPF_KPROBE).o
BPF_UPROBE_OUT := $(BPF_UPROBE).o
BPF_EVENTS_OUT := $(BPF_EVENTS).o
CLANG        ?= clang
CLANG_FLAGS  := -target bpf -O2 -g -I $(BPF_INCLUDE) -I $(BPF_SYSINC) -I $(BPF_LIBBPF_INC) -c

all: fmt vet proto build test

# Agent only by default (CI-friendly). Use `make cli` for the Rust REPL binary.
build: agent

agent:
	$(GO) build -o $(BINARY_AGENT) ./src/agent

# Rust REPL + discover (preferred CLI)
cli: rust-cli

rust-cli:
	cargo build -p phantom-cli --release

rust-workspace:
	cargo build --workspace

# Tauri desktop (macOS / Windows / Linux). Needs Node + Rust; eBPF agent still runs on Linux for real probes.
desktop-install:
	cd $(DESKTOP_DIR) && $(NPM) install

desktop-dev:
	cd $(DESKTOP_DIR) && npx tauri dev

desktop-build:
	cd $(DESKTOP_DIR) && $(NPM) run build && cargo build -p phantom-desktop --release

proto: $(PROTO_SRC)
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_SRC)

test:
	$(GO) test ./...

# E2E test for HTTP/1.0 traffic using only generic eBPF (kprobe + break tcp_sendmsg).
# Requires: agent, phantom-cli (Rust), bpf/probes/kernel/minikprobe.o
test-e2e-http10-generic:
	./scripts/e2e_http10_generic.sh

# E2E test: tcpdump-style CLI (break/trace/info/delete + L3/L4 metadata). Linux + CAP_BPF.
test-e2e-tcpdump-style-cli:
	./scripts/e2e_tcpdump_style_cli.sh

# E2E Go tests for network scenarios (HTTP/1.0, HTTP/1.1, raw TCP). Requires Linux, agent, kprobe.
test-e2e-network:
	E2E_NETWORK=1 $(GO) test -v ./test/e2e/ -run 'TestTcpdumpStyle'

# Go e2e used by CI: HTTP/1.0 + tcpdump-style (needs Linux, agent, minikprobe.o, E2E_* env).
test-e2e-ci:
	E2E_HTTP10=1 E2E_NETWORK=1 $(GO) test -v ./test/e2e/ -run 'Test(Http10Capture|TcpdumpStyle)'

# Run all e2e: CLI script + HTTP/1.0 script + Go e2e (network tests skip on non-Linux).
test-e2e-all: test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet license-check
	@which staticcheck >/dev/null 2>&1 && staticcheck ./... || true

# Add Apache-2.0 + SPDX file headers (see scripts/license-addlicense.sh for ignores).
# Override: make license-add LICENSE_COPYRIGHT="Your Name" LICENSE_YEAR=2025
license-add:
	bash scripts/license-addlicense.sh -v .

# Verify headers; used in CI. Fails with non-zero exit if any file is missing a license.
license-check:
	bash scripts/license-addlicense.sh -check .

build-bpf:
	$(CLANG) $(CLANG_FLAGS) $(BPF_KPROBE).c -o $(BPF_OUT)
	$(CLANG) $(CLANG_FLAGS) $(BPF_UPROBE).c -o $(BPF_UPROBE_OUT)
	$(CLANG) $(CLANG_FLAGS) $(BPF_EVENTS).c -o $(BPF_EVENTS_OUT)

clean:
	rm -f $(BINARY_AGENT) $(BPF_OUT) $(BPF_UPROBE_OUT) $(BPF_EVENTS_OUT)
	rm -f $(BINARY_RUST_CLI)
	$(GO) clean -cache -testcache
	find src/agent/bpf -name '*.o' -o -name '*.bpf.o' -o -name '*.skel.h' 2>/dev/null | xargs rm -f 2>/dev/null || true
