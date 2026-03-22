/**
 * Build a suggested REPL command from a discovery list row (tracepoint / kprobe / uprobe).
 */

export type DiscoverTab = "tp" | "kp" | "up";

const ELLIPSIS = "…";

export function suggestedCommandForDiscoveryRow(tab: DiscoverTab, line: string, binaryPath: string): string | null {
  const trimmed = line.trim();
  if (!trimmed || trimmed === ELLIPSIS) return null;

  if (tab === "kp") {
    return `break ${trimmed}`;
  }

  if (tab === "tp") {
    const slash = trimmed.indexOf("/");
    if (slash <= 0 || slash === trimmed.length - 1) return null;
    const sub = trimmed.slice(0, slash);
    const ev = trimmed.slice(slash + 1);
    return `hook add --point tracepoint:${sub}:${ev} --lang c --sec "pid>0"`;
  }

  const bin = binaryPath.trim() || "/bin/sh";
  return `hook add --point uprobe:${bin}:${trimmed} --lang c --sec "pid>0"`;
}
