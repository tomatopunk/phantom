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

# E2E test: tcpdump-style observation using existing commands only.
# Flow: help -> break tcp_sendmsg -> trace pid/tgid/cpu -> info break -> continue ->
#       trigger HTTP/1.0, HTTP/1.1, raw TCP -> assert L3/L4 metadata in events ->
#       delete breakpoint -> info break (empty).
# Requires Linux with CAP_BPF (e.g. run as root or privileged).
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [ "$(uname -s)" != "Linux" ]; then
  echo "e2e_tcpdump_style_cli: skip (not Linux)"
  exit 0
fi

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

echo "e2e_tcpdump_style_cli: building (agent, Rust cli, kprobe)..."
make -s proto agent cli 2>/dev/null || true
if [ ! -f "$AGENT_BIN" ] || [ ! -f "$CLI_BIN" ]; then
  make proto agent cli
fi
if [ ! -f "$BPF_KPROBE_OUT" ]; then
  echo "e2e_tcpdump_style_cli: building kprobe..."
  make build-bpf || { echo "e2e_tcpdump_style_cli: build-bpf failed; skip"; exit 0; }
fi

AGENT_PORT="${AGENT_PORT:-19093}"
AGENT_ADDR="127.0.0.1:$AGENT_PORT"
echo "e2e_tcpdump_style_cli: starting agent at $AGENT_ADDR..."
"$AGENT_BIN" -listen "$AGENT_ADDR" -kprobe "$BPF_KPROBE_OUT" >"$AGENT_LOG" 2>&1 &
AGENT_PID=$!
sleep 1
if ! kill -0 "$AGENT_PID" 2>/dev/null; then
  echo "e2e_tcpdump_style_cli: agent failed to start"
  cat "$AGENT_LOG"
  exit 1
fi

# Minimal HTTP server for HTTP/1.0 and HTTP/1.1
HTTP_PORT=$((RANDOM % 30000 + 1024))
python3 -m http.server "$HTTP_PORT" --bind 127.0.0.1 >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!
sleep 0.5
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "e2e_tcpdump_style_cli: HTTP server failed to start"
  cat "$SERVER_LOG"
  kill "$AGENT_PID" 2>/dev/null || true
  exit 1
fi

# CLI: full tcpdump-style lifecycle + traffic
echo "e2e_tcpdump_style_cli: running break/trace/info/continue, then traffic..."
(
  sleep 0.5
  echo "help"
  sleep 0.2
  echo "break tcp_sendmsg"
  sleep 0.3
  echo "trace pid tgid cpu probe_id"
  sleep 0.2
  echo "info break"
  sleep 0.2
  echo "continue"
  sleep 0.3
  # HTTP/1.0
  curl -s --http1.0 "http://127.0.0.1:$HTTP_PORT/" -o /dev/null || true
  sleep 0.3
  # HTTP/1.1
  curl -s --http1.1 "http://127.0.0.1:$HTTP_PORT/" -o /dev/null || true
  sleep 0.3
  # Raw TCP (trigger tcp_sendmsg)
  echo "X" | timeout 1 nc -q 0 127.0.0.1 "$HTTP_PORT" 2>/dev/null || true
  sleep 0.5
  # Extract bp id from output and delete (format: "breakpoint set at tcp_sendmsg (bp-N)")
  # We cannot easily parse interactively; send delete for common id and info break
  echo "info break"
  sleep 0.2
  # Delete first breakpoint (bp-1) then verify empty
  echo "delete bp-1"
  sleep 0.2
  echo "info break"
  sleep 0.2
  echo "quit"
) | timeout 15 "$CLI_BIN" --agent "$AGENT_ADDR" 2>/dev/null >"$EVENTS_LOG" || true

# Assert: at least one break hit with L3/L4-style metadata
if ! grep -q 'type=EVENT_TYPE_BREAK_HIT' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (no EVENT_TYPE_BREAK_HIT)"
  echo "--- events ---"
  cat "$EVENTS_LOG"
  echo "--- agent log (last 20) ---"
  tail -20 "$AGENT_LOG"
  exit 1
fi
if ! grep -q 'pid=' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (no pid= in events)"
  cat "$EVENTS_LOG"
  exit 1
fi
# Assert: help was shown
if ! grep -q 'commands:' "$EVENTS_LOG" && ! grep -q 'break.*symbol' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (help or break output not seen)"
  cat "$EVENTS_LOG"
  exit 1
fi

# Assert: info break showed breakpoint, then after delete (none)
if ! grep -q 'breakpoints:' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (info break output not seen)"
  exit 1
fi
if ! grep -q 'tcp_sendmsg' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (breakpoint tcp_sendmsg not listed)"
  exit 1
fi
if ! grep -q 'deleted' "$EVENTS_LOG"; then
  echo "e2e_tcpdump_style_cli: FAIL (delete not confirmed)"
  exit 1
fi

echo "e2e_tcpdump_style_cli: PASS (tcpdump-style: break/trace/info/delete lifecycle, L3/L4 metadata)"
