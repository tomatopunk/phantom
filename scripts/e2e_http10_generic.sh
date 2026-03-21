#!/usr/bin/env bash
# Copyright 2026 The Phantom Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# E2E test for HTTP/1.0 traffic using only generic eBPF: kprobe on tcp_sendmsg, no HTTP-specific code.
# Build agent + minikprobe.o, start agent, break tcp_sendmsg, send HTTP/1.0 request, assert break hit.
# Requires Linux with CAP_BPF (e.g. run as root or privileged).
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [ "$(uname -s)" != "Linux" ]; then
  echo "e2e_http10_generic: skip (not Linux)"
  exit 0
fi

# shellcheck source=e2e_linux_bpf_env.sh
source "$SCRIPT_DIR/e2e_linux_bpf_env.sh"

# Paths: only kprobe object (minikprobe.o), no http uprobe
BPF_KPROBE_OUT="${BPF_KPROBE_OUT:-$ROOT_DIR/src/agent/bpf/probes/kernel/minikprobe.o}"
AGENT_BIN="${AGENT_BIN:-$ROOT_DIR/phantom-agent}"
CLI_BIN="${CLI_BIN:-$ROOT_DIR/target/release/phantom-cli}"
EVENTS_LOG="$(mktemp)"
AGENT_LOG="$(mktemp)"
SERVER_LOG="$(mktemp)"
cleanup() {
  rm -f "$EVENTS_LOG" "$AGENT_LOG" "$SERVER_LOG"
  [ -n "$AGENT_PID" ] && kill "$AGENT_PID" 2>/dev/null || true
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "e2e_http10_generic: building (agent, Rust cli, kprobe only)..."
make -s proto agent cli 2>/dev/null || true
if [ ! -f "$AGENT_BIN" ] || [ ! -f "$CLI_BIN" ]; then
  make proto agent cli
fi
if [ ! -f "$BPF_KPROBE_OUT" ]; then
  echo "e2e_http10_generic: building kprobe (requires clang and kernel headers)..."
  make build-bpf || { echo "e2e_http10_generic: build-bpf failed; skip"; exit 0; }
fi

# Start agent with generic kprobe only (no -http-uprobe)
AGENT_PORT="${AGENT_PORT:-19092}"
AGENT_ADDR="127.0.0.1:$AGENT_PORT"
phantom_e2e_linux_bpf_env "$AGENT_BIN" "e2e_http10_generic"
echo "e2e_http10_generic: starting agent at $AGENT_ADDR (kprobe=$BPF_KPROBE_OUT)..."
if phantom_e2e_agent_needs_sudo; then
  echo "e2e_http10_generic: agent under sudo -E (CI BPF memlock)" >&2
  sudo -E "$AGENT_BIN" -listen "$AGENT_ADDR" -kprobe "$BPF_KPROBE_OUT" >"$AGENT_LOG" 2>&1 &
else
  "$AGENT_BIN" -listen "$AGENT_ADDR" -kprobe "$BPF_KPROBE_OUT" >"$AGENT_LOG" 2>&1 &
fi
AGENT_PID=$!
sleep 1
if ! kill -0 "$AGENT_PID" 2>/dev/null; then
  echo "e2e_http10_generic: agent failed to start"
  cat "$AGENT_LOG"
  exit 1
fi

# Start minimal HTTP server
HTTP_PORT=$((RANDOM + 1024))
python3 -m http.server "$HTTP_PORT" --bind 127.0.0.1 >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!
sleep 0.5
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "e2e_http10_generic: HTTP server failed to start"
  cat "$SERVER_LOG"
  kill "$AGENT_PID" 2>/dev/null || true
  exit 1
fi

# CLI: connect, break tcp_sendmsg (kernel symbol fired when curl sends TCP data), stream events, curl, quit
echo "e2e_http10_generic: setting break on tcp_sendmsg and sending HTTP/1.0 request..."
(
  sleep 0.5
  echo "break tcp_sendmsg"
  sleep 0.3
  curl -s --http1.0 "http://127.0.0.1:$HTTP_PORT/" -o /dev/null || true
  sleep 0.5
  echo "quit"
) | timeout 10 "$CLI_BIN" --agent "$AGENT_ADDR" 2>/dev/null >"$EVENTS_LOG" || true

# Assert we got at least one break hit (generic event, no HTTP parsing)
if grep -q 'type=EVENT_TYPE_BREAK_HIT' "$EVENTS_LOG" && grep -q 'pid=' "$EVENTS_LOG"; then
  echo "e2e_http10_generic: PASS (break hit seen on HTTP/1.0 traffic, generic kprobe)"
else
  echo "e2e_http10_generic: FAIL (no break hit event in stream)"
  echo "--- events ---"
  cat "$EVENTS_LOG"
  echo "--- agent log (last 20 lines) ---"
  tail -20 "$AGENT_LOG"
  exit 1
fi
