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

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        shell: {
          bg: "#1a1b26",
          panel: "#16161e",
          border: "#2d2f3a",
          accent: "#7aa2f7",
          muted: "#565f89",
        },
        app: {
          bg: "var(--app-bg)",
          panel: "var(--app-panel)",
          sidebar: "var(--app-sidebar)",
          separator: "var(--app-separator)",
          label: "var(--app-label)",
          secondary: "var(--app-secondary)",
          accent: "var(--app-accent)",
          "accent-muted": "var(--app-accent-muted)",
          hover: "var(--app-hover)",
          field: "var(--app-field-bg)",
        },
      },
    },
  },
  plugins: [],
};
