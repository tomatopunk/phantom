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

export const AGENT_HISTORY_DATALIST_ID = "phantom-agent-history";

type Props = {
  t: TFunction;
  agent: string;
  setAgent: (v: string) => void;
  agentHistory: string[];
  token: string;
  setToken: (v: string) => void;
  connected: boolean;
  capturing: boolean;
  filter: string;
  setFilter: (v: string) => void;
  setSelFi: (v: number | null) => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onStartCap: () => void;
  onStopCap: () => void;
};

export function AppHeader({
  t,
  agent,
  setAgent,
  token,
  setToken,
  connected,
  capturing,
  filter,
  setFilter,
  setSelFi,
  onConnect,
  onDisconnect,
  onStartCap,
  onStopCap,
  agentHistory,
}: Props) {
  return (
    <header
      className="flex shrink-0 flex-wrap items-center gap-2 border-b border-app-separator bg-app-panel px-3 py-2 text-sm text-app-label"
      role="toolbar"
      aria-label={t("header.toolbarAria")}
    >
      <span className="mr-1 font-semibold tracking-tight text-app-accent">{t("header.brand")}</span>
      <datalist id={AGENT_HISTORY_DATALIST_ID}>
        {agentHistory.map((h) => (
          <option key={h} value={h} />
        ))}
      </datalist>
      <label className="flex min-w-[10rem] flex-1 items-center gap-2">
        <span className="shrink-0 text-app-secondary">{t("header.agent")}</span>
        <input
          className="input-app min-w-0 flex-1 font-mono-tight text-xs"
          value={agent}
          disabled={connected}
          onChange={(e) => setAgent(e.target.value)}
          list={AGENT_HISTORY_DATALIST_ID}
          title={t("header.agentHistoryHint")}
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          autoComplete="on"
        />
      </label>
      <label className="flex items-center gap-2">
        <span className="shrink-0 text-app-secondary">{t("header.token")}</span>
        <input
          type="password"
          className="input-app w-32 font-mono-tight text-xs"
          value={token}
          disabled={connected}
          onChange={(e) => setToken(e.target.value)}
          autoComplete="off"
        />
      </label>
      {!connected ? (
        <button type="button" className="btn-app-primary text-xs" onClick={onConnect}>
          {t("header.connect")}
        </button>
      ) : (
        <button type="button" className="btn-app-danger text-xs" onClick={onDisconnect}>
          {t("header.disconnect")}
        </button>
      )}
      <span className="text-app-secondary" aria-hidden>
        |
      </span>
      {!capturing ? (
        <button type="button" disabled={!connected} className="btn-app-primary text-xs" onClick={onStartCap}>
          {t("header.startCapture")}
        </button>
      ) : (
        <button type="button" className="btn-app text-xs text-amber-800 dark:text-amber-200/90" onClick={onStopCap}>
          {t("header.stopCapture")}
        </button>
      )}
      <label className="flex min-w-[12rem] flex-[2] items-center gap-2">
        <span className="shrink-0 text-app-secondary">{t("header.displayFilter")}</span>
        <input
          className="input-app min-w-0 flex-1 font-mono-tight text-xs"
          placeholder={t("header.filterPlaceholder")}
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            setSelFi(null);
          }}
        />
      </label>
    </header>
  );
}
