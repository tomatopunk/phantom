/** Parse `info break|trace|hook|watch` plaintext from ExecuteResponse.output */

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
    const re =
      /^(\S+)(?:\s*\(temp\))?\s+(\S+)\s+enabled=([yn])(?:\s+condition\s+(.*))?$/;
    const m = re.exec(line.trim());
    if (!m) continue;
    rows.push({
      id: m[1],
      symbol: m[2],
      enabled: m[3] === "y",
      condition: m[4]?.trim() || undefined,
      temp,
    });
  }
  return rows;
}

export type TraceRow = { id: string; expressions: string };

export function parseTraceLines(lines: string[]): TraceRow[] {
  return lines.map((line) => {
    const sp = line.indexOf(" ");
    if (sp < 0) return { id: line, expressions: "" };
    return { id: line.slice(0, sp), expressions: line.slice(sp + 1).trim() };
  });
}

export type HookRow = { id: string; attach: string; filter?: string; note?: string };

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
    if (sp < 0) return { id: s, attach: "", filter, note };
    return { id: s.slice(0, sp), attach: s.slice(sp + 1).trim(), filter, note };
  });
}

export type WatchRow = { id: string; expression: string; last: string };

export function parseWatchLines(lines: string[]): WatchRow[] {
  const rows: WatchRow[] = [];
  for (const line of lines) {
    const lastIdx = line.lastIndexOf(" last=");
    if (lastIdx < 0) {
      const sp = line.indexOf(" ");
      if (sp < 0) rows.push({ id: line, expression: "", last: "" });
      else rows.push({ id: line.slice(0, sp), expression: line.slice(sp + 1).trim(), last: "" });
      continue;
    }
    const last = line.slice(lastIdx + " last=".length).trim();
    const before = line.slice(0, lastIdx);
    const sp = before.indexOf(" ");
    if (sp < 0) rows.push({ id: before, expression: "", last });
    else rows.push({ id: before.slice(0, sp), expression: before.slice(sp + 1).trim(), last });
  }
  return rows;
}
