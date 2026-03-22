/**
 * Build REPL commands from discovery list rows (tracepoint / kprobe / uprobe).
 */

export type DiscoverTab = "tp" | "kp" | "up";

/** Maps to session panel categories + REPL verbs. */
export type DiscoverProbeKind = "break" | "trace" | "hook" | "watch";

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
  traceExprs?: string;
  watchExpr?: string;
  /** User eBPF C for `break` / `tbreak` (CompileRaw). */
  breakUserSource?: string;
  /** Override attach point for break (default from discovery row, e.g. kprobe:sym). */
  breakAttach?: string;
  /** Optional BPF program function name for break. */
  breakProgram?: string;
};

const RINGBUF_MAP = `struct {
\t__uint(type, BPF_MAP_TYPE_RINGBUF);
\t__uint(max_entries, 256 * 1024);
} events SEC(".maps");`;

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

/** Default full-C for uprobe / uretprobe (program name \`uprobe_hook\`; pick via --program if needed). */
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

/** Full C source for an attach string such as \`kprobe:foo\` or \`tracepoint:a:b\`. */
export function defaultHookSourceForAttach(attach: string): string {
  const a = attach.trim();
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
 * REPL lines only (no hook attach: desktop uses compile_hook for full C).
 * For \`trace\` / \`watch\`, returns the trace/watch command only — load the hook via compile first in the Run panel.
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
  if (kind === "break" && (tab === "tp" || tab === "up")) return false;
  if (tab === "tp" && !parseTracepoint(trimmed)) return false;
  return true;
}

/**
 * Default options so `discoveryCommandForProbe` can build a `break` REPL line for kprobe rows
 * (full ringbuf C template). Hook/trace/watch do not need this for command text.
 */
export function defaultBreakOptsForDiscoveryKprobe(line: string): Pick<ProbeCommandOptions, "breakUserSource" | "breakAttach"> {
  const sym = line.trim();
  if (!sym || sym === ELLIPSIS) return {};
  return {
    breakUserSource: defaultHookSourceForKprobe(sym),
    breakAttach: `kprobe:${sym}`,
  };
}

export function buildProbeCommand(draft: ProbeRunDraft, opts: ProbeCommandOptions = {}): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  const traceExprs = opts.traceExprs ?? (draft.tab === "kp" ? "pid tgid comm" : "pid tgid");
  const watchExpr = opts.watchExpr ?? "pid";

  if (draft.tab === "kp") {
    if (draft.kind === "break") {
      const src = (opts.breakUserSource ?? "").trim();
      if (!src) return null;
      const attach = (opts.breakAttach ?? `kprobe:${trimmed}`).trim();
      if (!attach) return null;
      const prog = (opts.breakProgram ?? "").trim();
      let cmd = `break --attach ${attach} --source "${escapeForShellDoubleQuotes(src)}"`;
      if (prog) {
        cmd += ` --program "${escapeForShellDoubleQuotes(prog)}"`;
      }
      return cmd;
    }
    if (draft.kind === "trace") return `trace ${traceExprs}`;
    if (draft.kind === "hook") return null;
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    if (draft.kind === "break") return null;
    if (draft.kind === "trace") return `trace ${traceExprs}`;
    if (draft.kind === "hook") return null;
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  if (draft.kind === "break") return null;
  if (draft.kind === "trace") return `trace ${traceExprs}`;
  if (draft.kind === "hook") return null;
  if (draft.kind === "watch") return `watch ${watchExpr}`;
  return null;
}

export function discoveryCommandForProbe(
  tab: DiscoverTab,
  line: string,
  binaryPath: string,
  kind: DiscoverProbeKind,
): string | null {
  const breakDefaults = kind === "break" && tab === "kp" ? defaultBreakOptsForDiscoveryKprobe(line) : {};
  const lines = buildProbeRunLines({ tab, line, binaryPath, kind }, breakDefaults);
  if (lines.length === 0) return null;
  return lines.join("\n");
}

/** Attach point for hook compile when draft is \`hook\` (not trace/watch/break). */
export function templateAttachPointForDraft(draft: ProbeRunDraft): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;
  if (draft.kind === "trace" || draft.kind === "watch") return null;
  if (draft.kind === "break") return null;
  if (draft.tab === "kp") {
    if (draft.kind === "hook") return `kprobe:${trimmed}`;
    return null;
  }
  if (draft.tab === "tp") {
    if (draft.kind !== "hook") return null;
    const tp = parseTracepoint(trimmed);
    return tp ? `tracepoint:${tp.sub}:${tp.ev}` : null;
  }
  if (draft.kind === "hook") {
    const bin = draft.binaryPath.trim() || "/bin/sh";
    return `uprobe:${bin}:${trimmed}`;
  }
  return null;
}

/** Attach point for the probe that feeds trace/watch (same discovery row). */
export function templateAttachPointForEventSource(draft: ProbeRunDraft): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;
  if (draft.tab === "kp") return `kprobe:${trimmed}`;
  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    return tp ? `tracepoint:${tp.sub}:${tp.ev}` : null;
  }
  const bin = draft.binaryPath.trim() || "/bin/sh";
  return `uprobe:${bin}:${trimmed}`;
}

/** Hook compile attach point (hook row, or trace/watch paired attach). */
export function templateAttachPointForPreview(draft: ProbeRunDraft): string | null {
  if (draft.kind === "trace" || draft.kind === "watch") {
    return templateAttachPointForEventSource(draft);
  }
  return templateAttachPointForDraft(draft);
}
