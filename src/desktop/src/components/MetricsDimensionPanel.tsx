import type { TFunction } from "i18next";
import { GlossaryTip } from "../procGlossary";
import type { CpuJ, NetDev, TaskRow } from "../app/types";

type Dim = "host" | "nic" | "threads";

type Props = {
  t: TFunction;
  hostname: string;
  breadcrumbExtra: string;
  connected: boolean;
  metrics: Record<string, unknown> | null;
  dimension: Dim;
  setDimension: (d: Dim) => void;
  cpus: CpuJ[];
  netDevs: NetDev[];
  selNic: string | null;
  setSelNic: (n: string) => void;
  tgidStr: string;
  setTgidStr: (s: string) => void;
  tasks: TaskRow[];
  taskErr: string;
  loadTasks: () => void;
  setFilter: (s: string) => void;
};

export function MetricsDimensionPanel({
  t,
  hostname,
  breadcrumbExtra,
  connected,
  metrics,
  dimension,
  setDimension,
  cpus,
  netDevs,
  selNic,
  setSelNic,
  tgidStr,
  setTgidStr,
  tasks,
  taskErr,
  loadTasks,
  setFilter,
}: Props) {
  return (
    <>
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
                      {t("metrics.rxLabel")} <GlossaryTip term="rx_bytes" labelKey="metrics.byteAbbr" /> {n.rx_bytes} ·{" "}
                      {t("metrics.txLabel")} <GlossaryTip term="tx_bytes" labelKey="metrics.byteAbbr" /> {n.tx_bytes}
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
    </>
  );
}
