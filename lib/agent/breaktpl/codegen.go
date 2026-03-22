// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package breaktpl

import (
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/agent/breakdsl"
)

// ProgramName returns a valid C token for the generated BPF program function.
func ProgramName(probeID string) string {
	s := strings.ReplaceAll(probeID, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return "phantom_br_" + s
}

// GenerateC produces full BPF C source for a catalog entry and optional filter DSL.
func GenerateC(entry *Entry, filterDSL string) (string, error) {
	allowed := make(map[string]bool)
	for _, p := range entry.FilterParams {
		allowed[strings.ToLower(strings.TrimSpace(p))] = true
	}
	isKprobe := entry.Kind == KindKprobe
	filterC, err := breakdsl.ToCFilter(filterDSL, allowed, isKprobe)
	if err != nil {
		return "", fmt.Errorf("filter: %w", err)
	}
	fn := ProgramName(entry.ProbeID)
	switch entry.Kind {
	case KindKprobe:
		return generateKprobeC(entry.KprobeSymbol, fn, filterC), nil
	case KindTracepoint:
		if entry.TraceGroup != "sched" || entry.TraceEvent != "sched_process_fork" {
			return "", fmt.Errorf("tracepoint %s/%s not supported by codegen", entry.TraceGroup, entry.TraceEvent)
		}
		return generateForkTracepointC(fn, filterC), nil
	default:
		return "", fmt.Errorf("unsupported kind")
	}
}

func generateKprobeC(symbol, fnName, filterC string) string {
	return fmt.Sprintf(`#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include "common.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

struct phantom_full_event {
	struct event_header h;
	__u64 args[6];
} __attribute__((packed));

SEC("kprobe/%s")
int %s(struct pt_regs *ctx)
{
%s
	__u64 ts = bpf_ktime_get_ns();
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	__u32 tgid = (__u32)pid_tgid;
	__u32 cpu = bpf_get_smp_processor_id();

	struct phantom_full_event ev = {};
	ev.h.timestamp_ns = ts;
	ev.h.session_id = 0;
	ev.h.event_type = PHANTOM_EVENT_TYPE_BREAK_HIT;
	ev.h.pid = pid;
	ev.h.tgid = tgid;
	ev.h.cpu = cpu;
	ev.h.probe_id = 0;

	ev.args[0] = (__u64)PT_REGS_PARM1(ctx);
	ev.args[1] = (__u64)PT_REGS_PARM2(ctx);
	ev.args[2] = (__u64)PT_REGS_PARM3(ctx);
	ev.args[3] = (__u64)PT_REGS_PARM4(ctx);
	ev.args[4] = (__u64)PT_REGS_PARM5(ctx);
	ev.args[5] = (__u64)PT_REGS_PARM6(ctx);

	bpf_ringbuf_output(&events, &ev, sizeof(ev), 0);
	return 0;
}
`, symbol, fnName, filterC)
}

func generateForkTracepointC(fnName, filterC string) string {
	return fmt.Sprintf(`#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include "common.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

struct phantom_full_event {
	struct event_header h;
	__u64 args[6];
} __attribute__((packed));

SEC("tracepoint/sched/sched_process_fork")
int %s(struct trace_event_raw_sched_process_fork *ctx)
{
%s
	__u64 ts = bpf_ktime_get_ns();
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	__u32 tgid = (__u32)pid_tgid;
	__u32 cpu = bpf_get_smp_processor_id();

	__u32 parent_pid = BPF_CORE_READ(ctx, parent_pid);
	__u32 child_pid = BPF_CORE_READ(ctx, child_pid);

	struct phantom_full_event ev = {};
	ev.h.timestamp_ns = ts;
	ev.h.session_id = 0;
	ev.h.event_type = PHANTOM_EVENT_TYPE_BREAK_HIT;
	ev.h.pid = pid;
	ev.h.tgid = tgid;
	ev.h.cpu = cpu;
	ev.h.probe_id = 0;

	ev.args[0] = parent_pid;
	ev.args[1] = child_pid;
	ev.args[2] = 0;
	ev.args[3] = 0;
	ev.args[4] = 0;
	ev.args[5] = 0;

	bpf_ringbuf_output(&events, &ev, sizeof(ev), 0);
	return 0;
}
`, fnName, filterC)
}
