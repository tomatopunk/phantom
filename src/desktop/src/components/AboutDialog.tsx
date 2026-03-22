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

import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { APACHE_LICENSE_URL, GITHUB_REPO_URL } from "../app/constants";

const FALLBACK_VERSION = "0.1.0";

type Props = {
  open: boolean;
  onClose: () => void;
};

export function AboutDialog({ open, onClose }: Props) {
  const { t } = useTranslation();
  const [version, setVersion] = useState(FALLBACK_VERSION);
  const year = new Date().getFullYear();

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void (async () => {
      try {
        const { getVersion } = await import("@tauri-apps/api/app");
        const v = await getVersion();
        if (!cancelled && v) setVersion(v);
      } catch {
        if (!cancelled) setVersion(FALLBACK_VERSION);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="flex max-h-[90vh] w-full max-w-lg flex-col overflow-y-auto rounded-lg border border-app-separator bg-app-panel p-5 shadow-lg"
        role="dialog"
        aria-modal="true"
        aria-labelledby="about-title"
      >
        <h2 id="about-title" className="text-lg font-semibold tracking-tight text-app-label">
          {t("about.title")}
        </h2>
        <p className="mt-1 text-sm font-medium text-app-accent">{t("about.subtitle")}</p>
        <p className="mt-3 text-xs leading-relaxed text-app-secondary">{t("about.body")}</p>

        <div className="mt-4 rounded-md border border-app-separator bg-app-field px-3 py-2">
          <div className="text-[10px] uppercase tracking-wide text-app-secondary">{t("about.version")}</div>
          <div className="mt-0.5 font-mono-tight text-sm text-app-label">{version}</div>
        </div>

        <h3 className="mt-4 text-xs font-semibold uppercase tracking-wide text-app-label">
          {t("about.licenseHeading")}
        </h3>
        <p className="mt-1 text-xs leading-relaxed text-app-secondary">{t("about.licenseDetail")}</p>
        <p className="mt-2 text-sm">
          <a
            href={APACHE_LICENSE_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="text-app-accent underline hover:opacity-90"
          >
            {t("about.licenseName")}
          </a>
        </p>

        <h3 className="mt-4 text-xs font-semibold uppercase tracking-wide text-app-label">
          {t("about.repositoryHeading")}
        </h3>
        <p className="mt-1 text-xs leading-relaxed text-app-secondary">{t("about.repositoryHint")}</p>
        <p className="mt-2 break-all text-sm">
          <a
            href={GITHUB_REPO_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="text-app-accent underline hover:opacity-90"
          >
            {GITHUB_REPO_URL}
          </a>
        </p>

        <p className="mt-4 border-t border-app-separator pt-3 text-[11px] text-app-secondary">
          {t("about.copyright", { year })}
        </p>

        <div className="mt-4 flex justify-end">
          <button type="button" className="btn-app-primary text-xs" onClick={onClose}>
            {t("about.close")}
          </button>
        </div>
      </div>
    </div>
  );
}
