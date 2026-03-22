const STORAGE_KEY = "phantom-desktop.agent-address-history.v1";
const MAX_ENTRIES = 10;

export function loadAgentHistory(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    const out: string[] = [];
    for (const x of parsed) {
      if (typeof x === "string" && x.trim()) out.push(x.trim());
    }
    return out.slice(0, MAX_ENTRIES);
  } catch {
    return [];
  }
}

/** Remember a successfully used agent address; MRU order, cap at 10, deduped. */
export function rememberAgentAddress(agent: string): string[] {
  const a = agent.trim();
  if (!a) return loadAgentHistory();
  const prev = loadAgentHistory().filter((x) => x !== a);
  const next = [a, ...prev].slice(0, MAX_ENTRIES);
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    /* quota / private mode */
  }
  return next;
}
