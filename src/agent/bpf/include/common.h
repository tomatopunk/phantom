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

/* Shared definitions for eBPF programs (event layout, map keys). */

#ifndef PHANTOM_BPF_INCLUDE_COMMON_H
#define PHANTOM_BPF_INCLUDE_COMMON_H

#ifndef __u64
#define __u64 unsigned long long
#endif
#ifndef __u32
#define __u32 unsigned int
#endif

#define PHANTOM_EVENT_HEADER_SIZE 32
/* Header + six u64 args (break/watch template ringbuf record). */
#define PHANTOM_RING_RECORD_SIZE 80
#define PHANTOM_EVENT_TYPE_BREAK_HIT 1

struct event_header {
	__u64 timestamp_ns;
	__u32 session_id;
	__u32 event_type;
	__u32 pid;
	__u32 tgid;
	__u32 cpu;
	__u32 probe_id;
};

#endif /* PHANTOM_BPF_INCLUDE_COMMON_H */
