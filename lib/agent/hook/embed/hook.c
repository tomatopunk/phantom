#define __BPF_TRACING__
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>
#include "common.h"
#include "phantom_sock.h"
struct { __uint(type, BPF_MAP_TYPE_RINGBUF); __uint(max_entries, 256*1024); } events SEC(".maps");
SEC("kprobe")
int hook_handler(struct pt_regs *ctx) {
	__u64 ts = bpf_ktime_get_ns();
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 cpu = bpf_get_smp_processor_id();
	struct event_header ev = {
		.timestamp_ns = ts,
		.pid = (__u32)(pid_tgid >> 32),
		.tgid = (__u32)pid_tgid,
		.event_type = PHANTOM_EVENT_TYPE_BREAK_HIT,
		.cpu = cpu,
	};
	long arg0 = PT_REGS_PARM1(ctx);
	long arg1 = PT_REGS_PARM2(ctx);
	long arg2 = PT_REGS_PARM3(ctx);
	long arg3 = PT_REGS_PARM4(ctx);
	long arg4 = PT_REGS_PARM5(ctx);
	long arg5 = PT_REGS_PARM6(ctx);
	long ret = 0;
	(void)arg0; (void)arg1; (void)arg2; (void)arg3; (void)arg4; (void)arg5; (void)ret;
{{PROLOGUE}}
	/* user snippet: ctx, ev, arg0..arg5, ret; extra vars from registered prologue */
{{SNIPPET}}
	bpf_ringbuf_output(&events, &ev, sizeof(ev), 0);
	return 0;
}
char _license[] SEC("license") = "GPL";
