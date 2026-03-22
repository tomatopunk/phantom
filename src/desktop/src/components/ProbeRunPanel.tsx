/**
 * Copyright 2026 The Phantom Authors
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import type { TFunction } from "i18next";
import { useCallback, useEffect, useMemo, useState } from "react";
import * as api from "../api";
import {
  buildProbeRunLines,
  type ProbeRunDraft,
  templateAttachPointForPreview,
  templateSecForDraft,
} from "../app/discoverCommands";
import { technicalInputProps } from "../app/technicalInputProps";

type EditorTab = "command" | "source" | "compile";

type Props = {
  t: TFunction;
  draft: ProbeRunDraft | null;
  onDismissDraft: () => void;
  connected: boolean;
  runCommandLine: (line: string) => Promise<void>;
  setCmd: (s: string) => void;
  openConsole: () => void;
};

export function ProbeRunPanel({
  t,
  draft,
  onDismissDraft,
  connected,
  runCommandLine,
  setCmd,
  openConsole,
}: Props) {
  const [hookSec, setHookSec] = useState("pid>0");
  const [breakKernelSec, setBreakKernelSec] = useState("pid>=0");
  const [traceExprs, setTraceExprs] = useState("pid tgid comm");
  const [watchExpr, setWatchExpr] = useState("pid");
  const [cmdLine, setCmdLine] = useState("");
  const [editorTab, setEditorTab] = useState<EditorTab>("command");
  const [preview, setPreview] = useState<api.HookTemplatePreview | null>(null);
  const [previewLoadErr, setPreviewLoadErr] = useState("");
  const [runOut, setRunOut] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!draft) return;
    setHookSec("pid>0");
    setBreakKernelSec("pid>=0");
    setTraceExprs(draft.tab === "kp" ? "pid tgid comm" : "pid tgid");
    setWatchExpr("pid");
    setCmdLine(buildProbeRunLines(draft, {}).join("\n\n"));
    setPreview(null);
    setPreviewLoadErr("");
    setRunOut("");
    setEditorTab("command");
  }, [draft]);

  const cmdFromForm = useMemo(
    () =>
      draft
        ? buildProbeRunLines(draft, { hookSec, breakKernelSec, traceExprs, watchExpr }).join("\n\n") || null
        : null,
    [draft, hookSec, breakKernelSec, traceExprs, watchExpr],
  );

  const attachForPreview = draft ? templateAttachPointForPreview(draft) : null;
  const secForPreview =
    draft && attachForPreview ? templateSecForDraft(draft, hookSec, breakKernelSec) : null;

  const loadPreview = useCallback(async () => {
    if (!connected || !attachForPreview || !secForPreview) {
      setPreview(null);
      return;
    }
    setBusy(true);
    setPreviewLoadErr("");
    try {
      const r = await api.previewHookTemplate(attachForPreview, secForPreview, "");
      setPreview(r);
      if (!r.ok) setPreviewLoadErr(r.error_message || t("probeRun.preview.templateErr"));
    } catch (e) {
      setPreview(null);
      setPreviewLoadErr(String(e));
    } finally {
      setBusy(false);
    }
  }, [attachForPreview, connected, secForPreview, t]);

  useEffect(() => {
    if (editorTab === "source" || editorTab === "compile") {
      void loadPreview();
    }
  }, [editorTab, loadPreview]);

  const onRun = async () => {
    const lines = cmdLine
      .split(/\r?\n/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (lines.length === 0) return;
    setRunOut(t("common.ellipsis"));
    try {
      for (const ln of lines) {
        await runCommandLine(ln);
      }
      setRunOut(t("probeRun.ranOk"));
    } catch (e) {
      setRunOut(String(e));
    }
  };

  const onFillConsole = () => {
    const raw = cmdLine.trim();
    if (!raw) return;
    const lines = raw.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
    const first = lines[0] ?? "";
    setCmd(first);
    openConsole();
    if (lines.length > 1) {
      void navigator.clipboard.writeText(raw);
      setRunOut(t("probeRun.multiLineConsoleClipboard"));
    }
  };

  const tabBtn = (id: EditorTab, label: string) => (
    <button
      key={id}
      type="button"
      className={`rounded-md px-2 py-1 text-[10px] font-medium transition-colors ${
        editorTab === id ? "bg-app-accent-muted text-app-label" : "text-app-secondary hover:bg-app-hover"
      }`}
      onClick={() => setEditorTab(id)}
    >
      {label}
    </button>
  );

  if (!draft) {
    return (
      <div className="flex flex-1 min-h-0 flex-col p-3 text-app-secondary text-xs">
        <p className="leading-relaxed">{t("probeRun.empty")}</p>
      </div>
    );
  }

  const showTemplate = attachForPreview && secForPreview;
  const kindLabel = t(`discover.quick.${draft.kind}`);

  return (
    <div className="flex flex-1 min-h-0 flex-col gap-2 p-3 overflow-hidden">
      <div className="flex flex-wrap items-start justify-between gap-2 shrink-0">
        <div className="min-w-0">
          <div className="text-[10px] uppercase tracking-wide text-app-secondary">{t("probeRun.title")}</div>
          <div className="font-mono-tight text-[11px] text-app-label truncate" title={draft.line}>
            <span className="text-app-accent">{kindLabel}</span>
            <span className="text-app-secondary"> · </span>
            {draft.line.trim()}
          </div>
        </div>
        <button type="button" className="btn-app text-[10px] shrink-0" onClick={onDismissDraft}>
          {t("probeRun.dismiss")}
        </button>
      </div>

      <div className="rounded-md border border-app-separator/80 bg-app-field/40 p-2 space-y-2 text-[10px] shrink-0">
        {draft.kind === "hook" ||
        ((draft.kind === "trace" || draft.kind === "watch") && draft.tab !== "kp") ? (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.hookSec")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={hookSec}
              onChange={(e) => setHookSec(e.target.value)}
              {...technicalInputProps}
            />
          </label>
        ) : null}
        {draft.kind === "break" ||
        ((draft.kind === "trace" || draft.kind === "watch") && draft.tab === "kp") ? (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.breakKernelSec")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={breakKernelSec}
              onChange={(e) => setBreakKernelSec(e.target.value)}
              {...technicalInputProps}
            />
          </label>
        ) : null}
        {draft.kind === "trace" ? (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.traceExprs")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={traceExprs}
              onChange={(e) => setTraceExprs(e.target.value)}
              {...technicalInputProps}
            />
          </label>
        ) : null}
        {draft.kind === "watch" ? (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.watchExpr")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={watchExpr}
              onChange={(e) => setWatchExpr(e.target.value)}
              {...technicalInputProps}
            />
          </label>
        ) : null}
        {draft.kind === "break" && draft.tab === "kp" ? (
          <p className="text-app-secondary/90 leading-snug">{t("probeRun.breakBuiltInHint")}</p>
        ) : null}
        {draft.kind === "trace" || draft.kind === "watch" ? (
          <p className="text-app-secondary/90 leading-snug">{t("probeRun.traceWatchPairHint")}</p>
        ) : null}
        <div className="flex flex-wrap gap-1">
          <button
            type="button"
            className="btn-app text-[10px]"
            disabled={!cmdFromForm}
            onClick={() => cmdFromForm && setCmdLine(cmdFromForm)}
          >
            {t("probeRun.applyForm")}
          </button>
          <button
            type="button"
            className="btn-app text-[10px]"
            disabled={busy || !showTemplate}
            onClick={() => void loadPreview()}
          >
            {t("probeRun.refreshPreview")}
          </button>
        </div>
        {showTemplate ? (
          <div className="font-mono-tight text-[9px] text-app-secondary break-all">
            {t("probeRun.attach")} {attachForPreview}
            {draft.kind === "break" || ((draft.kind === "trace" || draft.kind === "watch") && draft.tab === "kp") ? (
              <span className="block pt-0.5">{t("probeRun.breakPreviewSec", { sec: breakKernelSec })}</span>
            ) : (draft.kind === "trace" || draft.kind === "watch") && draft.tab !== "kp" ? (
              <span className="block pt-0.5">{t("probeRun.hookPreviewSec", { sec: hookSec })}</span>
            ) : null}
          </div>
        ) : (
          <p className="text-app-secondary/90">{t("probeRun.noTemplate")}</p>
        )}
      </div>

      <div className="flex gap-1 shrink-0 border-b border-app-separator pb-1">
        {tabBtn("command", t("probeRun.tab.command"))}
        {tabBtn("source", t("probeRun.tab.source"))}
        {tabBtn("compile", t("probeRun.tab.compile"))}
      </div>

      <div className="flex-1 min-h-0 flex flex-col gap-2 overflow-hidden">
        {editorTab === "command" ? (
          <textarea
            className="flex-1 min-h-[120px] w-full resize-y rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[10px] text-app-label"
            value={cmdLine}
            onChange={(e) => setCmdLine(e.target.value)}
            {...technicalInputProps}
          />
        ) : null}
        {editorTab === "source" ? (
          <div className="flex-1 min-h-0 flex flex-col gap-1">
            {previewLoadErr ? <p className="text-[10px] text-rose-500 shrink-0">{previewLoadErr}</p> : null}
            {busy ? <p className="text-[10px] text-app-secondary shrink-0">{t("common.ellipsis")}</p> : null}
            <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[9px] text-app-label whitespace-pre-wrap break-all">
              {preview?.generated_source_c || (showTemplate ? "" : t("probeRun.noTemplate"))}
            </pre>
          </div>
        ) : null}
        {editorTab === "compile" ? (
          <div className="flex-1 min-h-0 flex flex-col gap-1">
            {previewLoadErr ? <p className="text-[10px] text-rose-500 shrink-0">{previewLoadErr}</p> : null}
            {!connected ? <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.needConnect")}</p> : null}
            {preview?.compile_attempted ? (
              <p
                className={`text-[10px] shrink-0 ${preview.compile_ok ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"}`}
              >
                {preview.compile_ok ? t("probeRun.compile.ok") : t("probeRun.compile.fail")}
              </p>
            ) : (
              <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.compile.skipped")}</p>
            )}
            <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[9px] text-app-secondary whitespace-pre-wrap break-all">
              {(preview?.compiler_output || "").trim() || "—"}
            </pre>
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-1 shrink-0 pt-1 border-t border-app-separator">
        <button
          type="button"
          className="btn-app text-xs"
          disabled={!connected || !cmdLine.trim()}
          onClick={() => void onRun()}
        >
          {t("probeRun.run")}
        </button>
        <button
          type="button"
          className="btn-app text-xs"
          disabled={!cmdLine.trim()}
          onClick={onFillConsole}
        >
          {t("probeRun.toConsole")}
        </button>
      </div>
      {runOut ? <p className="text-[10px] text-app-secondary shrink-0 truncate">{runOut}</p> : null}
    </div>
  );
}
