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

/** Must match the inline script in index.html. */
export const APPEARANCE_STORAGE_KEY = "phantom-appearance";

export type AppearancePreference = "light" | "dark" | "system";

export type ResolvedColorScheme = "light" | "dark";

/** Current OS / browser light-dark preference (call when "follow system" is active). */
export function readSystemPrefersDark(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export function readStoredAppearance(): AppearancePreference {
  try {
    const raw = localStorage.getItem(APPEARANCE_STORAGE_KEY);
    if (raw === "light" || raw === "dark" || raw === "system") return raw;
  } catch {
    /* ignore */
  }
  return "system";
}

export function applyDarkClassToDocument(dark: boolean): void {
  document.documentElement.classList.toggle("dark", dark);
}

export function persistAppearance(pref: AppearancePreference): void {
  try {
    localStorage.setItem(APPEARANCE_STORAGE_KEY, pref);
  } catch {
    /* ignore */
  }
}
