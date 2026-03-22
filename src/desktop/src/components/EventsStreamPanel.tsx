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

import type { Virtualizer } from "@tanstack/react-virtual";
import type { TFunction } from "i18next";
import type { Ref } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { hexLines } from "../app/eventUtils";
import type { DebugEventPayload } from "../app/types";

type Props = {
  t: TFunction;
  filtered: DebugEventPayload[];
  eventCount: number;
  /** Desktop-side retained rows cap (matches trim in App). */
  maxBuffer: number;
  onClearBuffer: () => void;
  parentRef: Ref<HTMLDivElement>;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  firstTs: number;
  selFi: number | null;
  setSelFi: (i: number | null) => void;
  relTimeNs: (first: number, cur: number) => string;
  selected: DebugEventPayload | null;
  connected: boolean;
  capturing: boolean;
};

export function EventsStreamPanel({
  t,
  filtered,
  eventCount,
  maxBuffer,
  onClearBuffer,
  parentRef,
  rowVirtualizer,
  firstTs,
  selFi,
  setSelFi,
  relTimeNs,
  selected,
  connected,
  capturing,
}: Props) {
  const emptyHint = !connected
    ? t("events.emptyDisconnected")
    : !capturing
      ? t("events.emptyNeedCapture")
      : t("events.emptyNoProbesYet");

  return (
    <Panel minSize={40} className="min-w-0 flex min-h-0 flex-1 flex-col bg-app-bg">
      <PanelGroup direction="vertical" className="flex-1 min-h-0">
        <Panel defaultSize={52} minSize={25} className="min-h-0 flex flex-col border-b border-app-separator">
          <div className="shrink-0 border-b border-app-separator px-3 py-2 flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1 space-y-0.5">
              <div className="text-xs text-app-secondary">
                {t("events.panelTitle", { filtered: filtered.length, total: eventCount })}
              </div>
              <p className="text-[10px] text-app-secondary/85 leading-snug m-0">
                {t("events.bufferCaption", { max: maxBuffer })}
              </p>
            </div>
            <button
              type="button"
              className="shrink-0 rounded-md border border-app-separator bg-app-field px-2 py-1 text-[10px] font-medium text-app-label hover:bg-app-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-app-accent disabled:opacity-40 disabled:pointer-events-none"
              disabled={eventCount === 0}
              title={t("events.clearBufferTitle", { max: maxBuffer })}
              aria-label={t("events.clearBufferAria")}
              onClick={onClearBuffer}
            >
              {t("events.clearBuffer")}
            </button>
          </div>
          <div ref={parentRef} className="flex-1 overflow-auto font-mono-tight text-app-label">
            {filtered.length === 0 ? (
              <div className="p-4 text-[11px] text-app-secondary leading-relaxed whitespace-pre-wrap">{emptyHint}</div>
            ) : (
              <>
                <div
                  className="sticky top-0 z-10 flex shrink-0 border-b border-app-separator bg-app-bg/95 py-1 text-[10px] font-medium text-app-secondary backdrop-blur-sm"
                  role="row"
                >
                  <span className="w-8 shrink-0 px-1">{t("events.colIndex")}</span>
                  <span className="w-24 shrink-0 truncate">{t("events.colDelta")}</span>
                  <span className="w-28 shrink-0 truncate">{t("events.colType")}</span>
                  <span className="w-12 shrink-0">{t("events.colPid")}</span>
                  <span className="w-12 shrink-0">{t("events.colTgid")}</span>
                  <span className="w-8 shrink-0">{t("events.colCpu")}</span>
                  <span className="w-16 shrink-0 truncate">{t("events.colProbe")}</span>
                  <span className="min-w-0 flex-1 truncate pr-2">{t("events.colPayload")}</span>
                </div>
                <div
                  style={{
                    height: `${rowVirtualizer.getTotalSize()}px`,
                    width: "100%",
                    position: "relative",
                  }}
                >
                  {rowVirtualizer.getVirtualItems().map((vi) => {
                  const ev = filtered[vi.index];
                  const sel = selFi === vi.index;
                  return (
                    <div
                      key={vi.key}
                      className={`absolute top-0 left-0 w-full flex cursor-pointer border-b border-app-separator/40 text-[11px] ${
                        sel ? "bg-app-accent-muted" : "hover:bg-app-hover"
                      }`}
                      style={{ height: `${vi.size}px`, transform: `translateY(${vi.start}px)` }}
                      onClick={() => setSelFi(vi.index)}
                    >
                      <span className="w-8 shrink-0 px-1 text-app-secondary">{vi.index}</span>
                      <span className="w-24 shrink-0 truncate">{relTimeNs(firstTs, ev.timestamp_ns)}</span>
                      <span className="w-28 shrink-0 truncate text-amber-800 dark:text-amber-200/90">{ev.event_type_name}</span>
                      <span className="w-12 shrink-0">{ev.pid}</span>
                      <span className="w-12 shrink-0">{ev.tgid}</span>
                      <span className="w-8 shrink-0">{ev.cpu}</span>
                      <span className="w-16 shrink-0 truncate">{ev.probe_id}</span>
                      <span className="min-w-0 flex-1 truncate pr-2 text-app-secondary">{ev.payload_utf8.replace(/\s+/g, " ")}</span>
                    </div>
                  );
                  })}
                </div>
              </>
            )}
          </div>
        </Panel>

        <PanelResizeHandle className="h-1 bg-app-separator hover:bg-app-accent/35" />

        <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col border-b border-app-separator">
          <div className="shrink-0 px-3 py-2 text-xs text-app-secondary">{t("detail.jsonTitle")}</div>
          <pre className="flex-1 overflow-auto bg-app-field p-2 text-[11px] font-mono-tight text-app-label">
            {selected ? JSON.stringify(selected, null, 2) : t("detail.pickRow")}
          </pre>
        </Panel>

        <PanelResizeHandle className="h-1 bg-app-separator hover:bg-app-accent/35" />

        <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col">
          <div className="shrink-0 px-3 py-2 text-xs text-app-secondary">
            {t("hex.title")}
            {selected?.payload_truncated ? t("hex.truncated") : ""}
          </div>
          <pre className="flex-1 overflow-auto whitespace-pre bg-app-field p-2 text-[11px] font-mono-tight text-app-label">
            {selected ? hexLines(selected.payload_hex).join("\n") : t("common.dash")}
          </pre>
        </Panel>
      </PanelGroup>
    </Panel>
  );
}
