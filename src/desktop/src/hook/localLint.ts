/**
 * Copyright 2026 The Phantom Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

/** Phase A: attach / SEC consistency (no agent). */

export type LocalProblem = {
  line: number;
  column: number;
  message: string;
  source: "local";
};

const attachRe = /^(kprobe|tracepoint|uprobe|uretprobe):(.+)$/i;

export function lintAttachPoint(attach: string): string | null {
  const a = attach.trim();
  if (!a) return "attach 不能为空";
  const m = attachRe.exec(a);
  if (!m) return "attach 需为 kprobe:sym、tracepoint:sub:event、uprobe:/path:sym 或 uretprobe:…";
  const kind = m[1].toLowerCase();
  const rest = m[2].trim();
  if (!rest) return "attach 目标为空";
  if (kind === "tracepoint") {
    const parts = rest.split(":");
    if (parts.length < 2 || !parts[0] || !parts.slice(1).join(":"))
      return "tracepoint 需为 tracepoint:subsystem:event";
    const sub = parts[0];
    const ev = parts.slice(1).join(":");
    if (!/^[a-zA-Z0-9_]+$/.test(sub) || !/^[a-zA-Z0-9_]+$/.test(ev))
      return "tracepoint subsystem/event 仅允许字母数字下划线";
  }
  if (kind === "uprobe" || kind === "uretprobe") {
    const last = rest.lastIndexOf(":");
    if (last <= 0) return "uprobe 需为 uprobe:/abs/path:symbol";
    const path = rest.slice(0, last).trim();
    const sym = rest.slice(last + 1).trim();
    if (!path.startsWith("/") || !sym) return "uprobe 需为绝对路径:符号";
  }
  return null;
}

/** Warn if SEC("...") in source disagrees with attach kind (heuristic). */
export function lintSecVsAttach(source: string, attach: string): LocalProblem[] {
  const problems: LocalProblem[] = [];
  const m = attachRe.exec(attach.trim());
  if (!m) return problems;
  const kind = m[1].toLowerCase();
  const secMatches = [...source.matchAll(/SEC\s*\(\s*"([^"]+)"\s*\)/g)];
  for (const sm of secMatches) {
    const sec = sm[0];
    const inner = sm[1];
    const idx = source.indexOf(sec);
    const line = source.slice(0, idx).split("\n").length;
    if (kind === "kprobe" && inner.startsWith("tracepoint/")) {
      problems.push({
        line,
        column: 1,
        message: "attach 为 kprobe，但 SEC 像 tracepoint",
        source: "local",
      });
    }
    if (kind === "tracepoint" && (inner === "kprobe" || inner === "uprobe" || inner === "uretprobe")) {
      problems.push({
        line,
        column: 1,
        message: "attach 为 tracepoint，但 SEC 像 kprobe/uprobe",
        source: "local",
      });
    }
  }
  return problems;
}

/** Simple bracket / string balance for C-ish text (phase A light L2). */
export function lintBracketBalance(source: string): LocalProblem[] {
  const problems: LocalProblem[] = [];
  let depth = 0;
  let line = 1;
  let col = 0;
  let inStr: '"' | "'" | null = null;
  let escape = false;
  let inLineComment = false;
  let inBlock = 0;

  for (let i = 0; i < source.length; i++) {
    const c = source[i];
    const next = source[i + 1];
    col++;
    if (c === "\n") {
      line++;
      col = 0;
      inLineComment = false;
      continue;
    }
    if (inLineComment) continue;
    if (inBlock > 0) {
      if (c === "/" && next === "*") inBlock++;
      else if (c === "*" && next === "/") {
        inBlock--;
        i++;
        col++;
      }
      continue;
    }
    if (!inStr) {
      if (c === "/" && next === "/") {
        inLineComment = true;
        i++;
        col++;
        continue;
      }
      if (c === "/" && next === "*") {
        inBlock = 1;
        i++;
        col++;
        continue;
      }
      if (c === '"' || c === "'") {
        inStr = c;
        escape = false;
        continue;
      }
      if (c === "(" || c === "{" || c === "[") depth++;
      if (c === ")" || c === "}" || c === "]") {
        depth--;
        if (depth < 0) {
          problems.push({ line, column: col, message: "多余的闭合括号", source: "local" });
          depth = 0;
        }
      }
    } else {
      if (escape) {
        escape = false;
        continue;
      }
      if (c === "\\") {
        escape = true;
        continue;
      }
      if (c === inStr) inStr = null;
    }
  }
  if (inStr) problems.push({ line, column: col, message: "未闭合字符串", source: "local" });
  if (inBlock > 0) problems.push({ line, column: col, message: "未闭合块注释", source: "local" });
  if (depth !== 0) problems.push({ line, column: col, message: "括号/brace 不平衡", source: "local" });
  return problems;
}

export function runLocalLint(source: string, attach: string): LocalProblem[] {
  const out: LocalProblem[] = [];
  const att = attach.trim();
  if (att !== "") {
    const a = lintAttachPoint(att);
    if (a) out.push({ line: 1, column: 1, message: a, source: "local" });
    out.push(...lintSecVsAttach(source, att));
  }
  out.push(...lintBracketBalance(source));
  return out;
}
