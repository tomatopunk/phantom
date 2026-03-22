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

import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import type { AppearancePreference } from "../theme/appTheme";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  open: boolean;
  onClose: () => void;
};

function resolvedUiLang(i18n: { resolvedLanguage?: string; language: string }): "en" | "zh" {
  const raw = (i18n.resolvedLanguage || i18n.language || "zh").toLowerCase();
  return raw.startsWith("zh") ? "zh" : "en";
}

export function SettingsDialog({ open, onClose }: Props) {
  const { t, i18n } = useTranslation();
  const { appearance, setAppearance } = useTheme();
  const lang = resolvedUiLang(i18n);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  const row = (value: AppearancePreference, labelKey: string) => (
    <label className="flex cursor-pointer items-center gap-2 py-1 text-app-label">
      <input
        type="radio"
        name="phantom-appearance"
        checked={appearance === value}
        onChange={() => setAppearance(value)}
        className="accent-app-accent"
      />
      {t(labelKey)}
    </label>
  );

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="flex max-h-[90vh] w-full max-w-md flex-col rounded-lg border border-app-separator bg-app-panel p-4 shadow-lg"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
      >
        <h2 id="settings-title" className="text-base font-semibold text-app-label">
          {t("settings.title")}
        </h2>
        <fieldset className="mt-3 border border-app-separator rounded-md p-3">
          <legend className="px-1 text-xs text-app-secondary">{t("settings.appearance")}</legend>
          <div className="mt-1 flex flex-col text-sm">
            {row("light", "settings.light")}
            {row("dark", "settings.dark")}
            {row("system", "settings.system")}
          </div>
        </fieldset>
        <fieldset className="mt-3 border border-app-separator rounded-md p-3">
          <legend className="px-1 text-xs text-app-secondary">{t("settings.language")}</legend>
          <div className="mt-1 flex flex-col text-sm">
            <label className="flex cursor-pointer items-center gap-2 py-1 text-app-label">
              <input
                type="radio"
                name="phantom-lang"
                checked={lang === "zh"}
                onChange={() => void i18n.changeLanguage("zh")}
                className="accent-app-accent"
              />
              {t("settings.langZh")}
            </label>
            <label className="flex cursor-pointer items-center gap-2 py-1 text-app-label">
              <input
                type="radio"
                name="phantom-lang"
                checked={lang === "en"}
                onChange={() => void i18n.changeLanguage("en")}
                className="accent-app-accent"
              />
              {t("settings.langEn")}
            </label>
          </div>
        </fieldset>
        <p className="mt-3 text-[10px] leading-snug text-app-secondary">{t("settings.hintShortcut")}</p>
        <div className="mt-4 flex justify-end">
          <button type="button" className="btn-app-primary text-xs" onClick={onClose}>
            {t("settings.close")}
          </button>
        </div>
      </div>
    </div>
  );
}
