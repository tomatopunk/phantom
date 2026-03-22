/**
 * Build REPL commands from discovery list rows (tracepoint / kprobe / uprobe).
 */

export type DiscoverTab = "tp" | "kp" | "up";

/** Maps to session panel categories + REPL verbs. */
export type DiscoverProbeKind = "break" | "trace" | "hook" | "watch";

const ELLIPSIS = "…";

function parseTracepoint(line: string): { sub: string; ev: string } | null {
  const trimmed = line.trim();
  const slash = trimmed.indexOf("/");
  if (slash <= 0 || slash === trimmed.length - 1) return null;
  return { sub: trimmed.slice(0, slash), ev: trimmed.slice(slash + 1) };
}

/**
 * Build a command for the given row and probe kind. Returns null if the row is invalid or kind unsupported.
 */
export function discoveryCommandForProbe(
  tab: DiscoverTab,
  line: string,
  binaryPath: string,
  kind: DiscoverProbeKind,
): string | null {
  const trimmed = line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  if (tab === "kp") {
    if (kind === "break") return `break ${trimmed}`;
    if (kind === "trace") return `trace pid tgid comm`;
    if (kind === "hook") return `hook add --point kprobe:${trimmed} --lang c --sec "pid>0"`;
    if (kind === "watch") return `watch pid`;
    return null;
  }

  if (tab === "tp") {
    const tp = parseTracepoint(trimmed);
    if (!tp) return null;
    if (kind === "break") return null;
    if (kind === "trace") return `trace pid tgid`;
    if (kind === "hook") return `hook add --point tracepoint:${tp.sub}:${tp.ev} --lang c --sec "pid>0"`;
    if (kind === "watch") return `watch pid`;
    return null;
  }

  const bin = binaryPath.trim() || "/bin/sh";
  if (kind === "break") return null;
  if (kind === "trace") return `trace pid tgid`;
  if (kind === "hook") return `hook add --point uprobe:${bin}:${trimmed} --lang c --sec "pid>0"`;
  if (kind === "watch") return `watch pid`;
  return null;
}
