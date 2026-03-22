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

export type ProbeCommandOptions = {
  /** `--sec` DSL for template hook (discover default: pid>0). */
  hookSec?: string;
  /** Kernel `--sec` for built-in `break` / `tbreak` (default pid>=0). */
  breakKernelSec?: string;
  traceExprs?: string;
  watchExpr?: string;
};

/**
 * Build a command for the given row and probe kind. Returns null if the row is invalid or kind unsupported.
 */
/**
 * Attaches a probe for the discovery row so ringbuf events are produced.
 * Used to prefix `trace` / `watch` (those verbs alone do not load eBPF).
 */
export function buildAttachCommandForDiscoveryRow(
  draft: ProbeRunDraft,
  opts: ProbeCommandOptions = {},
): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  const hookSec = opts.hookSec ?? "pid>0";
  const breakKernel = (opts.breakKernelSec ?? "pid>=0").trim();

  if (draft.tab === "kp") {
    if (!breakKernel || breakKernel === "pid>=0") return `break ${trimmed}`;
    return `break ${trimmed} --sec "${breakKernel}"`;
  }

  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    return `hook add --point tracepoint:${tp.sub}:${tp.ev} --lang c --sec "${hookSec}"`;
  }

  const bin = draft.binaryPath.trim() || "/bin/sh";
  return `hook add --point uprobe:${bin}:${trimmed} --lang c --sec "${hookSec}"`;
}

/**
 * Commands to run in order. For `trace` / `watch`, includes attach first, then the verb
 * (REPL accepts one line per Execute; the desktop runs these sequentially).
 */
export function buildProbeRunLines(draft: ProbeRunDraft, opts: ProbeCommandOptions = {}): string[] {
  const primary = buildProbeCommand(draft, opts);
  if (!primary) return [];
  if (draft.kind !== "trace" && draft.kind !== "watch") {
    return [primary];
  }
  const attach = buildAttachCommandForDiscoveryRow(draft, opts);
  if (!attach) return [primary];
  return [attach, primary];
}

export function buildProbeCommand(draft: ProbeRunDraft, opts: ProbeCommandOptions = {}): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  const hookSec = opts.hookSec ?? "pid>0";
  const breakKernel = (opts.breakKernelSec ?? "pid>=0").trim();
  const traceExprs = opts.traceExprs ?? (draft.tab === "kp" ? "pid tgid comm" : "pid tgid");
  const watchExpr = opts.watchExpr ?? "pid";

  if (draft.tab === "kp") {
    if (draft.kind === "break") {
      if (!breakKernel || breakKernel === "pid>=0") return `break ${trimmed}`;
      return `break ${trimmed} --sec "${breakKernel}"`;
    }
    if (draft.kind === "trace") return `trace ${traceExprs}`;
    if (draft.kind === "hook") return `hook add --point kprobe:${trimmed} --lang c --sec "${hookSec}"`;
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    if (draft.kind === "break") return null;
    if (draft.kind === "trace") return `trace ${traceExprs}`;
    if (draft.kind === "hook") return `hook add --point tracepoint:${tp.sub}:${tp.ev} --lang c --sec "${hookSec}"`;
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  const bin = draft.binaryPath.trim() || "/bin/sh";
  if (draft.kind === "break") return null;
  if (draft.kind === "trace") return `trace ${traceExprs}`;
  if (draft.kind === "hook") return `hook add --point uprobe:${bin}:${trimmed} --lang c --sec "${hookSec}"`;
  if (draft.kind === "watch") return `watch ${watchExpr}`;
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

/**
 * Attach point for template preview (agent PreviewHookTemplate). Null when no generated eBPF C (trace/watch).
 */
export function templateAttachPointForDraft(draft: ProbeRunDraft): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;
  if (draft.kind === "trace" || draft.kind === "watch") return null;
  if (draft.tab === "kp") {
    if (draft.kind === "break" || draft.kind === "hook") return `kprobe:${trimmed}`;
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

/** Hook / compile preview attach point (includes trace/watch paired attach). */
export function templateAttachPointForPreview(draft: ProbeRunDraft): string | null {
  if (draft.kind === "trace" || draft.kind === "watch") {
    return templateAttachPointForEventSource(draft);
  }
  return templateAttachPointForDraft(draft);
}

/** Sec DSL matching agent break/tbreak template vs default hook discover filter. */
export function templateSecForDraft(draft: ProbeRunDraft, hookSec: string, breakKernelSec = "pid>=0"): string | null {
  const attach = templateAttachPointForPreview(draft);
  if (!attach) return null;
  if (draft.kind === "break") return breakKernelSec.trim() || "pid>=0";
  if (draft.kind === "hook") return hookSec.trim() || "pid>0";
  if (draft.kind === "trace" || draft.kind === "watch") {
    if (draft.tab === "kp") return breakKernelSec.trim() || "pid>=0";
    return hookSec.trim() || "pid>0";
  }
  return null;
}
