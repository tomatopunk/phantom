import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import * as api from "../api";
import {
  parseBreakLines,
  parseHookLines,
  parseTraceLines,
  parseWatchLines,
  sliceSection,
} from "../session/parseInfo";

type Tab = "break" | "trace" | "hook" | "watch";

export function SessionProbesPanel({
  connected,
  refreshTrigger = 0,
}: {
  connected: boolean;
  /** Increment from parent after attach/delete elsewhere to re-fetch. */
  refreshTrigger?: number;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>("break");
  const [raw, setRaw] = useState<Record<Tab, string>>({
    break: "",
    trace: "",
    hook: "",
    watch: "",
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const refresh = useCallback(async () => {
    if (!connected) return;
    setBusy(true);
    setErr("");
    try {
      const [b, tr, h, w] = await Promise.all([
        api.executeCmd("info break"),
        api.executeCmd("info trace"),
        api.executeCmd("info hook"),
        api.executeCmd("info watch"),
      ]);
      const pick = (r: { output: string }) => r.output || "";
      setRaw({
        break: pick(b),
        trace: pick(tr),
        hook: pick(h),
        watch: pick(w),
      });
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }, [connected]);

  const runCmd = async (line: string) => {
    setBusy(true);
    setErr("");
    try {
      await api.executeCmd(line);
      await refresh();
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  };

  const breakLines = sliceSection(raw.break, "breakpoints");
  const traceLines = sliceSection(raw.trace, "traces");
  const hookLines = sliceSection(raw.hook, "hooks");
  const watchLines = sliceSection(raw.watch, "watches");

  const breaks = parseBreakLines(breakLines);
  const traces = parseTraceLines(traceLines);
  const hooks = parseHookLines(hookLines);
  const watches = parseWatchLines(watchLines);

  useEffect(() => {
    if (connected) void refresh();
  }, [connected, refresh, refreshTrigger]);

  return (
    <div className="border-t border-shell-border p-2 space-y-1 flex flex-col min-h-[120px] max-h-[200px]">
      <div className="flex items-center gap-1 flex-wrap">
        <span className="text-xs text-shell-muted">{t("sessionPanel.title")}</span>
        <button
          type="button"
          disabled={!connected || busy}
          className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
          onClick={() => void refresh()}
        >
          {t("sessionPanel.refresh")}
        </button>
      </div>
      <div className="flex gap-1 flex-wrap">
        {(["break", "trace", "hook", "watch"] as const).map((k) => (
          <button
            key={k}
            type="button"
            className={`px-2 py-0.5 rounded text-xs border ${
              tab === k ? "bg-white/10 border-shell-border" : "border-transparent hover:bg-white/5"
            }`}
            onClick={() => setTab(k)}
          >
            {t(`sessionPanel.tab.${k}`)}
          </button>
        ))}
      </div>
      {err && <p className="text-[10px] text-amber-400">{err}</p>}
      <div className="flex-1 overflow-auto text-[10px] font-mono-tight">
        {tab === "break" &&
          breaks.map((r) => (
            <div key={r.id} className="flex flex-wrap gap-1 items-center border-b border-shell-border/30 py-0.5">
              <span className="text-shell-accent">{r.id}</span>
              <span className="truncate max-w-[140px]" title={r.symbol}>
                {r.symbol}
              </span>
              <span className="text-shell-muted">{r.enabled ? "on" : "off"}</span>
              {r.condition && <span className="text-gray-500 truncate max-w-[100px]">{r.condition}</span>}
              <span className="ml-auto flex gap-0.5 shrink-0">
                <button type="button" className="text-blue-400 hover:underline" onClick={() => void runCmd(`delete ${r.id}`)}>
                  del
                </button>
                <button type="button" className="text-blue-400 hover:underline" onClick={() => void runCmd(`disable ${r.id}`)}>
                  off
                </button>
                <button type="button" className="text-blue-400 hover:underline" onClick={() => void runCmd(`enable ${r.id}`)}>
                  on
                </button>
              </span>
            </div>
          ))}
        {tab === "trace" &&
          traces.map((r) => (
            <div key={r.id} className="flex gap-1 items-center border-b border-shell-border/30 py-0.5">
              <span className="text-shell-accent">{r.id}</span>
              <span className="truncate flex-1">{r.expressions}</span>
              <button type="button" className="text-blue-400 hover:underline shrink-0" onClick={() => void runCmd(`delete ${r.id}`)}>
                del
              </button>
            </div>
          ))}
        {tab === "hook" &&
          hooks.map((r) => (
            <div key={r.id} className="flex flex-col gap-0.5 border-b border-shell-border/30 py-0.5">
              <div className="flex gap-1 items-center">
                <span className="text-shell-accent">{r.id}</span>
                <span className="truncate flex-1" title={r.attach}>
                  {r.attach}
                </span>
                <button type="button" className="text-blue-400 hover:underline shrink-0" onClick={() => void runCmd(`hook delete ${r.id}`)}>
                  del
                </button>
              </div>
              {(r.filter || r.note) && (
                <div className="text-gray-500 pl-2">
                  {r.filter && <span>filter={r.filter} </span>}
                  {r.note && <span>note={r.note}</span>}
                </div>
              )}
            </div>
          ))}
        {tab === "watch" &&
          watches.map((r) => (
            <div key={r.id} className="flex gap-1 items-center border-b border-shell-border/30 py-0.5">
              <span className="text-shell-accent">{r.id}</span>
              <span className="truncate">{r.expression}</span>
              <span className="text-shell-muted truncate">{r.last}</span>
              <button type="button" className="text-blue-400 hover:underline ml-auto shrink-0" onClick={() => void runCmd(`delete ${r.id}`)}>
                del
              </button>
            </div>
          ))}
        {tab === "break" && breaks.length === 0 && <p className="text-shell-muted">{t("sessionPanel.empty")}</p>}
        {tab === "trace" && traces.length === 0 && <p className="text-shell-muted">{t("sessionPanel.empty")}</p>}
        {tab === "hook" && hooks.length === 0 && <p className="text-shell-muted">{t("sessionPanel.empty")}</p>}
        {tab === "watch" && watches.length === 0 && <p className="text-shell-muted">{t("sessionPanel.empty")}</p>}
      </div>
    </div>
  );
}
