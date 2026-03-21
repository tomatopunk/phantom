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

# Sourced by e2e shell scripts (do not execute directly). Adjusts limits/caps so the
# agent can load BPF on constrained hosts (e.g. GitHub Actions):
#   - RLIMIT_MEMLOCK (cap_sys_resource): map creation / cilium/ebpf RemoveMemlock
#   - BPF load (cap_bpf): unprivileged BPF where enforced
# Kprobe kernel version uses uname in the agent (see lib/agent/runtime/kprobe_kernel_version_linux.go),
# not /proc/self/mem, so cap_sys_ptrace is not required for scripted e2e.
#
# GitHub Actions: file caps are not always enough for BPF program load memlock; scripts run the agent
# with sudo -n -E when GITHUB_ACTIONS is set (phantom_e2e_run_agent_sudo: bash ulimit -l then exec; -n avoids password wait on headless CI).

phantom_e2e_agent_needs_sudo() {
  [ -n "${GITHUB_ACTIONS:-}" ] && sudo -n true 2>/dev/null
}

phantom_e2e_soft_memlock() {
  ulimit -l unlimited 2>/dev/null || true
}

# Run agent as root with soft memlock raised before exec (CI: avoids inheriting a low locked-memory
# soft limit from the parent shell despite sudo).
phantom_e2e_run_agent_sudo() {
  local agent_bin="$1"
  shift
  sudo -n -E bash -c 'ulimit -l unlimited 2>/dev/null || true; exec "$0" "$@"' "$agent_bin" "$@"
}

# File capabilities persist across exec; needed when hard memlock limit cannot be raised (CI runners).
# Requires: setcap (libcap2-bin), passwordless sudo (CI) or manual skip locally.
phantom_e2e_grant_agent_file_caps() {
  local bin="$1"
  local tag="${2:-e2e}"
  [ -n "$bin" ] && [ -x "$bin" ] || return 0
  if ! command -v setcap >/dev/null 2>&1; then
    return 0
  fi
  # GitHub Actions: passwordless sudo without requiring `sudo -n` (some images differ).
  local ok_sudo=0
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    ok_sudo=1
  elif sudo -n true 2>/dev/null; then
    ok_sudo=1
  fi
  if [ "$ok_sudo" -eq 1 ] && sudo setcap cap_sys_resource,cap_bpf+ep "$bin" 2>/dev/null; then
    echo "${tag}: setcap cap_sys_resource,cap_bpf+ep on $(basename "$bin")" >&2
  fi
}

phantom_e2e_linux_bpf_env() {
  local bin="$1"
  local tag="$2"
  phantom_e2e_soft_memlock
  phantom_e2e_grant_agent_file_caps "$bin" "$tag"
}
