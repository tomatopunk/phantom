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
  message: string;
  onDismiss: () => void;
};

export function InlineErrorBanner({ t, message, onDismiss }: Props) {
  if (!message) return null;
  return (
    <div
      className="flex shrink-0 items-center gap-2 border-b border-amber-700/50 bg-amber-950/40 px-3 py-2 text-sm text-amber-100/95"
      role="alert"
    >
      <span className="min-w-0 flex-1 break-words">{message}</span>
      <button type="button" className="btn-app shrink-0 text-xs" onClick={onDismiss}>
        {t("errorBanner.dismiss")}
      </button>
    </div>
  );
}
