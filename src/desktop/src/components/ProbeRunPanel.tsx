/**
 * Copyright 2026 The Phantom Authors
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import type { TFunction } from "i18next";
import { useEffect, useMemo, useState } from "react";
import * as api from "../api";
import {
  buildProbeRunLines,
  catalogProbeIdForDiscoveryRow,
  defaultHookSourceForKprobe,
  defaultHookSourceForProbePoint,
  escapeForShellDoubleQuotes,
  type ProbeRunDraft,
  templateProbePointHintForDraft,
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
  const [probeId, setProbeId] = useState("");
  const [breakFilter, setBreakFilter] = useState("");
  const [breakLimit, setBreakLimit] = useState("");
  const [watchArgs, setWatchArgs] = useState("");
  const [hookProgram, setHookProgram] = useState("");
  const [hookUserSource, setHookUserSource] = useState("");
  const [cmdLine, setCmdLine] = useState("");
  const [editorTab, setEditorTab] = useState<EditorTab>("command");
  const [hookValidate, setHookValidate] = useState<api.ValidateCompileSourceResult | null>(null);
  const [hookValidateBusy, setHookValidateBusy] = useState(false);
  const [runOut, setRunOut] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!draft) return;
    const pid = catalogProbeIdForDiscoveryRow(draft.tab, draft.line) ?? "";
    setProbeId(pid);
    setBreakFilter("");
    setBreakLimit("");
    setWatchArgs("");
    setHookProgram("");
    const hint = templateProbePointHintForDraft(draft);
    setHookUserSource(hint ? defaultHookSourceForProbePoint(hint) : defaultHookSourceForKprobe("do_nanosleep"));
    setCmdLine(buildProbeRunLines(draft, {}).join("\n\n"));
    setHookValidate(null);
    setRunOut("");
    setEditorTab("command");
  }, [draft]);

  const cmdFromForm = useMemo(() => {
    if (!draft) return null;
    const limStr = breakLimit.trim();
    let lim: number | undefined;
    if (limStr !== "") {
      const n = parseInt(limStr, 10);
      if (Number.isFinite(n) && n >= 0) lim = n;
    }
    if (draft.kind === "break" && probeId.trim()) {
      let cmd = `break ${probeId.trim()}`;
      const f = breakFilter.trim();
      if (f) cmd += ` --filter "${escapeForShellDoubleQuotes(f)}"`;
      if (lim !== undefined) cmd += ` --limit ${lim}`;
      return cmd;
    }
    if (draft.kind === "watch" && probeId.trim()) {
      let cmd = `watch --sec ${probeId.trim()}`;
      const wa = watchArgs.trim();
      if (wa) cmd += ` --args ${wa}`;
      return cmd;
    }
    const lines = buildProbeRunLines(draft, {
      breakFilter: breakFilter.trim() || undefined,
      breakLimit: lim,
      watchArgs: watchArgs.trim() || undefined,
    });
    return lines.join("\n\n") || null;
  }, [draft, probeId, breakFilter, breakLimit, watchArgs]);

  const needsHookCompile = !!draft && draft.kind === "hook";
  const hookCompileReady = connected && (hookUserSource ?? "").trim() !== "";

  useEffect(() => {
    if (!(needsHookCompile && editorTab === "compile" && connected)) {
      setHookValidate(null);
      setHookValidateBusy(false);
      return;
    }
    let cancelled = false;
    const tid = window.setTimeout(() => {
      void (async () => {
        setHookValidateBusy(true);
        try {
          const r = await api.validateCompileSource(hookUserSource);
          if (!cancelled) setHookValidate(r);
        } catch (e) {
          if (!cancelled) {
            setHookValidate({
              ok: false,
              error_message: String(e),
              diagnostics: [],
              compiler_output: "",
            });
          }
        } finally {
          if (!cancelled) setHookValidateBusy(false);
        }
      })();
    }, 450);
    return () => {
      cancelled = true;
      window.clearTimeout(tid);
    };
  }, [needsHookCompile, editorTab, connected, hookUserSource]);

  const onRun = async () => {
    if (!draft) return;
    setRunOut(t("common.ellipsis"));
    try {
      if (draft.kind === "hook") {
        if (!connected || !hookCompileReady) {
          setRunOut(t("probeRun.needConnect"));
          return;
        }
        setBusy(true);
        const r = await api.compileHook(hookUserSource, hookProgram, 0);
        setBusy(false);
        if (!r.ok) {
          setRunOut(r.error_message || t("probeRun.compile.fail"));
          return;
        }
        setRunOut(t("probeRun.ranOk"));
        return;
      }
      const line = (cmdLine || cmdFromForm || "").trim();
      if (!line) {
        setRunOut("");
        return;
      }
      setBusy(true);
      await runCommandLine(line);
      setBusy(false);
      setRunOut(t("probeRun.ranOk"));
    } catch (e) {
      setBusy(false);
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

  const kindLabel = t(`discover.quick.${draft.kind}`);
  const probeHint = templateProbePointHintForDraft(draft);

  const runDisabled =
    draft.kind === "hook" ? !connected || busy || !hookCompileReady : !connected || busy || !(cmdLine.trim() || cmdFromForm);

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
          <p className="text-app-secondary/90 leading-snug m-0">{t("probeRun.hookVsBreakHint")}</p>
        ) : null}

        {(draft.kind === "break" || draft.kind === "watch") && (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.probeId")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={probeId}
              onChange={(e) => setProbeId(e.target.value)}
              placeholder="kprobe.do_sys_open"
              {...technicalInputProps}
            />
          </label>
        )}

        {draft.kind === "break" && (
          <>
            <label className="flex flex-col gap-0.5">
              <span className="text-app-secondary">{t("probeRun.breakFilter")}</span>
              <input
                className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                value={breakFilter}
                onChange={(e) => setBreakFilter(e.target.value)}
                placeholder='pid==1'
                {...technicalInputProps}
              />
            </label>
            <label className="flex flex-col gap-0.5">
              <span className="text-app-secondary">{t("probeRun.breakLimit")}</span>
              <input
                className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                value={breakLimit}
                onChange={(e) => setBreakLimit(e.target.value)}
                placeholder="0"
                {...technicalInputProps}
              />
            </label>
          </>
        )}

        {draft.kind === "watch" && (
          <label className="flex flex-col gap-0.5">
            <span className="text-app-secondary">{t("probeRun.watchArgs")}</span>
            <input
              className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
              value={watchArgs}
              onChange={(e) => setWatchArgs(e.target.value)}
              placeholder="2,3,4"
              {...technicalInputProps}
            />
          </label>
        )}

        {draft.kind === "hook" && (
          <>
            <label className="flex flex-col gap-0.5">
              <span className="text-app-secondary">{t("probeRun.hookProgram")}</span>
              <input
                className="w-full rounded border border-app-separator bg-app-bg px-1.5 py-1 font-mono-tight text-app-label"
                value={hookProgram}
                onChange={(e) => setHookProgram(e.target.value)}
                placeholder={t("probeRun.breakProgramPh")}
                {...technicalInputProps}
              />
            </label>
          </>
        )}

        {draft.kind === "break" ? (
          <p className="text-app-secondary/90 leading-snug m-0">{t("probeRun.breakTemplateHint")}</p>
        ) : null}
        {draft.kind === "watch" ? (
          <p className="text-app-secondary/90 leading-snug m-0">{t("probeRun.watchHint")}</p>
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
        </div>
        {probeHint ? (
          <div className="font-mono-tight text-[9px] text-app-secondary break-all">
            {t("probeRun.secHint")} {probeHint}
          </div>
        ) : draft.kind === "hook" ? (
          <p className="text-app-secondary/90">{t("probeRun.noTemplate")}</p>
        ) : null}
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
            {needsHookCompile ? (
              <textarea
                className="flex-1 min-h-[200px] w-full resize-y rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[10px] text-app-label"
                value={hookUserSource}
                onChange={(e) => setHookUserSource(e.target.value)}
                spellCheck={false}
              />
            ) : (
              <p className="text-[10px] text-app-secondary">{t("probeRun.hookSourceOnly")}</p>
            )}
          </div>
        ) : null}
        {editorTab === "compile" ? (
          needsHookCompile ? (
            <div className="flex-1 min-h-0 flex flex-col gap-1">
              {!connected ? <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.needConnect")}</p> : null}
              {hookValidateBusy ? <p className="text-[10px] text-app-secondary shrink-0">{t("common.ellipsis")}</p> : null}
              {hookValidate ? (
                <>
                  <p
                    className={`text-[10px] shrink-0 ${hookValidate.ok ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"}`}
                  >
                    {hookValidate.ok
                      ? t("probeRun.compile.ok")
                      : hookValidate.error_message || t("probeRun.compile.fail")}
                  </p>
                  {hookValidate.diagnostics && hookValidate.diagnostics.length > 0 ? (
                    <ul className="text-[9px] text-rose-600 dark:text-rose-400 list-disc pl-4 shrink-0 max-h-[80px] overflow-auto m-0">
                      {hookValidate.diagnostics.map((d, i) => (
                        <li key={i}>{d.message}</li>
                      ))}
                    </ul>
                  ) : null}
                  <pre className="flex-1 min-h-0 overflow-auto rounded-md border border-app-separator bg-app-bg p-2 font-mono-tight text-[9px] text-app-secondary whitespace-pre-wrap break-all">
                    {(hookValidate.compiler_output || "").trim() || "—"}
                  </pre>
                </>
              ) : connected ? (
                <p className="text-[10px] text-app-secondary shrink-0">{t("probeRun.breakCompileWaiting")}</p>
              ) : null}
            </div>
          ) : (
            <p className="text-[10px] text-app-secondary">{t("probeRun.compileHookOnly")}</p>
          )
        ) : null}
      </div>

      <div className="flex flex-wrap gap-1 shrink-0 pt-1 border-t border-app-separator">
        <button type="button" className="btn-app text-xs" disabled={runDisabled} onClick={() => void onRun()}>
          {draft.kind === "hook" ? t("probeRun.compileAttachRun") : t("probeRun.run")}
        </button>
        <button
          type="button"
          className="btn-app text-xs"
          disabled={!cmdLine.trim() || draft.kind === "hook"}
          onClick={onFillConsole}
        >
          {t("probeRun.toConsole")}
        </button>
      </div>
      {runOut ? <p className="text-[10px] text-app-secondary shrink-0 truncate">{runOut}</p> : null}
    </div>
  );
}
