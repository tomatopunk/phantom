import type { TFunction } from "i18next";
import type { MutableRefObject } from "react";
import type { DebugEventPayload } from "../app/types";

type Props = {
  t: TFunction;
  i18n: { language: string; changeLanguage: (lng: string) => Promise<unknown> };
  agent: string;
  setAgent: (v: string) => void;
  token: string;
  setToken: (v: string) => void;
  connected: boolean;
  capturing: boolean;
  filter: string;
  setFilter: (v: string) => void;
  setSelFi: (v: number | null) => void;
  eventsRef: MutableRefObject<DebugEventPayload[]>;
  bump: () => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onStartCap: () => void;
  onStopCap: () => void;
  exportJsonl: () => void;
};

export function AppHeader({
  t,
  i18n,
  agent,
  setAgent,
  token,
  setToken,
  connected,
  capturing,
  filter,
  setFilter,
  setSelFi,
  eventsRef,
  bump,
  onConnect,
  onDisconnect,
  onStartCap,
  onStopCap,
  exportJsonl,
}: Props) {
  return (
    <header className="flex flex-wrap items-center gap-2 px-2 py-1.5 border-b border-shell-border bg-shell-panel shrink-0">
      <span className="font-semibold text-shell-accent mr-2">{t("header.brand")}</span>
      <label className="flex items-center gap-1">
        {t("header.agent")}
        <input
          className="bg-black/40 border border-shell-border rounded px-1 py-0.5 w-44 font-mono-tight"
          value={agent}
          disabled={connected}
          onChange={(e) => setAgent(e.target.value)}
        />
      </label>
      <label className="flex items-center gap-1">
        {t("header.token")}
        <input
          type="password"
          className="bg-black/40 border border-shell-border rounded px-1 py-0.5 w-28 font-mono-tight"
          value={token}
          disabled={connected}
          onChange={(e) => setToken(e.target.value)}
        />
      </label>
      {!connected ? (
        <button
          type="button"
          className="px-2 py-0.5 rounded bg-blue-900/80 hover:bg-blue-800 border border-shell-border"
          onClick={onConnect}
        >
          {t("header.connect")}
        </button>
      ) : (
        <button
          type="button"
          className="px-2 py-0.5 rounded bg-red-900/50 hover:bg-red-800/60 border border-shell-border"
          onClick={onDisconnect}
        >
          {t("header.disconnect")}
        </button>
      )}
      <span className="text-shell-muted">|</span>
      {!capturing ? (
        <button
          type="button"
          disabled={!connected}
          className="px-2 py-0.5 rounded bg-emerald-900/60 hover:bg-emerald-800/80 border border-shell-border disabled:opacity-40"
          onClick={onStartCap}
        >
          {t("header.startCapture")}
        </button>
      ) : (
        <button
          type="button"
          className="px-2 py-0.5 rounded bg-amber-900/50 hover:bg-amber-800/60 border border-shell-border"
          onClick={onStopCap}
        >
          {t("header.stopCapture")}
        </button>
      )}
      <label className="flex items-center gap-1 flex-1 min-w-[12rem]">
        {t("header.displayFilter")}
        <input
          className="flex-1 bg-black/40 border border-shell-border rounded px-1 py-0.5 font-mono-tight"
          placeholder={t("header.filterPlaceholder")}
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            setSelFi(null);
          }}
        />
      </label>
      <button
        type="button"
        className="px-2 py-0.5 rounded border border-shell-border hover:bg-white/5"
        onClick={() => {
          eventsRef.current = [];
          setSelFi(null);
          bump();
        }}
      >
        {t("header.clearEvents")}
      </button>
      <button
        type="button"
        className="px-2 py-0.5 rounded border border-shell-border hover:bg-white/5"
        title={t("header.exportTitle")}
        onClick={exportJsonl}
      >
        {t("header.exportJsonl")}
      </button>
      <select
        className="bg-black/40 border border-shell-border rounded px-1 py-0.5 text-xs"
        value={i18n.language.startsWith("zh") ? "zh" : "en"}
        onChange={(e) => void i18n.changeLanguage(e.target.value)}
        aria-label="Language"
      >
        <option value="zh">{t("header.langZh")}</option>
        <option value="en">{t("header.langEn")}</option>
      </select>
    </header>
  );
}
