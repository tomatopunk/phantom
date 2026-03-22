/**
 * Copyright 2026 The Phantom Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

export type DebugEventPayload = {
  timestamp_ns: number;
  session_id: string;
  event_type: number;
  event_type_name: string;
  pid: number;
  tgid: number;
  cpu: number;
  probe_id: string;
  source_kind?: string;
  break_id?: string;
  hook_id?: string;
  template_probe_id?: string;
  payload_hex: string;
  payload_truncated: boolean;
  payload_utf8: string;
};

export type CpuJ = {
  label: string;
  user: number;
  nice: number;
  system: number;
  idle: number;
  iowait: number;
  irq: number;
  softirq: number;
  steal: number;
  guest: number;
  guest_nice: number;
};

export type NetDev = {
  name: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
  rx_dropped: number;
  tx_dropped: number;
};

export type TaskRow = {
  tid: number;
  name: string;
  state: string;
  vm_peak_kb: number;
  vm_size_kb: number;
  vm_rss_kb: number;
  vm_hwm_kb: number;
  threads_count: number;
};
