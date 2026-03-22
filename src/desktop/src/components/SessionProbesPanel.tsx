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

import type { ReactNode } from "react";
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
import { HookMapsPanel } from "./HookMapsPanel";

type Tab = "break" | "trace" | "hook" | "watch";

const TAB_ORDER: Tab[] = ["break", "trace", "hook", "watch"];

const TAB_SELECTED: Record<Tab, string> = {
  break:
    "border-rose-500/45 bg-rose-500/[0.11] text-app-label dark:border-rose-400/40 dark:bg-rose-500/[0.14]",
  trace:
    "border-sky-500/45 bg-sky-500/[0.11] text-app-label dark:border-sky-400/40 dark:bg-sky-500/[0.14]",
  hook:
    "border-violet-500/45 bg-violet-500/[0.11] text-app-label dark:border-violet-400/40 dark:bg-violet-500/[0.14]",
  watch:
    "border-amber-500/45 bg-amber-500/[0.11] text-app-label dark:border-amber-400/40 dark:bg-amber-500/[0.14]",
};

const TAB_HINT_BORDER: Record<Tab, string> = {
  break: "border-l-rose-500/55 dark:border-l-rose-400/50",
  trace: "border-l-sky-500/55 dark:border-l-sky-400/50",
  hook: "border-l-violet-500/55 dark:border-l-violet-400/50",
  watch: "border-l-amber-500/55 dark:border-l-amber-400/50",
};

function ProbeTabGlyph({ kind }: { kind: Tab }): ReactNode {
  const sw = 1.5;
  switch (kind) {
    case "break":
      return (
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" className="shrink-0 opacity-90" aria-hidden>
          <circle cx="12" cy="12" r="7" stroke="currentColor" strokeWidth={sw} />
          <circle cx="12" cy="12" r="2.5" fill="currentColor" />
        </svg>
      );
    case "trace":
      return (
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" className="shrink-0 opacity-90" aria-hidden>
          <path d="M4 16h3l3-6 4 6h6" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" />
          <path d="M4 8h3l2 4" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "hook":
      return (
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" className="shrink-0 opacity-90" aria-hidden>
          <path
            d="M10 6a3 3 0 1 1 4 5.2l-1.7 1.7M14 18a3 3 0 1 1-4-5.2l1.7-1.7"
            stroke="currentColor"
            strokeWidth={sw}
            strokeLinecap="round"
          />
          <path d="M8.5 8.5l-2 2M15.5 15.5l2-2" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" />
        </svg>
      );
    case "watch":
      return (
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" className="shrink-0 opacity-90" aria-hidden>
          <path
            d="M12 6c4 0 7 4.5 7 6s-3 6-7 6-7-4.5-7-6 3-6 7-6Z"
            stroke="currentColor"
            strokeWidth={sw}
            strokeLinejoin="round"
          />
          <circle cx="12" cy="12" r="2.5" stroke="currentColor" strokeWidth={sw} />
        </svg>
      );
    default:
      return null;
  }
}

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
  const [mapsHookId, setMapsHookId] = useState("");

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
    if (mapsHookId && !hooks.some((h) => h.id === mapsHookId)) {
      setMapsHookId("");
    }
  }, [hooks, mapsHookId]);

  useEffect(() => {
    if (connected) void refresh();
  }, [connected, refresh, refreshTrigger]);

  return (
    <div className="flex flex-1 min-h-0 flex-col p-3 gap-2">
      <div className="flex items-center gap-2 flex-wrap shrink-0">
        <span className="text-xs font-medium text-app-label">{t("sessionPanel.title")}</span>
        <button type="button" disabled={!connected || busy} className="btn-app text-xs" onClick={() => void refresh()}>
          {t("sessionPanel.refresh")}
        </button>
      </div>
      <p className="text-[11px] text-app-secondary leading-snug shrink-0">{t("sessionPanel.hintRepl")}</p>
      <div className="flex flex-col gap-1.5 shrink-0">
        <div className="flex flex-wrap gap-1.5" role="tablist" aria-label={t("sessionPanel.aria")}>
          {TAB_ORDER.map((k) => (
            <button
              key={k}
              type="button"
              role="tab"
              aria-selected={tab === k}
              className={`inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-app-accent ${
                tab === k ? TAB_SELECTED[k] : "border-transparent text-app-secondary hover:bg-app-hover"
              }`}
              onClick={() => setTab(k)}
            >
              <ProbeTabGlyph kind={k} />
              {t(`sessionPanel.tab.${k}`)}
            </button>
          ))}
        </div>
        <p className={`text-[10px] text-app-secondary leading-relaxed border-l-2 pl-2.5 py-0.5 ${TAB_HINT_BORDER[tab]}`}>
          {t(`sessionPanel.tabHint.${tab}`)}
        </p>
      </div>
      {err && <p className="text-[11px] text-amber-800 shrink-0 dark:text-amber-400">{err}</p>}
      <div className="flex-1 min-h-0 overflow-auto text-[10px] font-mono-tight text-app-label">
        {tab === "break" &&
          breaks.map((r) => (
            <div key={r.id} className="flex flex-wrap gap-1 items-center border-b border-app-separator/40 py-1">
              <span className="text-app-accent">{r.id}</span>
              <span className="truncate max-w-[140px]" title={r.symbol}>
                {r.symbol}
              </span>
              <span
                className={`shrink-0 rounded px-1 py-0.5 text-[10px] ${
                  r.enabled ? "bg-app-accent-muted text-app-accent" : "bg-app-field text-app-secondary"
                }`}
              >
                {r.enabled ? t("sessionPanel.bpEnabled") : t("sessionPanel.bpDisabled")}
              </span>
              {r.condition && <span className="text-gray-600 truncate max-w-[100px] dark:text-gray-500">{r.condition}</span>}
              <span className="ml-auto inline-flex flex-wrap justify-end gap-0.5 shrink-0">
                <button
                  type="button"
                  disabled={busy}
                  className="btn-app-danger px-1.5 py-0.5 text-[10px] leading-tight"
                  onClick={() => void runCmd(`delete ${r.id}`)}
                >
                  {t("sessionPanel.action.delete")}
                </button>
                <button
                  type="button"
                  disabled={busy}
                  className="btn-app px-1.5 py-0.5 text-[10px] leading-tight"
                  onClick={() => void runCmd(`disable ${r.id}`)}
                >
                  {t("sessionPanel.action.disable")}
                </button>
                <button
                  type="button"
                  disabled={busy}
                  className="btn-app-primary px-1.5 py-0.5 text-[10px] leading-tight"
                  onClick={() => void runCmd(`enable ${r.id}`)}
                >
                  {t("sessionPanel.action.enable")}
                </button>
              </span>
            </div>
          ))}
        {tab === "trace" &&
          traces.map((r) => (
            <div key={r.id} className="flex gap-1 items-center border-b border-app-separator/40 py-1">
              <span className="text-app-accent">{r.id}</span>
              <span className="truncate flex-1">{r.expressions}</span>
              <button
                type="button"
                disabled={busy}
                className="btn-app-danger shrink-0 px-1.5 py-0.5 text-[10px] leading-tight"
                onClick={() => void runCmd(`delete ${r.id}`)}
              >
                {t("sessionPanel.action.delete")}
              </button>
            </div>
          ))}
        {tab === "hook" &&
          hooks.map((r) => (
            <div
              key={r.id}
              className={`flex flex-col gap-0.5 border-b border-app-separator/40 py-1 cursor-pointer rounded-sm -mx-0.5 px-0.5 ${
                mapsHookId === r.id ? "bg-app-accent-muted/50" : ""
              }`}
              role="button"
              tabIndex={0}
              onClick={() => setMapsHookId(r.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  setMapsHookId(r.id);
                }
              }}
            >
              <div className="flex gap-1 items-center">
                <span className="text-app-accent">{r.id}</span>
                <span className="truncate flex-1" title={r.attach}>
                  {r.attach}
                </span>
                <button
                  type="button"
                  disabled={busy}
                  className="btn-app shrink-0 px-1.5 py-0.5 text-[10px] leading-tight"
                  onClick={(e) => {
                    e.stopPropagation();
                    setMapsHookId(r.id);
                  }}
                >
                  {t("sessionPanel.action.maps")}
                </button>
                <button
                  type="button"
                  disabled={busy}
                  className="btn-app-danger shrink-0 px-1.5 py-0.5 text-[10px] leading-tight"
                  onClick={(e) => {
                    e.stopPropagation();
                    void runCmd(`hook delete ${r.id}`);
                  }}
                >
                  {t("sessionPanel.action.delete")}
                </button>
              </div>
              {(r.filter || r.note) && (
                <div className="text-gray-600 pl-2 dark:text-gray-500">
                  {r.filter && <span>filter={r.filter} </span>}
                  {r.note && <span>note={r.note}</span>}
                </div>
              )}
            </div>
          ))}
        {tab === "watch" &&
          watches.map((r) => (
            <div key={r.id} className="flex gap-1 items-center border-b border-app-separator/40 py-1">
              <span className="text-app-accent">{r.id}</span>
              <span className="truncate">{r.expression}</span>
              <span className="text-app-secondary truncate">{r.last}</span>
              <button
                type="button"
                disabled={busy}
                className="btn-app-danger ml-auto shrink-0 px-1.5 py-0.5 text-[10px] leading-tight"
                onClick={() => void runCmd(`delete ${r.id}`)}
              >
                {t("sessionPanel.action.delete")}
              </button>
            </div>
          ))}
        {tab === "break" && breaks.length === 0 && <p className="text-app-secondary">{t("sessionPanel.empty")}</p>}
        {tab === "trace" && traces.length === 0 && <p className="text-app-secondary">{t("sessionPanel.empty")}</p>}
        {tab === "hook" && hooks.length === 0 && <p className="text-app-secondary">{t("sessionPanel.empty")}</p>}
        {tab === "watch" && watches.length === 0 && <p className="text-app-secondary">{t("sessionPanel.empty")}</p>}
        {tab === "hook" ? (
          <HookMapsPanel t={t} connected={connected} hookId={mapsHookId} onHookIdChange={setMapsHookId} />
        ) : null}
      </div>
    </div>
  );
}
