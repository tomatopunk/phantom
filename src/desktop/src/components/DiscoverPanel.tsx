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
    <div className="flex flex-1 min-h-0 flex-col p-3">
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
        />
        {discTab === "up" && (
          <input
            className="input-app flex-1 min-w-[6rem] text-xs font-mono-tight"
            value={discBin}
            onChange={(e) => setDiscBin(e.target.value)}
            placeholder={t("discover.binaryPh")}
          />
        )}
        <button type="button" disabled={!connected} className="btn-app text-xs shrink-0" onClick={runDiscover}>
          {t("discover.list")}
        </button>
      </div>
      <ul className="flex-1 min-h-0 overflow-auto font-mono-tight text-[10px] text-app-secondary space-y-0.5">
        {discLines.map((line, i) => (
          <li
            key={`${i}-${line.slice(0, 12)}`}
            className="cursor-pointer hover:text-app-accent truncate"
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
