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

import { listen } from "@tauri-apps/api/event";
import { useEffect } from "react";

type Payload = { action?: string };

export function usePhantomMenu(opts: {
  exportJsonl: () => void;
  clearEvents: () => void;
  onOpenSettings?: () => void;
  onOpenAbout?: () => void;
}) {
  const { exportJsonl, clearEvents, onOpenSettings, onOpenAbout } = opts;

  useEffect(() => {
    let unlisten: (() => void) | undefined;
    void listen<Payload>("phantom-menu", (e) => {
      const a = e.payload?.action;
      if (a === "export") exportJsonl();
      else if (a === "clear") clearEvents();
      else if (a === "open_settings") onOpenSettings?.();
      else if (a === "about") onOpenAbout?.();
    }).then((fn) => {
      unlisten = fn;
    });
    return () => {
      unlisten?.();
    };
  }, [exportJsonl, clearEvents, onOpenSettings, onOpenAbout]);
}
