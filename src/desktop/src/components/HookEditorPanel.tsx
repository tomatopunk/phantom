import Editor, { type OnMount } from "@monaco-editor/react";
import type * as Monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import * as api from "../api";
import { runLocalLint } from "../hook/localLint";
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
      const r = await api.executeCmd(cmd);
      if (r.ok) onProbesChanged?.();
      if (!r.ok) {
        setCompilerOut(r.error_message || r.output || "");
        setAgentProblems([
          {
            key: "tpl-err",
            source: "agent",
            line: 1,
            column: 1,
            message: r.error_message || r.output || "hook add failed",
            detail: r.output,
          },
        ]);
      }
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
    <div className="border-t border-shell-border p-2 space-y-1 flex flex-col flex-1 min-h-0 overflow-hidden">
      <div className="text-xs text-shell-muted">{t("hook.title")}</div>
      <div className="flex flex-wrap gap-1 items-center">
        <select
          className="flex-1 min-w-[8rem] bg-black/40 border border-shell-border rounded px-1 text-[10px]"
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
        <button
          type="button"
          className="px-2 text-[10px] rounded border border-shell-border"
          disabled={!selectedPreset}
          onClick={applyPreset}
        >
          {t("hookEditor.loadPreset")}
        </button>
        {selectedPreset?.mode === "template_sec" && (
          <button
            type="button"
            disabled={!connected || busy}
            className="px-2 text-[10px] rounded border border-shell-border disabled:opacity-40"
            onClick={() => void runTemplatePreset()}
          >
            {t("hookEditor.applyTemplate")}
          </button>
        )}
      </div>
      {selectedPreset && <p className="text-[10px] text-shell-muted">{selectedPreset.description}</p>}
      <div className="border border-shell-border rounded overflow-hidden min-h-[200px] h-[min(40vh,320px)] shrink-0">
        <Editor
          height="100%"
          defaultLanguage="c"
          theme="vs-dark"
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
      <div className="flex gap-1 flex-wrap">
        <input
          className="flex-1 min-w-[8rem] bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
          value={attach}
          onChange={(e) => setAttach(e.target.value)}
          placeholder={t("hook.attachPh")}
        />
        <input
          className="w-28 bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
          value={programName}
          onChange={(e) => setProgramName(e.target.value)}
          placeholder={t("hook.progNamePh")}
        />
        <button
          type="button"
          disabled={!connected || busy}
          className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
          onClick={() => void runCompileAttach()}
        >
          {t("hookEditor.compileAttach")}
        </button>
      </div>
      <div className="text-[10px] text-shell-muted">{t("hookEditor.hintSec")}</div>
      <div className="border border-shell-border rounded bg-black/20 max-h-28 overflow-auto shrink-0">
        <div className="px-1 py-0.5 text-shell-muted text-[10px] border-b border-shell-border/50">{t("hookEditor.problems")}</div>
        {problems.length === 0 ? (
          <p className="p-1 text-shell-muted text-[10px]">{t("hookEditor.noProblems")}</p>
        ) : (
          <ul className="p-1 space-y-0.5">
            {problems.map((p) => (
              <li key={p.key}>
                <button
                  type="button"
                  className="text-left w-full hover:bg-white/5 rounded px-0.5 font-mono-tight"
                  onClick={() => jumpToLine(p.line)}
                >
                  <span className={p.source === "agent" ? "text-amber-300" : "text-blue-300"}>[{p.source}]</span> L{p.line}:{p.column}{" "}
                  {p.message}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
      {compilerOut ? (
        <pre className="text-[10px] bg-black/40 border border-shell-border rounded p-1 max-h-32 overflow-auto whitespace-pre-wrap shrink-0">
          {compilerOut}
        </pre>
      ) : null}
    </div>
  );
}
