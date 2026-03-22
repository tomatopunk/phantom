/**
 * Build REPL commands from discovery list rows (tracepoint / kprobe / uprobe).
 */

export type DiscoverTab = "tp" | "kp" | "up";

/** Maps to session panel categories + REPL verbs. */
export type DiscoverProbeKind = "break" | "hook" | "watch";

/** Payload when opening the probe run composer from discovery. */
export type ProbeRunDraft = {
  tab: DiscoverTab;
  line: string;
  binaryPath: string;
  kind: DiscoverProbeKind;
};

const ELLIPSIS = "…";

function parseTracepoint(line: string): { sub: string; ev: string } | null {
  const trimmed = line.trim();
  const slash = trimmed.indexOf("/");
  if (slash <= 0 || slash === trimmed.length - 1) return null;
  return { sub: trimmed.slice(0, slash), ev: trimmed.slice(slash + 1) };
}

/** Escape for a token inside double quotes (agent `splitCommandLine` rules). */
export function escapeForShellDoubleQuotes(s: string): string {
  return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

export type ProbeCommandOptions = {
  /** Kernel predicate DSL for template break. */
  breakFilter?: string;
  breakLimit?: number;
  /** Comma-separated param indices for watch --args. */
  watchArgs?: string;
};

const RINGBUF_MAP = `struct {
\t__uint(type, BPF_MAP_TYPE_RINGBUF);
\t__uint(max_entries, 256 * 1024);
} events SEC(".maps");`;

/** Map a discovery row to a catalog probe_id when the agent has a template for it. */
export function catalogProbeIdForDiscoveryRow(tab: DiscoverTab, line: string): string | null {
  const t = line.trim();
  if (!t || t === ELLIPSIS) return null;
  if (tab === "kp") return `kprobe.${t}`;
  if (tab === "tp") {
    const p = parseTracepoint(t);
    if (!p) return null;
    if (p.sub === "sched" && p.ev === "sched_process_fork") return "tp.sched.sched_process_fork";
  }
  return null;
}

/** Default full-C hook for `kprobe:symbol` (ringbuf + pid). */
export function defaultHookSourceForKprobe(symbol: string): string {
  const sym = symbol.trim();
  return `#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
\t__u32 pid;
};

${RINGBUF_MAP}

SEC("kprobe/${sym}")
int BPF_PROG(kprobe_hook, struct pt_regs *regs)
{
\tstruct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
\tif (!e)
\t\treturn 0;
\te->pid = bpf_get_current_pid_tgid() >> 32;
\tbpf_ringbuf_submit(e, 0);
\treturn 0;
}
`;
}

/** Default full-C hook for `tracepoint:sub:event`. */
export function defaultHookSourceForTracepoint(sub: string, ev: string): string {
  return `#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
\t__u32 pid;
};

${RINGBUF_MAP}

SEC("tracepoint/${sub}/${ev}")
int BPF_PROG(tp_hook, void *ctx)
{
\tstruct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
\tif (!e)
\t\treturn 0;
\te->pid = bpf_get_current_pid_tgid() >> 32;
\tbpf_ringbuf_submit(e, 0);
\treturn 0;
}
`;
}

/** Default full-C for uprobe / uretprobe. */
export const defaultHookSourceForUprobe = `#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct event {
\t__u32 pid;
};

${RINGBUF_MAP}

SEC("uprobe")
int uprobe_hook(struct pt_regs *ctx)
{
\tstruct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
\tif (!e)
\t\treturn 0;
\te->pid = bpf_get_current_pid_tgid() >> 32;
\tbpf_ringbuf_submit(e, 0);
\treturn 0;
}
`;

/** Full C source for a legacy probe_point string (hook only). */
export function defaultHookSourceForProbePoint(probePoint: string): string {
  const a = probePoint.trim();
  if (a.startsWith("kprobe:")) {
    return defaultHookSourceForKprobe(a.slice("kprobe:".length));
  }
  if (a.startsWith("tracepoint:")) {
    const rest = a.slice("tracepoint:".length);
    const idx = rest.indexOf(":");
    if (idx <= 0) return defaultHookSourceForKprobe("do_nanosleep");
    const sub = rest.slice(0, idx);
    const ev = rest.slice(idx + 1);
    return defaultHookSourceForTracepoint(sub, ev);
  }
  if (a.startsWith("uprobe:") || a.startsWith("uretprobe:")) {
    return defaultHookSourceForUprobe;
  }
  return defaultHookSourceForKprobe("do_nanosleep");
}

/**
 * REPL lines only. Hook loads use compile_hook in the Run panel.
 */
export function buildProbeRunLines(draft: ProbeRunDraft, opts: ProbeCommandOptions = {}): string[] {
  const primary = buildProbeCommand(draft, opts);
  if (!primary) return [];
  return [primary];
}

/** Whether a discovery row can use this quick action (Run panel / optional REPL). */
export function discoveryQuickActionAvailable(
  tab: DiscoverTab,
  line: string,
  _binaryPath: string,
  kind: DiscoverProbeKind,
): boolean {
  const trimmed = line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return false;
  if (kind === "break") {
    return catalogProbeIdForDiscoveryRow(tab, line) != null;
  }
  if (kind === "watch") {
    return catalogProbeIdForDiscoveryRow(tab, line) != null;
  }
  if (tab === "tp" && !parseTracepoint(trimmed)) return false;
  return true;
}

export function buildProbeCommand(draft: ProbeRunDraft, opts: ProbeCommandOptions = {}): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  const probeId = catalogProbeIdForDiscoveryRow(draft.tab, draft.line);

  if (draft.kind === "break") {
    if (!probeId) return null;
    let cmd = `break ${probeId}`;
    const f = (opts.breakFilter ?? "").trim();
    if (f) cmd += ` --filter "${escapeForShellDoubleQuotes(f)}"`;
    if (opts.breakLimit != null && opts.breakLimit >= 0) cmd += ` --limit ${opts.breakLimit}`;
    return cmd;
  }

  if (draft.kind === "watch") {
    if (!probeId) return null;
    let cmd = `watch --sec ${probeId}`;
    const wa = (opts.watchArgs ?? "").trim();
    if (wa) cmd += ` --args ${wa}`;
    return cmd;
  }

  if (draft.kind === "hook") return null;
  return null;
}

export function discoveryCommandForProbe(
  tab: DiscoverTab,
  line: string,
  binaryPath: string,
  kind: DiscoverProbeKind,
): string | null {
  const lines = buildProbeRunLines({ tab, line, binaryPath, kind }, {});
  if (lines.length === 0) return null;
  return lines.join("\n");
}

/** Hint text: ELF SEC-derived probe_point for hook C templates (informational). */
export function templateProbePointHintForDraft(draft: ProbeRunDraft): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;
  if (draft.kind !== "hook") return null;
  if (draft.tab === "kp") return `kprobe:${trimmed}`;
  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    return tp ? `tracepoint:${tp.sub}:${tp.ev}` : null;
  }
  const bin = draft.binaryPath.trim() || "/bin/sh";
  return `uprobe:${bin}:${trimmed}`;
}
