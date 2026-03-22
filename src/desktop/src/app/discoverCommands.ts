/**
 * Build REPL commands from discovery list rows (tracepoint / kprobe / uprobe).
 */

export type DiscoverTab = "tp" | "kp" | "up";

/** Maps to session panel categories + REPL verbs. */
export type DiscoverProbeKind = "break" | "trace" | "hook" | "watch";

/** How `hook add` supplies the template body: DSL filter vs your C snippet. */
export type HookBodyMode = "sec" | "code";

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
  /** `--sec` DSL when hook body mode is `sec` (default pid>0). */
  hookSec?: string;
  traceExprs?: string;
  watchExpr?: string;
  hookBodyMode?: HookBodyMode;
  /** Template body when `hookBodyMode === "code"` (mutually exclusive with --sec on agent). */
  hookCodeSnippet?: string;
  /** User eBPF C for `break` / `tbreak` (CompileRaw). */
  breakUserSource?: string;
  /** Override attach point for break (default from discovery row, e.g. kprobe:sym). */
  breakAttach?: string;
  /** Optional BPF program function name for break. */
  breakProgram?: string;
};

/**
 * One `hook add` line: exactly one of sec (DSL) or code (snippet), per agent rules.
 */
export function buildHookAddCommandLine(point: string, opts: ProbeCommandOptions): string | null {
  const mode: HookBodyMode = opts.hookBodyMode ?? "sec";
  if (mode === "code") {
    const c = (opts.hookCodeSnippet ?? "").trim();
    if (!c) return null;
    return `hook add --point ${point} --lang c --code "${escapeForShellDoubleQuotes(c)}"`;
  }
  const sec = (opts.hookSec ?? "pid>0").trim();
  if (!sec) return null;
  return `hook add --point ${point} --lang c --sec "${escapeForShellDoubleQuotes(sec)}"`;
}

/**
 * Attaches a probe for the discovery row so ringbuf events are produced.
 * Used to prefix `trace` / `watch` (those verbs alone do not load eBPF).
 * Always uses **`hook add`** (never `break`) so trace/watch stay separate from breakpoint semantics.
 */
export function buildAttachCommandForDiscoveryRow(
  draft: ProbeRunDraft,
  opts: ProbeCommandOptions = {},
): string | null {
  const trimmed = draft.line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  const hookSec = opts.hookSec ?? "pid>0";

  if (draft.tab === "kp") {
    return buildHookAddCommandLine(`kprobe:${trimmed}`, { ...opts, hookBodyMode: "sec", hookSec });
  }

  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    return buildHookAddCommandLine(`tracepoint:${tp.sub}:${tp.ev}`, { ...opts, hookBodyMode: "sec", hookSec });
  }

  const bin = draft.binaryPath.trim() || "/bin/sh";
  return buildHookAddCommandLine(`uprobe:${bin}:${trimmed}`, { ...opts, hookBodyMode: "sec", hookSec });
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
    if (draft.kind === "hook") {
      const line = buildHookAddCommandLine(`kprobe:${trimmed}`, opts);
      return line;
    }
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  if (draft.tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    if (draft.kind === "break") return null;
    if (draft.kind === "trace") return `trace ${traceExprs}`;
    if (draft.kind === "hook") {
      return buildHookAddCommandLine(`tracepoint:${tp.sub}:${tp.ev}`, opts);
    }
    if (draft.kind === "watch") return `watch ${watchExpr}`;
    return null;
  }

  const bin = draft.binaryPath.trim() || "/bin/sh";
  if (draft.kind === "break") return null;
  if (draft.kind === "trace") return `trace ${traceExprs}`;
  if (draft.kind === "hook") {
    return buildHookAddCommandLine(`uprobe:${bin}:${trimmed}`, opts);
  }
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
  // User-program `break` does not use the hook template preview.
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

/** Hook / compile preview attach point (includes trace/watch paired attach). */
export function templateAttachPointForPreview(draft: ProbeRunDraft): string | null {
  if (draft.kind === "trace" || draft.kind === "watch") {
    return templateAttachPointForEventSource(draft);
  }
  return templateAttachPointForDraft(draft);
}

/**
 * Sec expression or code snippet for PreviewHookTemplate (mutually exclusive on agent).
 */
export function templatePreviewSecAndCode(
  draft: ProbeRunDraft,
  hookSec: string,
  _breakKernelSecUnused: string,
  hookBodyMode: HookBodyMode,
  hookCodeSnippet: string,
): { sec: string; code: string } {
  if (draft.kind === "trace" || draft.kind === "watch") {
    return { sec: hookSec.trim() || "pid>0", code: "" };
  }
  if (draft.kind === "break") {
    return { sec: "", code: "" };
  }
  if (draft.kind === "hook") {
    if (hookBodyMode === "code") {
      return { sec: "", code: hookCodeSnippet };
    }
    return { sec: hookSec.trim() || "pid>0", code: "" };
  }
  return { sec: "", code: "" };
}

/** True when preview API can run (attach + sec xor code). */
export function templatePreviewReady(
  draft: ProbeRunDraft,
  hookSec: string,
  breakKernelSec: string,
  hookBodyMode: HookBodyMode,
  hookCodeSnippet: string,
): boolean {
  if (draft.kind === "break") return false;
  const attach = templateAttachPointForPreview(draft);
  if (!attach) return false;
  const { sec, code } = templatePreviewSecAndCode(draft, hookSec, breakKernelSec, hookBodyMode, hookCodeSnippet);
  return (sec !== "" && code === "") || (sec === "" && code.trim() !== "");
}

