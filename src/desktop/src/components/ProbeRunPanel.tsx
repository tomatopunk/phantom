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
  type HookBodyMode,
  type ProbeRunDraft,
  templateAttachPointForPreview,
  templatePreviewReady,
  templatePreviewSecAndCode,
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
  const [breakAttach, setBreakAttach] = useState("");
  const [breakProgram, setBreakProgram] = useState("");
  const [breakUserSource, setBreakUserSource] = useState(
    '/* User eBPF: define SEC("...") and ringbuf map per agent docs */\n#include <linux/bpf.h>\n#include <bpf/bpf_helpers.h>\n',
  );
  const [hookBodyMode, setHookBodyMode] = useState<HookBodyMode>("sec");
  const [hookCodeSnippet, setHookCodeSnippet] = useState(
    "(void)ctx;\n/* Insert C before implicit bpf_ringbuf_output on &ev; see template in Source tab. */\n",
  );
  const [traceExprs, setTraceExprs] = useState("pid tgid comm");
  const [watchExpr, setWatchExpr] = useState("pid");
  const [cmdLine, setCmdLine] = useState("");
  const [editorTab, setEditorTab] = useState<EditorTab>("command");
  const [preview, setPreview] = useState<api.HookTemplatePreview | null>(null);
  const [previewLoadErr, setPreviewLoadErr] = useState("");
  const [breakValidate, setBreakValidate] = useState<api.ValidateCompileSourceResult | null>(null);
  const [breakValidateBusy, setBreakValidateBusy] = useState(false);
  const [runOut, setRunOut] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!draft) return;
    setHookSec("pid>0");
    const row = draft.line.trim();
    setBreakAttach(draft.tab === "kp" && row ? `kprobe:${row}` : "");
    setBreakProgram("");
    setBreakUserSource(
      '/* User eBPF: define SEC("...") and ringbuf map per agent docs */\n#include <linux/bpf.h>\n#include <bpf/bpf_helpers.h>\n',
    );
    setHookBodyMode("sec");
    setHookCodeSnippet(
      "(void)ctx;\n/* Insert C before implicit bpf_ringbuf_output on &ev; see template in Source tab. */\n",
    );
    setTraceExprs(draft.tab === "kp" ? "pid tgid comm" : "pid tgid");
    setWatchExpr("pid");
    setCmdLine(
      buildProbeRunLines(draft, {
        breakUserSource:
          '/* User eBPF: define SEC("...") and ringbuf map per agent docs */\n#include <linux/bpf.h>\n#include <bpf/bpf_helpers.h>\n',
        breakAttach: draft.tab === "kp" && row ? `kprobe:${row}` : "",
      }).join("\n\n"),
    );
    setPreview(null);
    setPreviewLoadErr("");
    setRunOut("");
    setEditorTab("command");
  }, [draft]);

  const cmdFromForm = useMemo(
    () =>
      draft
        ? buildProbeRunLines(draft, {
            hookSec,
            traceExprs,
            watchExpr,
            hookBodyMode,
            hookCodeSnippet,
            breakUserSource,
            breakAttach,
            breakProgram,
          }).join("\n\n") || null
        : null,
    [draft, hookSec, traceExprs, watchExpr, hookBodyMode, hookCodeSnippet, breakUserSource, breakAttach, breakProgram],
  );

  const attachForPreview = draft ? templateAttachPointForPreview(draft) : null;
  const previewSecCode = useMemo(
    () =>
      draft
        ? templatePreviewSecAndCode(draft, hookSec, "", hookBodyMode, hookCodeSnippet)
        : { sec: "", code: "" },
    [draft, hookSec, hookBodyMode, hookCodeSnippet],
  );
  const previewReady =
    !!draft &&
    !!attachForPreview &&
    templatePreviewReady(draft, hookSec, "", hookBodyMode, hookCodeSnippet);

  const loadPreview = useCallback(async () => {
    if (!connected || !attachForPreview || !previewReady) {
      setPreview(null);
      return;
    }
    setBusy(true);
    setPreviewLoadErr("");
    try {
      const r = await api.previewHookTemplate(attachForPreview, previewSecCode.sec, previewSecCode.code);
      setPreview(r);
      if (!r.ok) setPreviewLoadErr(r.error_message || t("probeRun.preview.templateErr"));
    } catch (e) {
      setPreview(null);
      setPreviewLoadErr(String(e));
    } finally {
      setBusy(false);
    }
  }, [attachForPreview, connected, previewReady, previewSecCode, t]);

  useEffect(() => {
    if (editorTab === "source" || (editorTab === "compile" && draft?.kind !== "break")) {
      void loadPreview();
    }
  }, [editorTab, draft?.kind, loadPreview]);

  useEffect(() => {
    if (!(draft?.kind === "break" && editorTab === "compile" && connected)) {
      setBreakValidate(null);
      setBreakValidateBusy(false);
      return;
    }
    let cancelled = false;
    const tid = window.setTimeout(() => {
      void (async () => {
        setBreakValidateBusy(true);
        try {
          const r = await api.validateCompileSource(breakUserSource);
          if (!cancelled) setBreakValidate(r);
        } catch (e) {
          if (!cancelled) {
            setBreakValidate({
              ok: false,
              error_message: String(e),
              diagnostics: [],
              compiler_output: "",
            });
          }
        } finally {
          if (!cancelled) setBreakValidateBusy(false);
        }
      })();
    }, 450);
    return () => {
      cancelled = true;
      window.clearTimeout(tid);
    };
  }, [draft?.kind, editorTab, connected, breakUserSource]);

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

  const showTemplate = previewReady;
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
        {draft.kind === "hook" ? (
          <>
            <p className="text-app-secondary/90 leading-snug m-0">{t("probeRun.hookVsBreakHint")}</p>
            <div className="flex flex-wrap gap-2 items-center">
              <span className="text-app-secondary">{t("probeRun.hookBodyMode")}</span>
              <label className="inline-flex items-center gap-1 cursor-pointer">
                <input
                  type="radio"
                  name="hookBodyMode"
                  checked={hookBodyMode === "sec"}
                  onChange={() => setHookBodyMode("sec")}
                />
                <span>{t("probeRun.hookBodySec")}</span>
              </label>
              <label className="inline-flex items-center gap-1 cursor-pointer">
                <input
                  type="radio"
                  name="hookBodyMode"
                  checked={hookBodyMode === "code"}
                  onChange={() => setHookBodyMode("code")}
                />
                <span>{t("probeRun.hookBodyCode")}</span>
              </label>
            </div>
            {hookBodyMode === "sec" ? (
              <label className="flex flex-col gap-0.5">
                <span className="text-app-secondary">{t("probeRun.hookSec")}</span>
                <input
                  className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                  value={hookSec}
                  onChange={(e) => setHookSec(e.target.value)}
                  {...technicalInputProps}
                />
              </label>
            ) : (
              <label className="flex flex-col gap-0.5 min-h-0">
                <span className="text-app-secondary">{t("probeRun.hookCodeSnippet")}</span>
                <textarea
                  className="w-full min-h-[100px] resize-y rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-[10px] text-app-label"
                  value={hookCodeSnippet}
                  onChange={(e) => setHookCodeSnippet(e.target.value)}
                  spellCheck={false}
                />
              </label>
            )}
          </>
        ) : null}
        {draft.kind === "trace" || draft.kind === "watch" ? (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.pairProbeSec")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={hookSec}
              onChange={(e) => setHookSec(e.target.value)}
              {...technicalInputProps}
            />
          </label>
        ) : null}
        {draft.kind === "break" && draft.tab === "kp" ? (
          <>
            <label className="flex flex-col gap-0.5">
              <span className="text-app-secondary">{t("probeRun.breakAttach")}</span>
              <input
                className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                value={breakAttach}
                onChange={(e) => setBreakAttach(e.target.value)}
                {...technicalInputProps}
              />
            </label>
            <label className="flex flex-col gap-0.5">
              <span className="text-app-secondary">{t("probeRun.breakProgram")}</span>
              <input
                className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                value={breakProgram}
                onChange={(e) => setBreakProgram(e.target.value)}
                placeholder={t("probeRun.breakProgramPh")}
                {...technicalInputProps}
              />
            </label>
            <label className="flex flex-col gap-0.5 min-h-0">
              <span className="text-app-secondary">{t("probeRun.breakUserSource")}</span>
              <textarea
                className="w-full min-h-[120px] resize-y rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-[10px] text-app-label"
                value={breakUserSource}
                onChange={(e) => setBreakUserSource(e.target.value)}
                spellCheck={false}
              />
            </label>
          </>
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
          <p className="text-app-secondary/90 leading-snug">{t("probeRun.breakUserEbpfHint")}</p>
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
            disabled={busy || !attachForPreview || !previewReady}
            onClick={() => void loadPreview()}
          >
            {t("probeRun.refreshPreview")}
          </button>
        </div>
        {showTemplate ? (
          <div className="font-mono-tight text-[9px] text-app-secondary break-all">
            {t("probeRun.attach")} {attachForPreview}
            {draft.kind === "hook" && hookBodyMode === "sec" ? (
              <span className="block pt-0.5">{t("probeRun.hookPreviewSec", { sec: hookSec })}</span>
            ) : draft.kind === "hook" && hookBodyMode === "code" ? (
              <span className="block pt-0.5">{t("probeRun.hookPreviewCode")}</span>
            ) : draft.kind === "trace" || draft.kind === "watch" ? (
              <span className="block pt-0.5">{t("probeRun.pairProbePreviewSec", { sec: hookSec })}</span>
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
          draft.kind === "break" ? (
            <div className="flex-1 min-h-0 flex flex-col gap-1">
              {!connected ? <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.needConnect")}</p> : null}
              {breakValidateBusy ? <p className="text-[10px] text-app-secondary shrink-0">{t("common.ellipsis")}</p> : null}
              {breakValidate ? (
                <>
                  <p
                    className={`text-[10px] shrink-0 ${breakValidate.ok ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"}`}
                  >
                    {breakValidate.ok
                      ? t("probeRun.compile.ok")
                      : breakValidate.error_message || t("probeRun.compile.fail")}
                  </p>
                  {breakValidate.diagnostics && breakValidate.diagnostics.length > 0 ? (
                    <ul className="text-[9px] text-rose-600 dark:text-rose-400 list-disc pl-4 shrink-0 max-h-[80px] overflow-auto m-0">
                      {breakValidate.diagnostics.map((d, i) => (
                        <li key={i}>{d.message}</li>
                      ))}
                    </ul>
                  ) : null}
                  <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[9px] text-app-secondary whitespace-pre-wrap break-all">
                    {(breakValidate.compiler_output || "").trim() || "—"}
                  </pre>
                </>
              ) : connected ? (
                <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.breakCompileWaiting")}</p>
              ) : null}
            </div>
          ) : (
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
          )
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
