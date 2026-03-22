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

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  applyDarkClassToDocument,
  persistAppearance,
  readStoredAppearance,
  readSystemPrefersDark,
  type AppearancePreference,
  type ResolvedColorScheme,
} from "./appTheme";

type ThemeContextValue = {
  appearance: AppearancePreference;
  setAppearance: (p: AppearancePreference) => void;
  resolved: ResolvedColorScheme;
  resolvedDark: boolean;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [appearance, setAppearanceState] = useState<AppearancePreference>(() => readStoredAppearance());
  const [systemDark, setSystemDark] = useState(() => readSystemPrefersDark());

  /** Browser / immediate sync when switching back to "system". */
  useEffect(() => {
    if (appearance === "system") {
      setSystemDark(readSystemPrefersDark());
    }
  }, [appearance]);

  /** Web: prefers-color-scheme. */
  useEffect(() => {
    const m = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setSystemDark(m.matches);
    m.addEventListener("change", onChange);
    return () => m.removeEventListener("change", onChange);
  }, []);

  /**
   * Tauri: locking the window with setTheme("light"|"dark") makes the WebView's
   * prefers-color-scheme follow the window, not macOS — so "follow system" never updates.
   * Use setTheme(null) for system mode and drive UI from onThemeChanged + theme().
   */
  useEffect(() => {
    let cancelled = false;
    let unlisten: (() => void) | undefined;

    void (async () => {
      try {
        const { getCurrentWindow } = await import("@tauri-apps/api/window");
        const w = getCurrentWindow();

        if (appearance === "system") {
          await w.setTheme(null);
          const t = await w.theme();
          if (cancelled) return;
          if (t === "dark") setSystemDark(true);
          else if (t === "light") setSystemDark(false);
          else setSystemDark(readSystemPrefersDark());

          unlisten = await w.onThemeChanged(({ payload }) => {
            setSystemDark(payload === "dark");
          });
        } else {
          await w.setTheme(appearance);
        }
      } catch {
        /* Vite / non-Tauri */
      }
    })();

    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, [appearance]);

  /** WebView lag: refresh when app regains focus (system mode only). */
  useEffect(() => {
    if (appearance !== "system") return;
    const refresh = () => setSystemDark(readSystemPrefersDark());
    const onVis = () => {
      if (document.visibilityState === "visible") refresh();
    };
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", onVis);
    return () => {
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", onVis);
    };
  }, [appearance]);

  const resolved: ResolvedColorScheme = useMemo(
    () => (appearance === "system" ? (systemDark ? "dark" : "light") : appearance),
    [appearance, systemDark],
  );

  const resolvedDark = resolved === "dark";

  useEffect(() => {
    applyDarkClassToDocument(resolvedDark);
  }, [resolvedDark]);

  const setAppearance = useCallback((p: AppearancePreference) => {
    persistAppearance(p);
    setAppearanceState(p);
  }, []);

  const value = useMemo(
    () => ({
      appearance,
      setAppearance,
      resolved,
      resolvedDark,
    }),
    [appearance, setAppearance, resolved, resolvedDark],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider");
  return ctx;
}

/** For Monaco and any consumer that only needs dark vs light. */
export function useResolvedDark(): boolean {
  return useTheme().resolvedDark;
}
