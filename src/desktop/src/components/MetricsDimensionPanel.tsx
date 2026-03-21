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
    <div className="flex flex-col flex-1 min-h-0 h-full">
      <div className="shrink-0 px-3 py-2 border-b border-app-separator text-xs text-app-secondary">
        {t("breadcrumb.prefix")}
        {hostname}
        {breadcrumbExtra}
      </div>
      <div className="shrink-0 flex gap-1 px-3 py-2 border-b border-app-separator" role="tablist" aria-label={t("dimension.aria")}>
        {(["host", "nic", "threads"] as const).map((d) => (
          <button
            key={d}
            type="button"
            role="tab"
            aria-selected={dimension === d}
            className={`rounded-md px-2 py-1 text-xs border transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-app-accent ${
              dimension === d ? "bg-app-accent-muted border-app-accent text-app-label" : "border-transparent text-app-secondary hover:bg-app-hover"
            }`}
            onClick={() => setDimension(d)}
          >
            {t(`dimension.${d}`)}
          </button>
        ))}
      </div>
      <div className="flex-1 min-h-0 overflow-auto p-3 text-xs text-app-label space-y-2">
        {!connected && <p className="text-app-secondary">{t("metrics.hintDisconnected")}</p>}
        {connected && dimension === "host" && metrics && (
          <div className="space-y-2">
            {(metrics.error_message as string) && (
              <p className="text-amber-400">{(metrics.error_message as string) || ""}</p>
            )}
            <div className="grid grid-cols-2 gap-2">
              <div className="rounded-md border border-app-separator bg-app-field p-2">
                <GlossaryTip term="loadavg_one" labelKey="metrics.load1m" /> {Number(metrics.loadavg_one).toFixed(2)}
              </div>
              <div className="rounded-md border border-app-separator bg-app-field p-2">
                <GlossaryTip term="loadavg_five" labelKey="metrics.load5m" /> {Number(metrics.loadavg_five).toFixed(2)}
              </div>
              <div className="rounded-md border border-app-separator bg-app-field p-2">
                <GlossaryTip term="mem_total_kb" labelKey="metrics.memTotal" />{" "}
                {(Number(metrics.mem_total_kb) / 1024 / 1024).toFixed(1)} GB
              </div>
              <div className="rounded-md border border-app-separator bg-app-field p-2">
                <GlossaryTip term="mem_available_kb" labelKey="metrics.memAvail" />{" "}
                {(Number(metrics.mem_available_kb) / 1024 / 1024).toFixed(1)} GB
              </div>
            </div>
            {!!metrics.mem_total_kb && (
              <div>
                <div className="flex justify-between text-[10px] text-app-secondary mb-0.5">
                  <span>{t("metrics.memBarCaption")}</span>
                </div>
                <div className="h-2 rounded overflow-hidden border border-app-separator bg-app-field">
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
              <div className="text-app-secondary mb-1">{t("metrics.statTableCaption")}</div>
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left text-app-secondary">
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
                    <tr key={c.label} className="border-t border-app-separator/60">
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
                <div className="text-app-secondary mt-1">{t("metrics.rowsTotal", { count: cpus.length })}</div>
              )}
            </div>
          </div>
        )}
        {connected && dimension === "nic" && metrics && (
          <div>
            <p className="text-app-secondary mb-2">{t("metrics.nicHint")}</p>
            <ul className="space-y-1 font-mono-tight">
              {netDevs.map((n) => (
                <li key={n.name}>
                  <button
                    type="button"
                    className={`w-full text-left px-2 py-0.5 rounded-md ${
                      selNic === n.name ? "bg-app-accent-muted" : "hover:bg-app-hover"
                    }`}
                    onClick={() => setSelNic(n.name)}
                  >
                    <span className="font-semibold">{n.name}</span>{" "}
                    <span className="text-app-secondary">
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
                className="input-app w-24 font-mono-tight py-1"
                value={tgidStr}
                onChange={(e) => setTgidStr(e.target.value)}
              />
              <button
                type="button"
                className="btn-app text-xs"
                onClick={loadTasks}
              >
                {t("tasks.loadTask")}
              </button>
            </div>
            {taskErr && <p className="text-amber-400">{taskErr}</p>}
            <table className="w-full font-mono-tight text-[10px] border-collapse">
              <thead>
                <tr className="text-app-secondary text-left">
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
                  <tr key={task.tid} className="border-t border-app-separator/50">
                    <td className="pr-1">{task.tid}</td>
                    <td className="truncate max-w-[80px]" title={task.name}>
                      {task.name}
                    </td>
                    <td>{task.vm_rss_kb}</td>
                    <td>
                      <button
                        type="button"
                        className="text-app-accent hover:underline"
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
    </div>
  );
}
