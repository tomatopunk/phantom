# Phantom — build and CI
.PHONY: all build proto test fmt vet lint lint-go-linux lint-go-ubuntu lint-go-macos lint-go-ci rust-lint-ci ci-lint clean agent cli rust-cli rust-workspace build-bpf build-uprobe-e2e-helper test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network test-e2e-ci test-e2e-mr test-e2e-all desktop-install desktop-dev desktop-build license-add license-check

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

# Go e2e: HTTP/1.0 + tcpdump-style + scenario tests (recv, open, fork, uprobe) when E2E_SCENARIOS=1.
# Needs Linux, agent, minikprobe.o, optional uprobe helper from build-uprobe-e2e-helper.
test-e2e-ci:
	E2E_HTTP10=1 E2E_NETWORK=1 E2E_SCENARIOS=1 $(GO) test -v ./test/e2e/ -run 'Test(Http10Capture|TcpdumpStyle|E2E)'

# Build tiny C binary for uprobe e2e (Linux cc; no-op on other uname in recipe).
build-uprobe-e2e-helper:
	@if [ "$$(uname -s)" = Linux ]; then \
		cc -g -O0 -o test/e2e/uprobe_helper/uprobe_helper test/e2e/uprobe_helper/main.c; \
	fi

# Re-apply file caps after shell scripts run `go build -o phantom-agent` (replaces inode / drops xattrs).
# Go e2e on GHA uses sudo for the agent anyway; this helps local `make test-e2e-mr` without GITHUB_ACTIONS.
.PHONY: phantom-e2e-reapply-caps
phantom-e2e-reapply-caps:
	@if [ "$$(uname -s)" = Linux ] && command -v sudo >/dev/null && sudo -n true 2>/dev/null; then \
		sudo setcap cap_sys_resource,cap_bpf+ep ./phantom-agent && getcap ./phantom-agent; \
	fi

# MR/CI full BPF e2e: Rust CLI + shell scripts + extended Go e2e.
test-e2e-mr: cli build-uprobe-e2e-helper
	./scripts/e2e_http10_generic.sh
	./scripts/e2e_tcpdump_style_cli.sh
	$(MAKE) phantom-e2e-reapply-caps
	$(MAKE) test-e2e-ci

# Run all e2e: CLI script + HTTP/1.0 script + Go e2e (network tests skip on non-Linux).
test-e2e-all: test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet license-check
	@which staticcheck >/dev/null 2>&1 && staticcheck ./... || true

# --- golangci-lint: match .github/workflows/ci.yml `lint` job (ubuntu + macos matrix) ---
# CI does NOT set GOOS; ubuntu workers analyze linux/amd64, macos workers analyze darwin (arm64 on Apple runners).
# Running only GOOS=linux locally misses //go:build !linux files (e.g. btf_spec_stub.go) that macOS CI still checks.
GOLANGCI_VERSION_EXPECT := v2.11.3

lint-go-ubuntu:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "install golangci-lint $(GOLANGCI_VERSION_EXPECT) (see CI golangci-lint-action)"; exit 1; }
	GOOS=linux GOARCH=amd64 golangci-lint run ./...

# macOS CI uses arm64; override on Intel: make lint-go-macos CI_DARWIN_ARCH=amd64
CI_DARWIN_ARCH ?= arm64

lint-go-macos:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "install golangci-lint $(GOLANGCI_VERSION_EXPECT) (see CI golangci-lint-action)"; exit 1; }
	GOOS=darwin GOARCH=$(CI_DARWIN_ARCH) golangci-lint run ./...

lint-go-ci: proto license-check
	@$(MAKE) lint-go-ubuntu
	@$(MAKE) lint-go-macos

# Back-compat name: linux/amd64 only (does not cover the macOS CI matrix leg).
lint-go-linux: proto license-check lint-go-ubuntu

# --- Rust: match .github/workflows/ci.yml `rust-lint` job ---
rust-lint-ci:
	cargo fmt -p phantom-cli -p phantom-client -- --check
	cargo clippy -p phantom-cli -p phantom-client --all-targets -- -D warnings

# Static checks equivalent to CI `lint` + `rust-lint` (no go test / e2e). Run before every push.
ci-lint: lint-go-ci rust-lint-ci

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
