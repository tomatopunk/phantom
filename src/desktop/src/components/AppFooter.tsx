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

type Props = {
  t: TFunction;
  sessionId: string | null;
  connected: boolean;
  capturing: boolean;
  metricsAt: string | null;
};

export function AppFooter({ t, sessionId, connected, capturing, metricsAt }: Props) {
  return (
    <footer className="flex shrink-0 flex-wrap gap-x-4 gap-y-1 border-t border-app-separator bg-app-panel px-3 py-2 text-[11px] text-app-secondary">
      <span>
        {t("footer.session")} {sessionId ?? t("common.dash")}
      </span>
      <span>
        {t("footer.connected")} {connected ? t("common.yes") : t("common.no")}
      </span>
      <span>
        {t("footer.capture")} {capturing ? t("footer.capturing") : t("footer.stopped")}
      </span>
      <span>
        {t("footer.metricsRefresh")} {metricsAt ?? t("common.dash")}
      </span>
      <span className="ml-auto">{t("footer.shortcutsHint")}</span>
    </footer>
  );
}
