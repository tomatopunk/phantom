import type { TFunction } from "i18next";

type Tab = "tp" | "kp" | "up";

type Props = {
  t: TFunction;
  discTab: Tab;
  setDiscTab: (k: Tab) => void;
  discPrefix: string;
  setDiscPrefix: (s: string) => void;
  discBin: string;
  setDiscBin: (s: string) => void;
  discLines: string[];
  runDiscover: () => void;
  connected: boolean;
};

function discTabLabel(t: TFunction, k: Tab) {
  return k === "tp" ? t("discover.tracepoint") : k === "kp" ? t("discover.kprobe") : t("discover.uprobe");
}

export function DiscoverPanel({
  t,
  discTab,
  setDiscTab,
  discPrefix,
  setDiscPrefix,
  discBin,
  setDiscBin,
  discLines,
  runDiscover,
  connected,
}: Props) {
  return (
    <div className="border-b border-shell-border p-2 shrink-0 flex flex-col max-h-[30vh] min-h-[120px]">
      <div className="flex gap-1 mb-1">
        {(["tp", "kp", "up"] as const).map((k) => (
          <button
            key={k}
            type="button"
            className={`px-2 py-0.5 rounded text-xs border ${
              discTab === k ? "bg-white/10 border-shell-border" : "border-transparent hover:bg-white/5"
            }`}
            onClick={() => setDiscTab(k)}
          >
            {discTabLabel(t, k)}
          </button>
        ))}
      </div>
      <div className="flex gap-1 mb-1">
        <input
          className="flex-1 bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
          value={discPrefix}
          onChange={(e) => setDiscPrefix(e.target.value)}
          placeholder={t("discover.prefixPh")}
        />
        {discTab === "up" && (
          <input
            className="flex-1 bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
            value={discBin}
            onChange={(e) => setDiscBin(e.target.value)}
            placeholder={t("discover.binaryPh")}
          />
        )}
        <button
          type="button"
          disabled={!connected}
          className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
          onClick={runDiscover}
        >
          {t("discover.list")}
        </button>
      </div>
      <ul className="flex-1 overflow-auto font-mono-tight text-[10px] text-gray-400 space-y-0.5">
        {discLines.map((line, i) => (
          <li
            key={`${i}-${line.slice(0, 12)}`}
            className="cursor-pointer hover:text-shell-accent truncate"
            title={line}
            onClick={() => navigator.clipboard.writeText(line)}
          >
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}
