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

import { useVirtualizer } from "@tanstack/react-virtual";
import { listen } from "@tauri-apps/api/event";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { eventMatchesFilter, MAX_EVENTS } from "./app/eventUtils";
import type { CpuJ, DebugEventPayload, NetDev, TaskRow } from "./app/types";
import * as api from "./api";
import { AboutDialog } from "./components/AboutDialog";
import { AppFooter } from "./components/AppFooter";
import { AppHeader } from "./components/AppHeader";
import { CmdReplBlock } from "./components/CmdReplBlock";
import { DiscoverPanel } from "./components/DiscoverPanel";
import { EventsStreamPanel } from "./components/EventsStreamPanel";
import { HookEditorPanel } from "./components/HookEditorPanel";
import { InlineErrorBanner } from "./components/InlineErrorBanner";
import { MetricsDimensionPanel } from "./components/MetricsDimensionPanel";
import { SessionProbesPanel } from "./components/SessionProbesPanel";
import { SettingsDialog } from "./components/SettingsDialog";
import { usePhantomMenu } from "./hooks/usePhantomMenu";
import { AppShell } from "./layout/AppShell";

export type { DebugEventPayload } from "./app/types";

export default function App() {
  const { t } = useTranslation();

  function relTimeNs(first: number, cur: number): string {
    const d = (cur - first) / 1e6;
    if (Number.isNaN(d)) return t("common.dash");
    return `${d.toFixed(3)} ms`;
  }

  const [agent, setAgent] = useState("127.0.0.1:9090");
  const [token, setToken] = useState("");
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [capturing, setCapturing] = useState(false);
  const [filter, setFilter] = useState("");
  const [selFi, setSelFi] = useState<number | null>(null);
  const [appError, setAppError] = useState("");

  const eventsRef = useRef<DebugEventPayload[]>([]);
  const [tick, setTick] = useState(0);
  const bump = useCallback(() => setTick((x) => x + 1), []);

  const [metrics, setMetrics] = useState<Record<string, unknown> | null>(null);
  const [metricsAt, setMetricsAt] = useState<string | null>(null);
  const [dimension, setDimension] = useState<"host" | "nic" | "threads">("host");
  const [selNic, setSelNic] = useState<string | null>(null);
  const [tgidStr, setTgidStr] = useState("1");
  const [tasks, setTasks] = useState<TaskRow[]>([]);
  const [taskErr, setTaskErr] = useState("");

  const [discTab, setDiscTab] = useState<"tp" | "kp" | "up">("tp");
  const [discPrefix, setDiscPrefix] = useState("sched");
  const [discBin, setDiscBin] = useState("/bin/sh");
  const [discLines, setDiscLines] = useState<string[]>([]);

  const [cmd, setCmd] = useState("help");
  const [cmdOut, setCmdOut] = useState("");
  const [probeRefresh, setProbeRefresh] = useState(0);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);

  const parentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let alive = true;
    const pending: (() => void)[] = [];
    (async () => {
      const u1 = await listen<DebugEventPayload>("debug-event", (e) => {
        const arr = eventsRef.current;
        arr.push(e.payload);
        if (arr.length > MAX_EVENTS) {
          arr.splice(0, arr.length - MAX_EVENTS);
        }
        bump();
      });
      if (alive) pending.push(u1);
      else u1();
      const u2 = await listen<{ message: string }>("debug-event-error", (e) => {
        console.warn("stream error (may reconnect)", e.payload);
      });
      if (alive) pending.push(u2);
      else u2();
    })();
    return () => {
      alive = false;
      pending.forEach((f) => f());
    };
  }, [bump]);

  const eventCount = eventsRef.current.length;
  const filtered = useMemo(
    () => eventsRef.current.filter((ev) => eventMatchesFilter(ev, filter)),
    [tick, filter],
  );

  const rowVirtualizer = useVirtualizer({
    count: filtered.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 26,
    overscan: 12,
  });

  const firstTs = filtered[0]?.timestamp_ns ?? eventsRef.current[0]?.timestamp_ns ?? 0;
  const selected = selFi !== null && selFi >= 0 && selFi < filtered.length ? filtered[selFi] : null;

  useEffect(() => {
    if (!connected) {
      setMetrics(null);
      return;
    }
    let cancelled = false;
    const poll = async () => {
      try {
        const m = await api.fetchHostMetrics();
        if (!cancelled) {
          setMetrics(m);
          setMetricsAt(new Date().toLocaleTimeString());
        }
      } catch (e) {
        if (!cancelled) setMetrics({ error_message: String(e) });
      }
    };
    poll();
    const id = setInterval(poll, 2000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [connected]);

  const exportJsonl = useCallback(() => {
    const blob = new Blob([eventsRef.current.map((x: DebugEventPayload) => JSON.stringify(x)).join("\n")], {
      type: "application/x-ndjson",
    });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `phantom-events-${Date.now()}.jsonl`;
    a.click();
    URL.revokeObjectURL(a.href);
  }, []);

  const clearEvents = useCallback(() => {
    eventsRef.current = [];
    setSelFi(null);
    bump();
  }, [bump]);

  usePhantomMenu({
    exportJsonl,
    clearEvents,
    onOpenSettings: () => setSettingsOpen(true),
    onOpenAbout: () => setAboutOpen(true),
  });

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "e") {
        e.preventDefault();
        exportJsonl();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === ",") {
        e.preventDefault();
        setSettingsOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [exportJsonl]);

  const onConnect = async () => {
    setAppError("");
    try {
      const sid = await api.connectAgent(agent.trim(), token);
      setSessionId(sid);
      setConnected(true);
    } catch (e) {
      setAppError(String(e));
    }
  };

  const onDisconnect = async () => {
    setAppError("");
    try {
      await api.stopCapture();
    } catch {
      /* */
    }
    setCapturing(false);
    try {
      await api.disconnectAgent();
    } catch (e) {
      setAppError(String(e));
    }
    setConnected(false);
    setSessionId(null);
  };

  const onStartCap = async () => {
    setAppError("");
    try {
      await api.startCapture();
      setCapturing(true);
    } catch (e) {
      setAppError(String(e));
    }
  };

  const onStopCap = async () => {
    setAppError("");
    try {
      await api.stopCapture();
    } catch (e) {
      setAppError(String(e));
    }
    setCapturing(false);
  };

  const loadTasks = async () => {
    setTaskErr("");
    const n = parseInt(tgidStr, 10);
    if (!n || n < 0) {
      setTaskErr(t("tasks.invalidTgid"));
      return;
    }
    try {
      const r = await api.fetchTaskTree(n);
      const err = (r.error_message as string) || "";
      if (err) setTaskErr(err);
      const ts = (r.tasks as TaskRow[]) ?? [];
      setTasks(ts);
    } catch (e) {
      setTaskErr(String(e));
      setTasks([]);
    }
  };

  const runDiscover = async () => {
    setDiscLines([t("common.ellipsis")]);
    try {
      if (discTab === "tp") {
        setDiscLines(await api.listTracepoints(discPrefix, 5000));
      } else if (discTab === "kp") {
        setDiscLines(await api.listKprobes(discPrefix, 5000));
      } else {
        setDiscLines(await api.listUprobes(discBin, discPrefix, 2000));
      }
    } catch (e) {
      setDiscLines([String(e)]);
    }
  };

  const runCmd = async () => {
    setCmdOut(t("common.ellipsis"));
    try {
      const r = await api.executeCmd(cmd);
      setCmdOut(r.output?.trim() ? r.output : t("common.emptyOutput"));
    } catch (e) {
      setCmdOut(String(e));
    }
  };

  const cpus = (metrics?.cpus as CpuJ[] | undefined) ?? [];
  const netDevs = (metrics?.net_devs as NetDev[] | undefined) ?? [];
  const hostname = String(metrics?.hostname ?? t("common.dash"));

  const breadcrumbExtra =
    (selNic ? ` › ${t("breadcrumb.nic", { name: selNic })}` : "") +
    (dimension === "threads" && tgidStr ? ` › ${t("breadcrumb.tgid", { id: tgidStr })}` : "");

  return (
    <div className="flex h-full flex-col bg-app-bg text-sm text-app-label">
      <AppHeader
        t={t}
        agent={agent}
        setAgent={setAgent}
        token={token}
        setToken={setToken}
        connected={connected}
        capturing={capturing}
        filter={filter}
        setFilter={setFilter}
        setSelFi={setSelFi}
        onConnect={onConnect}
        onDisconnect={onDisconnect}
        onStartCap={onStartCap}
        onStopCap={onStopCap}
      />
      <InlineErrorBanner t={t} message={appError} onDismiss={() => setAppError("")} />

      <AppShell
        t={t}
        overview={
          <MetricsDimensionPanel
            t={t}
            hostname={hostname}
            breadcrumbExtra={breadcrumbExtra}
            connected={connected}
            metrics={metrics}
            dimension={dimension}
            setDimension={setDimension}
            cpus={cpus}
            netDevs={netDevs}
            selNic={selNic}
            setSelNic={setSelNic}
            tgidStr={tgidStr}
            setTgidStr={setTgidStr}
            tasks={tasks}
            taskErr={taskErr}
            loadTasks={loadTasks}
            setFilter={setFilter}
          />
        }
        discover={
          <DiscoverPanel
            t={t}
            discTab={discTab}
            setDiscTab={setDiscTab}
            discPrefix={discPrefix}
            setDiscPrefix={setDiscPrefix}
            discBin={discBin}
            setDiscBin={setDiscBin}
            discLines={discLines}
            runDiscover={runDiscover}
            connected={connected}
          />
        }
        session={<SessionProbesPanel connected={connected} refreshTrigger={probeRefresh} />}
        repl={<CmdReplBlock t={t} cmd={cmd} setCmd={setCmd} cmdOut={cmdOut} runCmd={runCmd} connected={connected} />}
        hook={<HookEditorPanel connected={connected} onProbesChanged={() => setProbeRefresh((n) => n + 1)} />}
        events={
          <EventsStreamPanel
            t={t}
            filtered={filtered}
            eventCount={eventCount}
            parentRef={parentRef}
            rowVirtualizer={rowVirtualizer}
            firstTs={firstTs}
            selFi={selFi}
            setSelFi={setSelFi}
            relTimeNs={relTimeNs}
            selected={selected}
          />
        }
      />

      <AppFooter t={t} sessionId={sessionId} connected={connected} capturing={capturing} metricsAt={metricsAt} />
      <SettingsDialog open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </div>
  );
}
