import type { DebugEventPayload } from "./types";

export const MAX_EVENTS = 8000;

export function hexLines(hex: string, bytesPerLine = 16): string[] {
  const lines: string[] = [];
  for (let i = 0; i < hex.length; i += bytesPerLine * 2) {
    const chunk = hex.slice(i, i + bytesPerLine * 2);
    const parts: string[] = [];
    for (let j = 0; j < chunk.length; j += 2) {
      parts.push(chunk.slice(j, j + 2));
    }
    const addr = (i / 2).toString(16).padStart(8, "0");
    lines.push(`${addr}  ${parts.join(" ")}`);
  }
  return lines;
}

export function eventMatchesFilter(ev: DebugEventPayload, q: string): boolean {
  if (!q.trim()) return true;
  const tokens = q.toLowerCase().trim().split(/\s+/);
  const hay = [
    ev.event_type_name,
    String(ev.event_type),
    String(ev.pid),
    String(ev.tgid),
    String(ev.cpu),
    ev.probe_id,
    ev.payload_utf8,
    ev.session_id,
  ]
    .join(" ")
    .toLowerCase();
  return tokens.every((tok) => hay.includes(tok));
}
