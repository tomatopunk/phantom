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
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { commandSuggestionsAtCursor } from "../app/commandCompletions";
import { technicalInputProps } from "../app/technicalInputProps";

type Props = {
  t: TFunction;
  cmd: string;
  setCmd: (s: string) => void;
  cmdOut: string;
  runCmd: () => void;
  connected: boolean;
};

export function CmdReplBlock({ t, cmd, setCmd, cmdOut, runCmd, connected }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
  const prevCmd = useRef(cmd);
  const [caret, setCaret] = useState(cmd.length);
  const [focused, setFocused] = useState(false);
  const [activeIdx, setActiveIdx] = useState(0);

  useEffect(() => {
    if (prevCmd.current !== cmd && document.activeElement !== inputRef.current) {
      setCaret(cmd.length);
    }
    prevCmd.current = cmd;
  }, [cmd]);

  const sug = useMemo(() => commandSuggestionsAtCursor(cmd, Math.min(Math.max(0, caret), cmd.length)), [cmd, caret]);

  useEffect(() => {
    setActiveIdx(0);
  }, [cmd, caret, sug.items.join("\n")]);

  const applyItem = useCallback(
    (item: string) => {
      const { replaceFrom, replaceTo } = commandSuggestionsAtCursor(cmd, Math.min(Math.max(0, caret), cmd.length));
      const next = cmd.slice(0, replaceFrom) + item + cmd.slice(replaceTo);
      setCmd(next);
      const pos = replaceFrom + item.length;
      queueMicrotask(() => {
        const el = inputRef.current;
        if (el) {
          el.focus();
          el.setSelectionRange(pos, pos);
        }
        setCaret(pos);
      });
    },
    [cmd, caret, setCmd],
  );

  const syncCaretFromInput = () => {
    const el = inputRef.current;
    if (el) setCaret(el.selectionStart ?? cmd.length);
  };

  const showList = focused && sug.items.length > 0;

  return (
    <div className="flex flex-1 min-h-0 flex-col gap-2 p-3">
      <div className="text-xs font-medium text-app-label shrink-0">{t("cmd.title")}</div>
      <p className="text-[10px] text-app-secondary shrink-0 leading-snug">{t("cmd.completionHint")}</p>
      <div className="relative flex gap-2 shrink-0">
        <input
          ref={inputRef}
          id="phantom-cmd-input"
          className="input-app flex-1 font-mono-tight text-xs"
          value={cmd}
          {...technicalInputProps}
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={showList}
          aria-controls="phantom-cmd-suggestions"
          aria-label={t("cmd.inputAria")}
          onChange={(e) => {
            setCmd(e.target.value);
            setCaret(e.target.selectionStart ?? e.target.value.length);
          }}
          onSelect={syncCaretFromInput}
          onKeyUp={syncCaretFromInput}
          onClick={syncCaretFromInput}
          onFocus={() => setFocused(true)}
          onBlur={() => {
            window.setTimeout(() => setFocused(false), 120);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              runCmd();
              return;
            }
            if (e.key === "Tab" && sug.items.length > 0) {
              e.preventDefault();
              applyItem(sug.items[activeIdx % sug.items.length]!);
              return;
            }
            if (e.key === "ArrowDown" && sug.items.length > 0) {
              e.preventDefault();
              setActiveIdx((i) => (i + 1) % sug.items.length);
              return;
            }
            if (e.key === "ArrowUp" && sug.items.length > 0) {
              e.preventDefault();
              setActiveIdx((i) => (i - 1 + sug.items.length) % sug.items.length);
              return;
            }
            if (e.key === "Escape" && showList) {
              e.preventDefault();
              inputRef.current?.blur();
            }
          }}
        />
        <button type="button" disabled={!connected} className="btn-app text-xs shrink-0" onClick={runCmd}>
          {t("cmd.run")}
        </button>
        {showList ? (
          <ul
            id="phantom-cmd-suggestions"
            role="listbox"
            aria-label={t("cmd.completionAria")}
            className="absolute left-0 right-[5.5rem] top-full z-20 mt-0.5 max-h-48 overflow-auto rounded-md border border-app-separator bg-app-panel py-0.5 shadow-md"
          >
            {sug.items.map((item, i) => (
              <li
                key={`${i}-${item.slice(0, 48)}`}
                role="option"
                aria-selected={i === activeIdx % sug.items.length}
                className={`cursor-pointer px-2 py-1 font-mono-tight text-[10px] ${
                  i === activeIdx % sug.items.length ? "bg-app-accent-muted text-app-label" : "text-app-secondary hover:bg-app-hover"
                }`}
                onMouseDown={(ev) => {
                  ev.preventDefault();
                  applyItem(item);
                }}
                onMouseEnter={() => setActiveIdx(i)}
              >
                {item}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-field p-2 text-[10px] text-app-label whitespace-pre-wrap font-mono-tight">
        {cmdOut}
      </pre>
    </div>
  );
}
