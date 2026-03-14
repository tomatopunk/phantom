/* Minimal kprobe: submits event_header to ringbuf. Build on Linux with clang and kernel headers. */
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include "../../include/common.h"

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("kprobe")
int kprobe_handler(struct pt_regs *ctx)
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
