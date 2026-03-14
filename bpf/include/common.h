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
