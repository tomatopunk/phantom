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
import { useState } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

export type ToolSection = "overview" | "discover" | "session" | "console" | "hook";

type Props = {
  t: TFunction;
  overview: React.ReactNode;
  discover: React.ReactNode;
  session: React.ReactNode;
  repl: React.ReactNode;
  hook: React.ReactNode;
  events: React.ReactNode;
};

const NAV: { id: ToolSection; labelKey: string }[] = [
  { id: "overview", labelKey: "sidebar.nav.overview" },
  { id: "discover", labelKey: "sidebar.nav.discover" },
  { id: "session", labelKey: "sidebar.nav.session" },
  { id: "console", labelKey: "sidebar.nav.console" },
  { id: "hook", labelKey: "sidebar.nav.hook" },
];

export function AppShell({ t, overview, discover, session, repl, hook, events }: Props) {
  const [tool, setTool] = useState<ToolSection>("overview");

  const panel =
    tool === "overview"
      ? overview
      : tool === "discover"
        ? discover
        : tool === "session"
          ? session
          : tool === "console"
            ? repl
            : hook;

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <nav
        className="w-[52px] shrink-0 flex flex-col gap-0.5 py-2 px-1 border-r border-app-separator bg-app-sidebar"
        aria-label={t("sidebar.aria")}
      >
        {NAV.map((item) => (
          <button
            key={item.id}
            type="button"
            title={t(item.labelKey)}
            aria-current={tool === item.id ? "page" : undefined}
            className={`flex flex-col items-center justify-center gap-0.5 rounded-md py-2 px-1 text-[10px] leading-tight transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-app-accent ${
              tool === item.id
                ? "bg-app-accent-muted text-app-label"
                : "text-app-secondary hover:bg-app-hover"
            }`}
            onClick={() => setTool(item.id)}
          >
            <SidebarGlyph kind={item.id} active={tool === item.id} />
            <span className="text-center break-all">{t(`${item.labelKey}Short`)}</span>
          </button>
        ))}
      </nav>

      <PanelGroup direction="horizontal" className="flex-1 min-w-0">
        <Panel defaultSize={32} minSize={22} className="flex min-h-0 min-w-0 flex-col border-r border-app-separator bg-app-panel">
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{panel}</div>
        </Panel>

        <PanelResizeHandle className="w-1 bg-app-separator hover:bg-app-accent/35 transition-colors" />

        {events}
      </PanelGroup>
    </div>
  );
}

function SidebarGlyph({ kind, active }: { kind: ToolSection; active: boolean }) {
  const stroke = active ? "var(--app-accent)" : "var(--app-secondary)";
  const sw = 1.6;
  switch (kind) {
    case "overview":
      return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
          <path d="M4 19V5M4 19h16M4 19l4-6 4 3 4-8 4 11" stroke={stroke} strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "discover":
      return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
          <circle cx="10.5" cy="10.5" r="5.5" stroke={stroke} strokeWidth={sw} />
          <path d="M15 15L20 20" stroke={stroke} strokeWidth={sw} strokeLinecap="round" />
        </svg>
      );
    case "session":
      return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
          <rect x="4" y="5" width="16" height="12" rx="2" stroke={stroke} strokeWidth={sw} />
          <path d="M8 9h8M8 13h5" stroke={stroke} strokeWidth={sw} strokeLinecap="round" />
        </svg>
      );
    case "console":
      return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
          <path d="M5 7l4 5-4 5M11 17h8" stroke={stroke} strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "hook":
      return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
          <path d="M6 8h12v10a2 2 0 01-2 2H8a2 2 0 01-2-2V8z" stroke={stroke} strokeWidth={sw} />
          <path d="M8 8V6a2 2 0 012-2h4a2 2 0 012 2v2" stroke={stroke} strokeWidth={sw} />
        </svg>
      );
    default:
      return null;
  }
}
