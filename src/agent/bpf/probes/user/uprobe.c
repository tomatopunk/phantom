/*
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

/* Minimal uprobe: submits event_header to ringbuf. Build on Linux with clang and kernel headers. */
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include "../../include/common.h"

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("uprobe")
int uprobe_handler(struct pt_regs *ctx)
{
	__u64 ts = bpf_ktime_get_ns();
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	__u32 tgid = (__u32)pid_tgid;
	__u32 cpu = bpf_get_smp_processor_id();

	struct event_header ev = {
		.timestamp_ns = ts,
		.session_id = 0,
		.event_type = PHANTOM_EVENT_TYPE_BREAK_HIT,
		.pid = pid,
		.tgid = tgid,
		.cpu = cpu,
		.probe_id = 0,
	};

	bpf_ringbuf_output(&events, &ev, sizeof(ev), 0);
	return 0;
}

char _license[] SEC("license") = "GPL";
