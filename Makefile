# Phantom — build and CI
.PHONY: all build proto test fmt vet lint clean agent cli build-bpf test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network test-e2e-all

BINARY_AGENT := phantom-agent
BINARY_CLI   := phantom-cli
GO           := go
PROTO_DIR    := pkg/api/proto
PROTO_SRC    := $(PROTO_DIR)/debugger.proto
BPF_INCLUDE  := $(CURDIR)/bpf/include
BPF_SYSINC   := /usr/include/$(shell uname -m)-linux-gnu
# /usr/include for libbpf headers (bpf/bpf_helpers.h, bpf/bpf_tracing.h) on Linux
BPF_LIBBPF_INC := /usr/include
BPF_KPROBE   := bpf/probes/kernel/minikprobe
BPF_UPROBE   := bpf/probes/user/uprobe
BPF_EVENTS   := bpf/core/events
BPF_OUT      := $(BPF_KPROBE).o
BPF_UPROBE_OUT := $(BPF_UPROBE).o
BPF_EVENTS_OUT := $(BPF_EVENTS).o
CLANG        ?= clang
CLANG_FLAGS  := -target bpf -O2 -g -I $(BPF_INCLUDE) -I $(BPF_SYSINC) -I $(BPF_LIBBPF_INC) -c

all: fmt vet proto build test

build: agent cli

agent:
	$(GO) build -o $(BINARY_AGENT) ./cmd/agent

cli:
	$(GO) build -o $(BINARY_CLI) ./cmd/cli

proto: $(PROTO_SRC)
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_SRC)

test:
	$(GO) test ./...

# E2E test for HTTP/1.0 traffic using only generic eBPF (kprobe + break tcp_sendmsg).
# Requires: agent, cli, bpf/probes/kernel/minikprobe.o
test-e2e-http10-generic:
	./scripts/e2e_http10_generic.sh

# E2E test: tcpdump-style CLI (break/trace/info/delete + L3/L4 metadata). Linux + CAP_BPF.
test-e2e-tcpdump-style-cli:
	./scripts/e2e_tcpdump_style_cli.sh

# E2E Go tests for network scenarios (HTTP/1.0, HTTP/1.1, raw TCP). Requires Linux, agent, kprobe.
test-e2e-network:
	E2E_NETWORK=1 $(GO) test -v ./test/e2e/ -run 'TestTcpdumpStyle'

# Run all e2e: CLI script + HTTP/1.0 script + Go e2e (network tests skip on non-Linux).
test-e2e-all: test-e2e-http10-generic test-e2e-tcpdump-style-cli test-e2e-network

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet
	@which staticcheck >/dev/null 2>&1 && staticcheck ./... || true

build-bpf:
	$(CLANG) $(CLANG_FLAGS) $(BPF_KPROBE).c -o $(BPF_OUT)
	$(CLANG) $(CLANG_FLAGS) $(BPF_UPROBE).c -o $(BPF_UPROBE_OUT)
	$(CLANG) $(CLANG_FLAGS) $(BPF_EVENTS).c -o $(BPF_EVENTS_OUT)

clean:
	rm -f $(BINARY_AGENT) $(BINARY_CLI) $(BPF_OUT) $(BPF_UPROBE_OUT) $(BPF_EVENTS_OUT)
	$(GO) clean -cache -testcache
	find bpf -name '*.o' -o -name '*.bpf.o' -o -name '*.skel.h' 2>/dev/null | xargs rm -f 2>/dev/null || true
