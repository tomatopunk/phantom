// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import "fmt"

// MinimalKprobeRingbufC returns full eBPF C for a kprobe that emits one ringbuf sample per hit.
func MinimalKprobeRingbufC(kprobeSymbol string) string {
	return fmt.Sprintf(`#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
	__u32 pid;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("kprobe/%s")
int BPF_PROG(kprobe_hook, struct pt_regs *regs)
{
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		return 0;
	e->pid = bpf_get_current_pid_tgid() >> 32;
	bpf_ringbuf_submit(e, 0);
	return 0;
}
`, kprobeSymbol)
}

// MinimalTracepointRingbufC is a fixed tracepoint program (sched:sched_process_fork).
const MinimalTracepointRingbufC = `#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
	__u32 pid;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("tracepoint/sched/sched_process_fork")
int BPF_PROG(tp_sched_process_fork, void *ctx)
{
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		return 0;
	e->pid = bpf_get_current_pid_tgid() >> 32;
	bpf_ringbuf_submit(e, 0);
	return 0;
}
`

// MinimalUprobeRingbufC is full C for a uprobe; attach must be uprobe:/abs:symbol matching SEC.
const MinimalUprobeRingbufC = `#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
	__u32 pid;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("uprobe")
int uprobe_e2e_marker(struct pt_regs *ctx)
{
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		return 0;
	e->pid = bpf_get_current_pid_tgid() >> 32;
	bpf_ringbuf_submit(e, 0);
	return 0;
}
`
