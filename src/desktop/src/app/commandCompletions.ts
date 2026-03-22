/**
 * REPL command suggestions for the desktop CLI input (Phantom agent execute).
 */

export type CommandSuggestResult = {
  replaceFrom: number;
  replaceTo: number;
  items: string[];
};

const INFO_SUB = ["break", "trace", "watch", "hook", "session"];

const HOOK_SUB = ["add", "attach", "delete", "list"];

const VERBS = [
  "break",
  "b",
  "tbreak",
  "print",
  "p",
  "trace",
  "t",
  "continue",
  "c",
  "delete",
  "disable",
  "enable",
  "condition",
  "info",
  "list",
  "bt",
  "watch",
  "help",
  "hook",
  "quit",
  "exit",
  "q",
];

const SNIPPETS = [
  "info break",
  "info trace",
  "info watch",
  "info hook",
  "info session",
  "help break",
  "help hook",
  "help trace",
  'hook add --point kprobe:do_nanosleep --lang c --sec "pid>0"',
  'hook add --point tracepoint:sched:sched_switch --lang c --sec "pid>0"',
  "hook list",
  "trace pid tgid comm",
  "watch tgid",
  "print pid",
  "continue",
];

function norm(s: string): string {
  return s.toLowerCase();
}

/**
 * Word (non-whitespace run) containing cursor; if cursor is immediately after whitespace, empty word at cursor.
 */
function wordBoundsAtCursor(line: string, cursor: number): { wordStart: number; wordEnd: number } {
  if (cursor > 0 && /\s/.test(line[cursor - 1])) {
    return { wordStart: cursor, wordEnd: cursor };
  }
  let wordStart = cursor;
  while (wordStart > 0 && !/\s/.test(line[wordStart - 1])) wordStart--;
  let wordEnd = cursor;
  while (wordEnd < line.length && !/\s/.test(line[wordEnd])) wordEnd++;
  return { wordStart, wordEnd };
}

function uniqLimited(items: string[], max: number): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const x of items) {
    if (!seen.has(x)) {
      seen.add(x);
      out.push(x);
      if (out.length >= max) break;
    }
  }
  return out;
}

export function commandSuggestionsAtCursor(line: string, cursor: number): CommandSuggestResult {
  const before = line.slice(0, cursor);
  const { wordStart, wordEnd } = wordBoundsAtCursor(line, cursor);
  const word = line.slice(wordStart, wordEnd);

  const infoHead = before.match(/^\s*info\s+/i);
  if (infoHead && infoHead.index !== undefined) {
    const kwEnd = infoHead.index + infoHead[0].length;
    const tok = before.trim().split(/\s+/);
    if (tok[0]?.toLowerCase() === "info" && tok.length > 2) {
      return { replaceFrom: cursor, replaceTo: cursor, items: [] };
    }
    if (cursor >= kwEnd) {
      const replaceFrom = kwEnd;
      const replaceTo = wordEnd < kwEnd ? cursor : wordEnd;
      const partial = line.slice(replaceFrom, replaceTo);
      if (!partial.includes(" ")) {
        const items = INFO_SUB.filter((s) => norm(s).startsWith(norm(partial)));
        return { replaceFrom, replaceTo, items: uniqLimited(items, 32) };
      }
    }
  }

  const hookHead = before.match(/^\s*hook\s+/i);
  if (hookHead && hookHead.index !== undefined) {
    const kwEnd = hookHead.index + hookHead[0].length;
    const tok = before.trim().split(/\s+/);
    if (tok[0]?.toLowerCase() === "hook" && tok.length > 2) {
      return { replaceFrom: cursor, replaceTo: cursor, items: [] };
    }
    if (cursor >= kwEnd) {
      const replaceFrom = kwEnd;
      const replaceTo = wordEnd < kwEnd ? cursor : wordEnd;
      const partial = line.slice(replaceFrom, replaceTo);
      if (!partial.includes(" ")) {
        const items = HOOK_SUB.filter((s) => norm(s).startsWith(norm(partial)));
        return { replaceFrom, replaceTo, items: uniqLimited(items, 32) };
      }
    }
  }

  const prior = line.slice(0, wordStart);
  if (prior.trim() !== "") {
    return { replaceFrom: cursor, replaceTo: cursor, items: [] };
  }

  const verbHits = VERBS.filter((v) => norm(v).startsWith(norm(word)));
  const snippetHits = SNIPPETS.filter((s) => norm(s).startsWith(norm(word)));
  const items = uniqLimited([...verbHits, ...snippetHits], 32);
  return { replaceFrom: wordStart, replaceTo: wordEnd, items };
}
