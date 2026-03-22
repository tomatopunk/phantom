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

import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

export function GlossaryTip({
  term,
  labelKey,
  label,
  children,
}: {
  term: string;
  /** i18n key under translation root (e.g. metrics.load1m) */
  labelKey?: string;
  /** Fallback if labelKey omitted */
  label?: string;
  children?: ReactNode;
}) {
  const { t } = useTranslation();
  const title = t(`glossary.${term}`, { defaultValue: term });
  const text = children ?? (labelKey ? t(labelKey) : label ?? term);
  return (
    <span className="cursor-help border-b border-dotted border-app-secondary" title={title}>
      {text}
    </span>
  );
}
