import { useVirtualizer } from "@tanstack/react-virtual";
import { listen } from "@tauri-apps/api/event";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import * as api from "./api";
import { HookEditorPanel } from "./components/HookEditorPanel";
import { SessionProbesPanel } from "./components/SessionProbesPanel";
import { GlossaryTip } from "./procGlossary";

const MAX_EVENTS = 8000;
const METRICS_POLL_MS = 2000;

export type DebugEventPayload = {
  timestamp_ns: number;
  session_id: string;
  event_type: number;
  event_type_name: string;
  pid: number;
  tgid: number;
  cpu: number;
  probe_id: string;
  payload_hex: string;
  payload_truncated: boolean;
  payload_utf8: string;
};

type CpuJ = {
  label: string;
  user: number;
  nice: number;
  system: number;
  idle: number;
  iowait: number;
  irq: number;
  softirq: number;
  steal: number;
  guest: number;
  guest_nice: number;
};

type NetDev = {
  name: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
  rx_dropped: number;
  tx_dropped: number;
};

type TaskRow = {
  tid: number;
  name: string;
  state: string;
  vm_peak_kb: number;
  vm_size_kb: number;
  vm_rss_kb: number;
  vm_hwm_kb: number;
  threads_count: number;
};

function hexLines(hex: string, bytesPerLine = 16): string[] {
  const lines: string[] = [];
  for (let i = 0; i < hex.length; i += bytesPerLine * 2) {
    const chunk = hex.slice(i, i + bytesPerLine * 2);
    const parts: string[] = [];
    for (let j = 0; j < chunk.length; j += 2) {
      parts.push(chunk.slice(j, j + 2));
    }
    const addr = (i / 2).toString(16).padStart(8, "0");
    lines.push(`${addr}  ${parts.join(" ")}`);
  }
  return lines;
}

function eventMatchesFilter(ev: DebugEventPayload, q: string): boolean {
  if (!q.trim()) return true;
  const tokens = q.toLowerCase().trim().split(/\s+/);
  const hay = [
    ev.event_type_name,
    String(ev.event_type),
    String(ev.pid),
    String(ev.tgid),
    String(ev.cpu),
    ev.probe_id,
    ev.payload_utf8,
    ev.session_id,
  ]
    .join(" ")
    .toLowerCase();
  return tokens.every((tok) => hay.includes(tok));
}

export default function App() {
  const { t, i18n } = useTranslation();

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
        console.warn("stream error", e.payload);
        setCapturing(false);
        bump();
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
    const id = setInterval(poll, METRICS_POLL_MS);
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

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "e") {
        e.preventDefault();
        exportJsonl();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [exportJsonl]);

  const onConnect = async () => {
    try {
      const sid = await api.connectAgent(agent.trim(), token);
      setSessionId(sid);
      setConnected(true);
    } catch (e) {
      alert(String(e));
    }
  };

  const onDisconnect = async () => {
    try {
      await api.stopCapture();
    } catch {
      /* */
    }
    setCapturing(false);
    try {
      await api.disconnectAgent();
    } catch (e) {
      alert(String(e));
    }
    setConnected(false);
    setSessionId(null);
  };

  const onStartCap = async () => {
    try {
      await api.startCapture();
      setCapturing(true);
    } catch (e) {
      alert(String(e));
    }
  };

  const onStopCap = async () => {
    try {
      await api.stopCapture();
    } catch (e) {
      alert(String(e));
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
      setCmdOut(
        r.ok ? r.output || t("common.emptyOutput") : r.error_message || r.output || t("common.errorOutput"),
      );
    } catch (e) {
      setCmdOut(String(e));
    }
  };

  const cpus = (metrics?.cpus as CpuJ[] | undefined) ?? [];
  const netDevs = (metrics?.net_devs as NetDev[] | undefined) ?? [];
  const hostname = String(metrics?.hostname ?? t("common.dash"));

  const discTabLabel = (k: "tp" | "kp" | "up") =>
    k === "tp" ? t("discover.tracepoint") : k === "kp" ? t("discover.kprobe") : t("discover.uprobe");

  const breadcrumbExtra =
    (selNic ? ` › ${t("breadcrumb.nic", { name: selNic })}` : "") +
    (dimension === "threads" && tgidStr ? ` › ${t("breadcrumb.tgid", { id: tgidStr })}` : "");

  return (
    <div className="h-full flex flex-col bg-shell-bg text-gray-200 text-sm">
      <header className="flex flex-wrap items-center gap-2 px-2 py-1.5 border-b border-shell-border bg-shell-panel shrink-0">
        <span className="font-semibold text-shell-accent mr-2">{t("header.brand")}</span>
        <label className="flex items-center gap-1">
          {t("header.agent")}
          <input
            className="bg-black/40 border border-shell-border rounded px-1 py-0.5 w-44 font-mono-tight"
            value={agent}
            disabled={connected}
            onChange={(e) => setAgent(e.target.value)}
          />
        </label>
        <label className="flex items-center gap-1">
          {t("header.token")}
          <input
            type="password"
            className="bg-black/40 border border-shell-border rounded px-1 py-0.5 w-28 font-mono-tight"
            value={token}
            disabled={connected}
            onChange={(e) => setToken(e.target.value)}
          />
        </label>
        {!connected ? (
          <button
            type="button"
            className="px-2 py-0.5 rounded bg-blue-900/80 hover:bg-blue-800 border border-shell-border"
            onClick={onConnect}
          >
            {t("header.connect")}
          </button>
        ) : (
          <button
            type="button"
            className="px-2 py-0.5 rounded bg-red-900/50 hover:bg-red-800/60 border border-shell-border"
            onClick={onDisconnect}
          >
            {t("header.disconnect")}
          </button>
        )}
        <span className="text-shell-muted">|</span>
        {!capturing ? (
          <button
            type="button"
            disabled={!connected}
            className="px-2 py-0.5 rounded bg-emerald-900/60 hover:bg-emerald-800/80 border border-shell-border disabled:opacity-40"
            onClick={onStartCap}
          >
            {t("header.startCapture")}
          </button>
        ) : (
          <button
            type="button"
            className="px-2 py-0.5 rounded bg-amber-900/50 hover:bg-amber-800/60 border border-shell-border"
            onClick={onStopCap}
          >
            {t("header.stopCapture")}
          </button>
        )}
        <label className="flex items-center gap-1 flex-1 min-w-[12rem]">
          {t("header.displayFilter")}
          <input
            className="flex-1 bg-black/40 border border-shell-border rounded px-1 py-0.5 font-mono-tight"
            placeholder={t("header.filterPlaceholder")}
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              setSelFi(null);
            }}
          />
        </label>
        <button
          type="button"
          className="px-2 py-0.5 rounded border border-shell-border hover:bg-white/5"
          onClick={() => {
            eventsRef.current = [];
            setSelFi(null);
            bump();
          }}
        >
          {t("header.clearEvents")}
        </button>
        <button
          type="button"
          className="px-2 py-0.5 rounded border border-shell-border hover:bg-white/5"
          title={t("header.exportTitle")}
          onClick={exportJsonl}
        >
          {t("header.exportJsonl")}
        </button>
        <select
          className="bg-black/40 border border-shell-border rounded px-1 py-0.5 text-xs"
          value={i18n.language.startsWith("zh") ? "zh" : "en"}
          onChange={(e) => void i18n.changeLanguage(e.target.value)}
          aria-label="Language"
        >
          <option value="zh">{t("header.langZh")}</option>
          <option value="en">{t("header.langEn")}</option>
        </select>
      </header>

      <div className="flex-1 min-h-0 flex">
        <PanelGroup direction="horizontal" className="flex-1">
          <Panel defaultSize={28} minSize={18} className="min-w-0 flex flex-col border-r border-shell-border bg-shell-panel">
            <div className="px-2 py-1 border-b border-shell-border text-xs text-shell-muted">
              {t("breadcrumb.prefix")}
              {hostname}
              {breadcrumbExtra}
            </div>
            <div className="flex gap-1 px-2 py-1 border-b border-shell-border">
              {(["host", "nic", "threads"] as const).map((d) => (
                <button
                  key={d}
                  type="button"
                  className={`px-2 py-0.5 rounded text-xs border ${
                    dimension === d ? "bg-shell-accent/20 border-shell-accent" : "border-transparent hover:bg-white/5"
                  }`}
                  onClick={() => setDimension(d)}
                >
                  {t(`dimension.${d}`)}
                </button>
              ))}
            </div>
            <div className="shrink-0 max-h-[34vh] overflow-auto p-2 text-xs space-y-2">
              {!connected && <p className="text-shell-muted">{t("metrics.hintDisconnected")}</p>}
              {connected && dimension === "host" && metrics && (
                <div className="space-y-2">
                  {(metrics.error_message as string) && (
                    <p className="text-amber-400">{(metrics.error_message as string) || ""}</p>
                  )}
                  <div className="grid grid-cols-2 gap-2">
                    <div className="bg-black/30 rounded p-2 border border-shell-border">
                      <GlossaryTip term="loadavg_one" labelKey="metrics.load1m" /> {Number(metrics.loadavg_one).toFixed(2)}
                    </div>
                    <div className="bg-black/30 rounded p-2 border border-shell-border">
                      <GlossaryTip term="loadavg_five" labelKey="metrics.load5m" /> {Number(metrics.loadavg_five).toFixed(2)}
                    </div>
                    <div className="bg-black/30 rounded p-2 border border-shell-border">
                      <GlossaryTip term="mem_total_kb" labelKey="metrics.memTotal" />{" "}
                      {(Number(metrics.mem_total_kb) / 1024 / 1024).toFixed(1)} GB
                    </div>
                    <div className="bg-black/30 rounded p-2 border border-shell-border">
                      <GlossaryTip term="mem_available_kb" labelKey="metrics.memAvail" />{" "}
                      {(Number(metrics.mem_available_kb) / 1024 / 1024).toFixed(1)} GB
                    </div>
                  </div>
                  {!!metrics.mem_total_kb && (
                    <div>
                      <div className="flex justify-between text-[10px] text-shell-muted mb-0.5">
                        <span>{t("metrics.memBarCaption")}</span>
                      </div>
                      <div className="h-2 bg-black/40 rounded overflow-hidden border border-shell-border">
                        <div
                          className="h-full bg-blue-600/70"
                          style={{
                            width: `${Math.min(
                              100,
                              (100 * (Number(metrics.mem_total_kb) - Number(metrics.mem_available_kb))) /
                                Number(metrics.mem_total_kb),
                            )}%`,
                          }}
                        />
                      </div>
                    </div>
                  )}
                  <div className="font-mono-tight text-[10px] overflow-x-auto">
                    <div className="text-shell-muted mb-1">{t("metrics.statTableCaption")}</div>
                    <table className="w-full border-collapse">
                      <thead>
                        <tr className="text-left text-shell-muted">
                          <th className="pr-2">{t("metrics.cpuCol")}</th>
                          <th>
                            <GlossaryTip term="cpu_user" labelKey="metrics.colUser" />
                          </th>
                          <th>
                            <GlossaryTip term="cpu_system" labelKey="metrics.colSys" />
                          </th>
                          <th>
                            <GlossaryTip term="cpu_idle" labelKey="metrics.colIdle" />
                          </th>
                          <th>
                            <GlossaryTip term="cpu_iowait" labelKey="metrics.colIow" />
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {cpus.slice(0, 9).map((c) => (
                          <tr key={c.label} className="border-t border-shell-border/50">
                            <td className="pr-2">{c.label}</td>
                            <td>{c.user}</td>
                            <td>{c.system}</td>
                            <td>{c.idle}</td>
                            <td>{c.iowait}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {cpus.length > 9 && (
                      <div className="text-shell-muted mt-1">{t("metrics.rowsTotal", { count: cpus.length })}</div>
                    )}
                  </div>
                </div>
              )}
              {connected && dimension === "nic" && metrics && (
                <div>
                  <p className="text-shell-muted mb-2">{t("metrics.nicHint")}</p>
                  <ul className="space-y-1 font-mono-tight">
                    {netDevs.map((n) => (
                      <li key={n.name}>
                        <button
                          type="button"
                          className={`w-full text-left px-1 rounded ${
                            selNic === n.name ? "bg-shell-accent/25" : "hover:bg-white/5"
                          }`}
                          onClick={() => setSelNic(n.name)}
                        >
                          <span className="font-semibold">{n.name}</span>{" "}
                          <span className="text-shell-muted">
                            {t("metrics.rxLabel")}{" "}
                            <GlossaryTip term="rx_bytes" labelKey="metrics.byteAbbr" /> {n.rx_bytes} · {t("metrics.txLabel")}{" "}
                            <GlossaryTip term="tx_bytes" labelKey="metrics.byteAbbr" /> {n.tx_bytes}
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {connected && dimension === "threads" && (
                <div className="space-y-2">
                  <div className="flex gap-1 items-center">
                    <GlossaryTip term="tgid" labelKey="tasks.colTgid" />
                    <input
                      className="bg-black/40 border border-shell-border rounded px-1 w-24 font-mono-tight"
                      value={tgidStr}
                      onChange={(e) => setTgidStr(e.target.value)}
                    />
                    <button
                      type="button"
                      className="px-2 py-0.5 rounded border border-shell-border hover:bg-white/5"
                      onClick={loadTasks}
                    >
                      {t("tasks.loadTask")}
                    </button>
                  </div>
                  {taskErr && <p className="text-amber-400">{taskErr}</p>}
                  <table className="w-full font-mono-tight text-[10px] border-collapse">
                    <thead>
                      <tr className="text-shell-muted text-left">
                        <th>
                          <GlossaryTip term="tid" labelKey="tasks.colTid" />
                        </th>
                        <th>{t("tasks.colName")}</th>
                        <th>
                          <GlossaryTip term="vm_rss_kb" labelKey="tasks.colRss" />
                        </th>
                        <th />
                      </tr>
                    </thead>
                    <tbody>
                      {tasks.map((task) => (
                        <tr key={task.tid} className="border-t border-shell-border/40">
                          <td className="pr-1">{task.tid}</td>
                          <td className="truncate max-w-[80px]" title={task.name}>
                            {task.name}
                          </td>
                          <td>{task.vm_rss_kb}</td>
                          <td>
                            <button
                              type="button"
                              className="text-shell-accent hover:underline"
                              onClick={() => {
                                setFilter(String(tgidStr));
                              }}
                            >
                              {t("tasks.filter")}
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            <div className="flex-1 min-h-0 flex flex-col overflow-hidden border-t border-shell-border">
            <div className="border-b border-shell-border p-2 shrink-0 flex flex-col max-h-[30vh] min-h-[120px]">
              <div className="flex gap-1 mb-1">
                {(["tp", "kp", "up"] as const).map((k) => (
                  <button
                    key={k}
                    type="button"
                    className={`px-2 py-0.5 rounded text-xs border ${
                      discTab === k ? "bg-white/10 border-shell-border" : "border-transparent hover:bg-white/5"
                    }`}
                    onClick={() => setDiscTab(k)}
                  >
                    {discTabLabel(k)}
                  </button>
                ))}
              </div>
              <div className="flex gap-1 mb-1">
                <input
                  className="flex-1 bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
                  value={discPrefix}
                  onChange={(e) => setDiscPrefix(e.target.value)}
                  placeholder={t("discover.prefixPh")}
                />
                {discTab === "up" && (
                  <input
                    className="flex-1 bg-black/40 border border-shell-border rounded px-1 text-xs font-mono-tight"
                    value={discBin}
                    onChange={(e) => setDiscBin(e.target.value)}
                    placeholder={t("discover.binaryPh")}
                  />
                )}
                <button
                  type="button"
                  disabled={!connected}
                  className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
                  onClick={runDiscover}
                >
                  {t("discover.list")}
                </button>
              </div>
              <ul className="flex-1 overflow-auto font-mono-tight text-[10px] text-gray-400 space-y-0.5">
                {discLines.map((line, i) => (
                  <li
                    key={`${i}-${line.slice(0, 12)}`}
                    className="cursor-pointer hover:text-shell-accent truncate"
                    title={line}
                    onClick={() => navigator.clipboard.writeText(line)}
                  >
                    {line}
                  </li>
                ))}
              </ul>
            </div>

            <SessionProbesPanel connected={connected} refreshTrigger={probeRefresh} />

            <div className="border-t border-shell-border p-2 space-y-1 shrink-0">
              <div className="text-xs text-shell-muted">{t("cmd.title")}</div>
              <div className="flex gap-1">
                <input
                  className="flex-1 bg-black/40 border border-shell-border rounded px-1 font-mono-tight text-xs"
                  value={cmd}
                  onChange={(e) => setCmd(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && runCmd()}
                />
                <button
                  type="button"
                  disabled={!connected}
                  className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
                  onClick={runCmd}
                >
                  {t("cmd.run")}
                </button>
              </div>
              <pre className="text-[10px] bg-black/30 rounded p-1 max-h-24 overflow-auto whitespace-pre-wrap">
                {cmdOut}
              </pre>
            </div>

            <HookEditorPanel
              connected={connected}
              onProbesChanged={() => setProbeRefresh((n) => n + 1)}
            />
            </div>
          </Panel>

          <PanelResizeHandle className="w-1 bg-shell-border hover:bg-shell-accent/40 transition-colors" />

          <Panel minSize={40} className="min-w-0 flex flex-col">
            <PanelGroup direction="vertical" className="flex-1 min-h-0">
              <Panel defaultSize={52} minSize={25} className="min-h-0 flex flex-col border-b border-shell-border">
                <div className="px-2 py-0.5 text-xs text-shell-muted border-b border-shell-border shrink-0">
                  {t("events.panelTitle", { filtered: filtered.length, total: eventCount })}
                </div>
                <div ref={parentRef} className="flex-1 overflow-auto font-mono-tight">
                  <div
                    style={{
                      height: `${rowVirtualizer.getTotalSize()}px`,
                      width: "100%",
                      position: "relative",
                    }}
                  >
                    {rowVirtualizer.getVirtualItems().map((vi) => {
                      const ev = filtered[vi.index];
                      const sel = selFi === vi.index;
                      return (
                        <div
                          key={vi.key}
                          className={`absolute top-0 left-0 w-full flex cursor-pointer border-b border-shell-border/30 text-[11px] ${
                            sel ? "bg-shell-accent/15" : "hover:bg-white/5"
                          }`}
                          style={{ height: `${vi.size}px`, transform: `translateY(${vi.start}px)` }}
                          onClick={() => setSelFi(vi.index)}
                        >
                          <span className="w-8 shrink-0 text-shell-muted px-1">{vi.index}</span>
                          <span className="w-24 shrink-0 truncate">{relTimeNs(firstTs, ev.timestamp_ns)}</span>
                          <span className="w-28 shrink-0 truncate text-amber-200/90">{ev.event_type_name}</span>
                          <span className="w-12 shrink-0">{ev.pid}</span>
                          <span className="w-12 shrink-0">{ev.tgid}</span>
                          <span className="w-8 shrink-0">{ev.cpu}</span>
                          <span className="w-16 shrink-0 truncate">{ev.probe_id}</span>
                          <span className="flex-1 truncate text-gray-500">{ev.payload_utf8.replace(/\s+/g, " ")}</span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </Panel>

              <PanelResizeHandle className="h-1 bg-shell-border hover:bg-shell-accent/40" />

              <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col border-b border-shell-border">
                <div className="px-2 py-0.5 text-xs text-shell-muted shrink-0">{t("detail.jsonTitle")}</div>
                <pre className="flex-1 overflow-auto p-2 text-[11px] font-mono-tight bg-black/20">
                  {selected ? JSON.stringify(selected, null, 2) : t("detail.pickRow")}
                </pre>
              </Panel>

              <PanelResizeHandle className="h-1 bg-shell-border hover:bg-shell-accent/40" />

              <Panel defaultSize={24} minSize={12} className="min-h-0 flex flex-col">
                <div className="px-2 py-0.5 text-xs text-shell-muted shrink-0">
                  {t("hex.title")}
                  {selected?.payload_truncated ? t("hex.truncated") : ""}
                </div>
                <pre className="flex-1 overflow-auto p-2 text-[11px] font-mono-tight bg-black/20 whitespace-pre">
                  {selected ? hexLines(selected.payload_hex).join("\n") : t("common.dash")}
                </pre>
              </Panel>
            </PanelGroup>
          </Panel>
        </PanelGroup>
      </div>

      <footer className="shrink-0 px-2 py-1 border-t border-shell-border text-[11px] text-shell-muted flex flex-wrap gap-3 bg-shell-panel">
        <span>
          {t("footer.session")} {sessionId ?? t("common.dash")}
        </span>
        <span>
          {t("footer.connected")} {connected ? t("common.yes") : t("common.no")}
        </span>
        <span>
          {t("footer.capture")} {capturing ? t("footer.capturing") : t("footer.stopped")}
        </span>
        <span>
          {t("footer.metricsRefresh")} {metricsAt ?? t("common.dash")}
        </span>
        <span className="ml-auto">{t("footer.shortcutExport")}</span>
      </footer>
    </div>
  );
}
