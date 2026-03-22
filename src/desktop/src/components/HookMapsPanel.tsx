/**
 * Copyright 2026 The Phantom Authors
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import type { TFunction } from "i18next";
import { useCallback, useEffect, useState } from "react";
import * as api from "../api";
import { technicalInputProps } from "../app/technicalInputProps";

type Props = {
  t: TFunction;
  connected: boolean;
  /** From `info hook` row selection; also editable. */
  hookId: string;
  onHookIdChange: (id: string) => void;
};

export function HookMapsPanel({ t, connected, hookId, onHookIdChange }: Props) {
  const [maps, setMaps] = useState<api.HookMapDescriptor[]>([]);
  const [mapName, setMapName] = useState("");
  const [maxEntries, setMaxEntries] = useState(16);
  const [entries, setEntries] = useState<api.MapEntryHex[]>([]);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const listMaps = useCallback(async () => {
    const id = hookId.trim();
    if (!connected || !id) return;
    setBusy(true);
    setErr("");
    setMaps([]);
    setEntries([]);
    try {
      const r = await api.listHookMaps(id);
      if (!r.ok) {
        setErr(r.error_message || t("hookMaps.listFailed"));
        return;
      }
      setMaps(r.maps);
      setMapName((prev) => {
        if (prev && r.maps.some((m) => m.name === prev)) return prev;
        return r.maps[0]?.name ?? "";
      });
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }, [connected, hookId, t]);

  const readMap = useCallback(async () => {
    const id = hookId.trim();
    const name = mapName.trim();
    if (!connected || !id || !name) return;
    setBusy(true);
    setErr("");
    setEntries([]);
    try {
      const r = await api.readHookMap(id, name, Math.max(1, Math.min(4096, maxEntries)));
      if (!r.ok) {
        setErr(r.error_message || t("hookMaps.readFailed"));
        return;
      }
      setEntries(r.entries);
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }, [connected, hookId, mapName, maxEntries, t]);

  useEffect(() => {
    setMaps([]);
    setMapName("");
    setEntries([]);
    setErr("");
  }, [hookId]);

  return (
    <div className="mt-2 pt-2 border-t border-app-separator/60 space-y-2 shrink-0">
      <div className="text-[10px] font-medium text-app-label">{t("hookMaps.title")}</div>
      <p className="text-[10px] text-app-secondary leading-snug m-0">{t("hookMaps.hint")}</p>
      <div className="flex flex-wrap gap-1.5 items-end">
        <label className="flex flex-col gap-0.5 min-w-[120px] flex-1">
          <span className="text-[10px] text-app-secondary">{t("hookMaps.hookId")}</span>
          <input
            className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-[10px] text-app-label"
            value={hookId}
            onChange={(e) => onHookIdChange(e.target.value)}
            placeholder={t("hookMaps.hookIdPh")}
            {...technicalInputProps}
          />
        </label>
        <button type="button" disabled={!connected || busy || !hookId.trim()} className="btn-app text-[10px]" onClick={() => void listMaps()}>
          {t("hookMaps.listMaps")}
        </button>
      </div>
      {maps.length > 0 ? (
        <div className="flex flex-wrap gap-1.5 items-end">
          <label className="flex flex-col gap-0.5 min-w-[140px] flex-1">
            <span className="text-[10px] text-app-secondary">{t("hookMaps.mapName")}</span>
            <select
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-[10px] text-app-label"
              value={mapName}
              onChange={(e) => setMapName(e.target.value)}
            >
              {maps.map((m) => (
                <option key={m.name} value={m.name}>
                  {m.name} ({m.map_type})
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-0.5 w-[88px]">
            <span className="text-[10px] text-app-secondary">{t("hookMaps.maxEntries")}</span>
            <input
              type="number"
              min={1}
              max={4096}
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-[10px] text-app-label"
              value={maxEntries}
              onChange={(e) => setMaxEntries(Number(e.target.value) || 16)}
            />
          </label>
          <button type="button" disabled={!connected || busy || !mapName} className="btn-app-primary text-[10px]" onClick={() => void readMap()}>
            {t("hookMaps.readMap")}
          </button>
        </div>
      ) : null}
      {err ? <p className="text-[10px] text-amber-800 dark:text-amber-400 m-0">{err}</p> : null}
      {busy ? <p className="text-[10px] text-app-secondary m-0">{t("common.ellipsis")}</p> : null}
      {entries.length > 0 ? (
        <div className="max-h-[160px] overflow-auto rounded border border-app-separator/80 bg-app-field/30 p-1.5 font-mono-tight text-[9px] text-app-label space-y-1">
          {entries.map((e, i) => (
            <div key={i} className="border-b border-app-separator/40 pb-1 last:border-0">
              <div className="text-app-secondary">key</div>
              <div className="break-all whitespace-pre-wrap">{e.key_hex}</div>
              <div className="text-app-secondary pt-0.5">value</div>
              <div className="break-all whitespace-pre-wrap">{e.value_hex}</div>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
