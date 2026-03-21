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
  parentRef: Ref<HTMLDivElement>;
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  firstTs: number;
  selFi: number | null;
  setSelFi: (i: number | null) => void;
  relTimeNs: (first: number, cur: number) => string;
  selected: DebugEventPayload | null;
};

export function EventsStreamPanel({
  t,
  filtered,
  eventCount,
  parentRef,
  rowVirtualizer,
  firstTs,
  selFi,
  setSelFi,
  relTimeNs,
  selected,
}: Props) {
  return (
    <Panel minSize={40} className="min-w-0 flex flex-col">
      <PanelGroup direction="vertical" className="flex-1 min-h-0">
        <Panel defaultSize={52} minSize={25} className="min-h-0 flex flex-col border-b border-shell-border">
          <div className="px-2 py-0.5 text-xs text-shell-muted border-b border-shell-border shrink-0">
            {t("events.panelTitle", { filtered: filtered.length, total: eventCount })}
          </div>
          <div ref={parentRef} className="flex-1 overflow-auto font-mono-tight">
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
                    className={`absolute top-0 left-0 w-full flex cursor-pointer border-b border-shell-border/30 text-[11px] ${
                      sel ? "bg-shell-accent/15" : "hover:bg-white/5"
                    }`}
                    style={{ height: `${vi.size}px`, transform: `translateY(${vi.start}px)` }}
                    onClick={() => setSelFi(vi.index)}
                  >
                    <span className="w-8 shrink-0 text-shell-muted px-1">{vi.index}</span>
                    <span className="w-24 shrink-0 truncate">{relTimeNs(firstTs, ev.timestamp_ns)}</span>
                    <span className="w-28 shrink-0 truncate text-amber-200/90">{ev.event_type_name}</span>
                    <span className="w-12 shrink-0">{ev.pid}</span>
                    <span className="w-12 shrink-0">{ev.tgid}</span>
                    <span className="w-8 shrink-0">{ev.cpu}</span>
                    <span className="w-16 shrink-0 truncate">{ev.probe_id}</span>
                    <span className="flex-1 truncate text-gray-500">{ev.payload_utf8.replace(/\s+/g, " ")}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </Panel>

        <PanelResizeHandle className="h-1 bg-shell-border hover:bg-shell-accent/40" />

        <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col border-b border-shell-border">
          <div className="px-2 py-0.5 text-xs text-shell-muted shrink-0">{t("detail.jsonTitle")}</div>
          <pre className="flex-1 overflow-auto p-2 text-[11px] font-mono-tight bg-black/20">
            {selected ? JSON.stringify(selected, null, 2) : t("detail.pickRow")}
          </pre>
        </Panel>

        <PanelResizeHandle className="h-1 bg-shell-border hover:bg-shell-accent/40" />

        <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col">
          <div className="px-2 py-0.5 text-xs text-shell-muted shrink-0">
            {t("hex.title")}
            {selected?.payload_truncated ? t("hex.truncated") : ""}
          </div>
          <pre className="flex-1 overflow-auto p-2 text-[11px] font-mono-tight bg-black/20 whitespace-pre">
            {selected ? hexLines(selected.payload_hex).join("\n") : t("common.dash")}
          </pre>
        </Panel>
      </PanelGroup>
    </Panel>
  );
}
