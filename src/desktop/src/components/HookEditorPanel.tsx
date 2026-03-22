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

import Editor, { type OnMount } from "@monaco-editor/react";
import type * as Monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import * as api from "../api";
import { runLocalLint } from "../hook/localLint";
import { useResolvedDark } from "../theme/ThemeProvider";
import { probePresets } from "../presets/types";

type Problem = {
  key: string;
  source: "local" | "agent";
  line: number;
  column: number;
  message: string;
  detail?: string;
};

function mapSeverity(sev: string, monaco: typeof Monaco): Monaco.MarkerSeverity {
  const s = sev.toLowerCase();
  if (s === "error" || s === "fatal") return monaco.MarkerSeverity.Error;
  if (s === "warning") return monaco.MarkerSeverity.Warning;
  return monaco.MarkerSeverity.Info;
}

function isUserSourcePath(p: string): boolean {
  const b = p.replace(/\\/g, "/").split("/").pop() || p;
  return b === "program.c" || b === "hook.c" || p === "<stdin>";
}

export function HookEditorPanel({
  connected,
  onProbesChanged,
}: {
  connected: boolean;
  onProbesChanged?: () => void;
}) {
  const { t } = useTranslation();
  const resolvedDark = useResolvedDark();
  const monacoTheme = resolvedDark ? "vs-dark" : "light";
  const [src, setSrc] = useState(
    '#include <linux/bpf.h>\n#include <bpf/bpf_helpers.h>\n\nchar LICENSE[] SEC("license") = "Dual BSD/GPL";\n',
  );
  const [attach, setAttach] = useState("tracepoint:sched:sched_switch");
  const [programName, setProgramName] = useState("");
  const [presetId, setPresetId] = useState("");
  const [busy, setBusy] = useState(false);
  const [compilerOut, setCompilerOut] = useState("");
  const [localProblems, setLocalProblems] = useState<Problem[]>([]);
  const [agentProblems, setAgentProblems] = useState<Problem[]>([]);
  const problems = useMemo(() => [...localProblems, ...agentProblems], [localProblems, agentProblems]);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<typeof Monaco | null>(null);
  const modelRef = useRef<Monaco.editor.ITextModel | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const applyLocalMarkers = useCallback((text: string, att: string, monaco: typeof Monaco) => {
    const model = modelRef.current;
    if (!model) return;
    const local = runLocalLint(text, att);
    const markers: Monaco.editor.IMarkerData[] = local.map((p) => ({
      severity: monaco.MarkerSeverity.Warning,
      startLineNumber: Math.max(1, p.line),
      startColumn: Math.max(1, p.column),
      endLineNumber: Math.max(1, p.line),
      endColumn: Math.max(2, p.column + 1),
      message: `[${t("hookEditor.sourceLocal")}] ${p.message}`,
    }));
    monaco.editor.setModelMarkers(model, "local", markers);
    setLocalProblems(
      local.map((p, i) => ({
        key: `l-${i}`,
        source: "local" as const,
        line: p.line,
        column: p.column,
        message: p.message,
      })),
    );
  }, [t]);

  const clearAgentDiagnostics = useCallback(() => {
    const monaco = monacoRef.current;
    const model = modelRef.current;
    if (monaco && model) monaco.editor.setModelMarkers(model, "agent", []);
    setAgentProblems([]);
    setCompilerOut("");
  }, []);

  useEffect(() => {
    const monaco = monacoRef.current;
    if (!monaco) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      clearAgentDiagnostics();
      applyLocalMarkers(src, attach, monaco);
    }, 250);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [src, attach, applyLocalMarkers, clearAgentDiagnostics]);

  const onMount: OnMount = useCallback(
    (editor, monaco) => {
      editorRef.current = editor;
      monacoRef.current = monaco;
      modelRef.current = editor.getModel();
      applyLocalMarkers(editor.getValue(), attach, monaco);
    },
    [attach, applyLocalMarkers],
  );

  const selectedPreset = useMemo(
    () => probePresets.find((p) => p.id === presetId) ?? null,
    [presetId],
  );

  const applyPreset = () => {
    if (!selectedPreset) return;
    setAttach(selectedPreset.attach);
    if (selectedPreset.mode === "full_c" && selectedPreset.cTemplate) {
      setSrc(selectedPreset.cTemplate);
    }
  };

  const runTemplatePreset = async () => {
    if (!selectedPreset || selectedPreset.mode !== "template_sec" || !selectedPreset.sec) return;
    setBusy(true);
    clearAgentDiagnostics();
    try {
      const secEsc = selectedPreset.sec.replace(/"/g, '\\"');
      const cmd = `hook add --point ${selectedPreset.attach} --lang c --sec "${secEsc}"`;
      await api.executeCmd(cmd);
      onProbesChanged?.();
    } catch (e) {
      const msg = String(e);
      setCompilerOut(msg);
      setAgentProblems([
        {
          key: "tpl-err",
          source: "agent",
          line: 1,
          column: 1,
          message: msg,
          detail: msg,
        },
      ]);
    } finally {
      setBusy(false);
    }
  };

  const runCompileAttach = async () => {
    if (!connected) return;
    const monaco = monacoRef.current;
    const model = modelRef.current;
    setBusy(true);
    clearAgentDiagnostics();
    try {
      const r = await api.compileHook(src, attach, programName);
      if (r.ok) {
        if (monaco && model) monaco.editor.setModelMarkers(model, "agent", []);
        onProbesChanged?.();
      } else {
        const diags = r.diagnostics ?? [];
        const out = r.compiler_output || r.error_message || "";
        setCompilerOut(out);
        const ap: Problem[] = diags
          .filter((d) => isUserSourcePath(d.path) || diags.length <= 2)
          .map((d, i) => ({
            key: `a-${i}`,
            source: "agent" as const,
            line: d.line > 0 ? d.line : 1,
            column: d.column > 0 ? d.column : 1,
            message: `${d.severity}: ${d.message}`,
            detail: out,
          }));
        if (ap.length === 0 && out) {
          ap.push({
            key: "a-raw",
            source: "agent",
            line: 1,
            column: 1,
            message: r.error_message || "compile failed",
            detail: out,
          });
        }
        setAgentProblems(ap);
        if (monaco && model) {
          const relevant = diags.filter(
            (d) => isUserSourcePath(d.path) || diags.every((x) => !isUserSourcePath(x.path)),
          );
          const markers: Monaco.editor.IMarkerData[] = relevant.map((d) => ({
            severity: mapSeverity(d.severity, monaco),
            startLineNumber: Math.max(1, d.line),
            startColumn: Math.max(1, d.column),
            endLineNumber: Math.max(1, d.line),
            endColumn: Math.max(2, d.column + 1),
            message: `[agent] ${d.message}`,
          }));
          monaco.editor.setModelMarkers(model, "agent", markers);
        }
      }
    } finally {
      setBusy(false);
    }
  };

  const jumpToLine = (line: number) => {
    const ed = editorRef.current;
    if (!ed) return;
    ed.revealLineInCenter(Math.max(1, line));
    ed.setPosition({ lineNumber: Math.max(1, line), column: 1 });
    ed.focus();
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden gap-2 p-3">
      <div className="text-xs font-medium text-app-label shrink-0">{t("hook.title")}</div>
      <div className="flex flex-wrap gap-2 items-center shrink-0">
        <select
          className="input-app flex-1 min-w-[8rem] text-[10px]"
          value={presetId}
          onChange={(e) => setPresetId(e.target.value)}
        >
          <option value="">{t("hookEditor.presetPlaceholder")}</option>
          {probePresets.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <button type="button" className="btn-app text-[10px]" disabled={!selectedPreset} onClick={applyPreset}>
          {t("hookEditor.loadPreset")}
        </button>
        {selectedPreset?.mode === "template_sec" && (
          <button
            type="button"
            disabled={!connected || busy}
            className="btn-app text-[10px]"
            onClick={() => void runTemplatePreset()}
          >
            {t("hookEditor.applyTemplate")}
          </button>
        )}
      </div>
      {selectedPreset && <p className="text-[10px] text-app-secondary shrink-0">{selectedPreset.description}</p>}
      <div className="flex min-h-[200px] flex-1 flex-col overflow-hidden rounded-md border border-app-separator">
        <Editor
          height="100%"
          defaultLanguage="c"
          theme={monacoTheme}
          value={src}
          onChange={(v) => setSrc(v ?? "")}
          onMount={onMount}
          options={{
            minimap: { enabled: true },
            fontSize: 12,
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
          }}
        />
      </div>
      <div className="flex flex-wrap gap-2 shrink-0">
        <input
          className="input-app flex-1 min-w-[8rem] text-xs font-mono-tight"
          value={attach}
          onChange={(e) => setAttach(e.target.value)}
          placeholder={t("hook.attachPh")}
        />
        <input
          className="input-app w-28 text-xs font-mono-tight"
          value={programName}
          onChange={(e) => setProgramName(e.target.value)}
          placeholder={t("hook.progNamePh")}
        />
        <button type="button" disabled={!connected || busy} className="btn-app-primary text-xs" onClick={() => void runCompileAttach()}>
          {t("hookEditor.compileAttach")}
        </button>
      </div>
      <div className="text-[10px] text-app-secondary shrink-0">{t("hookEditor.hintSec")}</div>
      <div className="max-h-28 overflow-auto shrink-0 rounded-md border border-app-separator bg-app-field">
        <div className="px-2 py-1 text-app-secondary text-[10px] border-b border-app-separator/60">{t("hookEditor.problems")}</div>
        {problems.length === 0 ? (
          <p className="p-2 text-app-secondary text-[10px]">{t("hookEditor.noProblems")}</p>
        ) : (
          <ul className="p-1 space-y-0.5">
            {problems.map((p) => (
              <li key={p.key}>
                <button
                  type="button"
                  className="text-left w-full rounded px-0.5 font-mono-tight hover:bg-black/5 dark:hover:bg-white/5"
                  onClick={() => jumpToLine(p.line)}
                >
                  <span
                    className={
                      p.source === "agent" ? "text-amber-800 dark:text-amber-300" : "text-blue-700 dark:text-blue-300"
                    }
                  >
                    [{p.source}]
                  </span>{" "}
                  L{p.line}:{p.column}{" "}
                  {p.message}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
      {compilerOut ? (
        <pre className="text-[10px] rounded-md border border-app-separator bg-app-field p-2 max-h-32 overflow-auto whitespace-pre-wrap shrink-0 text-app-label">
          {compilerOut}
        </pre>
      ) : null}
    </div>
  );
}
