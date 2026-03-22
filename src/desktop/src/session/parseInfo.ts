/**
 * Copyright 2026 The Phantom Authors
 *
 * SPDX-License-Identifier: Apache-2.0
 */

/** Parse `info break|break-templates|hook|watch` plaintext from ExecuteResponse.output */

export function sliceSection(output: string, title: string): string[] {
  const needle = `${title}:\n`;
  const i = output.indexOf(needle);
  if (i < 0) return [];
  const rest = output.slice(i + needle.length);
  const lines = rest.split("\n");
  const res: string[] = [];
  for (const line of lines) {
    const t = line.trim();
    if (t === "") {
      if (res.length > 0) break;
      continue;
    }
    if (t === "(none)") return [];
    res.push(line.trimEnd().trim());
  }
  return res;
}

export type BreakRow = { id: string; symbol: string; enabled: boolean; condition?: string; temp?: boolean };

export function parseBreakLines(lines: string[]): BreakRow[] {
  const rows: BreakRow[] = [];
  for (const line of lines) {
    const temp = line.includes("(temp)");
    const trimmed = line.trim();
    const re = /^(\S+)(?:\s*\(temp\))?\s+probe_id=(\S+)\s+enabled=([yn])/;
    const m = re.exec(trimmed);
    if (!m) continue;
    let rest = trimmed.slice(m[0].length).trim();
    let condition: string | undefined;
    if (rest.startsWith("condition ")) {
      const fi = rest.indexOf(" filter=");
      if (fi >= 0) {
        condition = rest.slice("condition ".length, fi).trim();
        rest = rest.slice(fi).trim();
      } else {
        condition = rest.slice("condition ".length).trim();
        rest = "";
      }
    }
    rows.push({
      id: m[1],
      symbol: m[2],
      enabled: m[3] === "y",
      condition,
      temp,
    });
  }
  return rows;
}

export type CatalogTemplateRow = { line: string };

export function parseCatalogTemplateLines(lines: string[]): CatalogTemplateRow[] {
  return lines.map((line) => ({ line: line.trim() }));
}

export type HookRow = { id: string; probePoint: string; filter?: string; note?: string };

export function parseHookLines(lines: string[]): HookRow[] {
  return lines.map((line) => {
    let s = line.trim();
    let filter: string | undefined;
    let note: string | undefined;
    const fm = /\s+filter="((?:[^"\\]|\\.)*)"/.exec(s);
    if (fm) {
      filter = fm[1].replace(/\\"/g, '"').replace(/\\\\/g, "\\");
      s = (s.slice(0, fm.index) + s.slice(fm.index + fm[0].length)).replace(/\s+/g, " ").trim();
    }
    const nm = /\s+note=(\S+)$/.exec(s);
    if (nm) {
      note = nm[1];
      s = s.slice(0, nm.index).trimEnd();
    }
    s = s.replace(/\s+/g, " ").trim();
    const sp = s.indexOf(" ");
    if (sp < 0) return { id: s, probePoint: "", filter, note };
    return { id: s.slice(0, sp), probePoint: s.slice(sp + 1).trim(), filter, note };
  });
}

export type WatchRow = { id: string; probeId: string; paramText: string };

export function parseWatchLines(lines: string[]): WatchRow[] {
  const rows: WatchRow[] = [];
  for (const line of lines) {
    const m = /^(\S+)\s+probe_id=(\S+)\s+param_indices=(.+)$/.exec(line.trim());
    if (m) {
      rows.push({ id: m[1], probeId: m[2], paramText: m[3].trim() });
      continue;
    }
    const sp = line.indexOf(" ");
    if (sp < 0) rows.push({ id: line.trim(), probeId: "", paramText: "" });
    else rows.push({ id: line.slice(0, sp), probeId: "", paramText: line.slice(sp + 1).trim() });
  }
  return rows;
}
