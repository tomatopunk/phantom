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

import type { TFunction } from "i18next";
import { useEffect, useState } from "react";
import {
  discoveryCommandForProbe,
  type DiscoverProbeKind,
  type ProbeRunDraft,
} from "../app/discoverCommands";
import { technicalInputProps } from "../app/technicalInputProps";

type Tab = "tp" | "kp" | "up";

const QUICK_KINDS: DiscoverProbeKind[] = ["break", "trace", "hook", "watch"];

const QUICK_BTN: Record<DiscoverProbeKind, string> = {
  break:
    "rounded border px-1.5 py-0.5 text-[10px] font-medium transition-colors border-rose-500/40 bg-rose-500/[0.06] hover:bg-rose-500/15 disabled:opacity-35 dark:border-rose-400/35",
  trace:
    "rounded border px-1.5 py-0.5 text-[10px] font-medium transition-colors border-sky-500/40 bg-sky-500/[0.06] hover:bg-sky-500/15 disabled:opacity-35 dark:border-sky-400/35",
  hook:
    "rounded border px-1.5 py-0.5 text-[10px] font-medium transition-colors border-violet-500/40 bg-violet-500/[0.06] hover:bg-violet-500/15 disabled:opacity-35 dark:border-violet-400/35",
  watch:
    "rounded border px-1.5 py-0.5 text-[10px] font-medium transition-colors border-amber-500/40 bg-amber-500/[0.06] hover:bg-amber-500/15 disabled:opacity-35 dark:border-amber-400/35",
};

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
  setCmd: (s: string) => void;
  openConsole: () => void;
  onOpenProbeRun: (draft: ProbeRunDraft) => void;
  runCommandLine: (line: string) => Promise<void>;
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
  setCmd,
  openConsole,
  onOpenProbeRun,
  runCommandLine,
}: Props) {
  const [selectedLine, setSelectedLine] = useState<string | null>(null);
  const [flash, setFlash] = useState("");

  useEffect(() => {
    setSelectedLine(null);
  }, [discTab, discLines]);

  const showFlash = (msg: string) => {
    setFlash(msg);
    window.setTimeout(() => setFlash(""), 1600);
  };

  const onQuick = async (line: string, kind: DiscoverProbeKind, e: { shiftKey: boolean }) => {
    const cmd = discoveryCommandForProbe(discTab, line, discBin, kind);
    if (!cmd) return;
    void navigator.clipboard.writeText(cmd);
    if (e.shiftKey) {
      setCmd(cmd);
      openConsole();
      await runCommandLine(cmd);
      showFlash(t("discover.quick.ran"));
      return;
    }
    onOpenProbeRun({ tab: discTab, line, binaryPath: discBin, kind });
    showFlash(t("discover.quick.openComposer"));
  };

  const toggleSelect = (line: string) => {
    setSelectedLine((s) => (s === line ? null : line));
  };

  return (
    <div className="flex flex-1 min-h-0 flex-col p-3">
      <p className="text-[11px] text-app-secondary mb-2 shrink-0 leading-snug">
        {t("discover.hint")}
        {flash ? <span className="ml-1 text-app-accent">{flash}</span> : null}
      </p>
      <div className="flex flex-wrap gap-1 mb-2" role="tablist" aria-label={t("discover.aria")}>
        {(["tp", "kp", "up"] as const).map((k) => (
          <button
            key={k}
            type="button"
            role="tab"
            aria-selected={discTab === k}
            className={`rounded-md px-2 py-1 text-xs border transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-app-accent ${
              discTab === k ? "border-app-separator bg-app-field text-app-label" : "border-transparent text-app-secondary hover:bg-app-hover"
            }`}
            onClick={() => setDiscTab(k)}
          >
            {discTabLabel(t, k)}
          </button>
        ))}
      </div>
      <div className="flex flex-wrap gap-2 mb-2">
        <input
          className="input-app flex-1 min-w-[6rem] text-xs font-mono-tight"
          value={discPrefix}
          onChange={(e) => setDiscPrefix(e.target.value)}
          placeholder={t("discover.prefixPh")}
          {...technicalInputProps}
        />
        {discTab === "up" && (
          <input
            className="input-app flex-1 min-w-[6rem] text-xs font-mono-tight"
            value={discBin}
            onChange={(e) => setDiscBin(e.target.value)}
            placeholder={t("discover.binaryPh")}
            {...technicalInputProps}
          />
        )}
        <button type="button" disabled={!connected} className="btn-app text-xs shrink-0" onClick={runDiscover}>
          {t("discover.list")}
        </button>
      </div>
      <ul className="flex-1 min-h-0 overflow-auto space-y-0.5 font-mono-tight text-[10px] text-app-secondary">
        {discLines.map((line, i) => {
          const open = selectedLine === line;
          return (
            <li
              key={`${i}-${line.slice(0, 12)}`}
              className={`rounded-md border border-transparent ${open ? "border-app-separator bg-app-field" : ""}`}
            >
              <button
                type="button"
                className="w-full truncate px-1.5 py-1 text-left hover:text-app-accent"
                title={t("discover.rowExpandTitle")}
                onClick={() => toggleSelect(line)}
              >
                {line}
              </button>
              {open ? (
                <div
                  className="flex flex-wrap gap-1 border-t border-app-separator/50 px-1.5 py-1.5"
                  role="group"
                  aria-label={t("discover.quick.ariaGroup")}
                >
                  {QUICK_KINDS.map((kind) => {
                    const cmd = discoveryCommandForProbe(discTab, line, discBin, kind);
                    const disabled = !connected || !cmd;
                    let title = cmd ?? "";
                    if (disabled) {
                      if (kind === "break" && (discTab === "tp" || discTab === "up")) title = t("discover.quick.breakUnavailable");
                      else if (!cmd) title = t("discover.quick.invalidRow");
                    }
                    return (
                      <button
                        key={kind}
                        type="button"
                        disabled={disabled}
                        title={title}
                        className={QUICK_BTN[kind]}
                        onClick={(e) => void onQuick(line, kind, e)}
                      >
                        {t(`discover.quick.${kind}`)}
                      </button>
                    );
                  })}
                  <span className="w-full text-[9px] text-app-secondary/90 leading-snug pt-0.5">{t("discover.quick.shiftHint")}</span>
                </div>
              ) : null}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
