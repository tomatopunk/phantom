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
  cmd: string;
  setCmd: (s: string) => void;
  cmdOut: string;
  runCmd: () => void;
  connected: boolean;
};

export function CmdReplBlock({ t, cmd, setCmd, cmdOut, runCmd, connected }: Props) {
  return (
    <div className="flex flex-1 min-h-0 flex-col gap-2 p-3">
      <div className="text-xs font-medium text-app-label shrink-0">{t("cmd.title")}</div>
      <div className="flex gap-2 shrink-0">
        <input
          className="input-app flex-1 font-mono-tight text-xs"
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && runCmd()}
          aria-label={t("cmd.inputAria")}
        />
        <button type="button" disabled={!connected} className="btn-app text-xs shrink-0" onClick={runCmd}>
          {t("cmd.run")}
        </button>
      </div>
      <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-field p-2 text-[10px] text-app-label whitespace-pre-wrap font-mono-tight">
        {cmdOut}
      </pre>
    </div>
  );
}
