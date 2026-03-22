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

import { invoke } from "@tauri-apps/api/core";

export function connectAgent(agent: string, token: string) {
  return invoke<string>("connect_agent", { agent, token });
}

export function disconnectAgent() {
  return invoke<void>("disconnect_agent");
}

export function startCapture() {
  return invoke<void>("start_capture");
}

export function stopCapture() {
  return invoke<void>("stop_capture");
}

export function fetchHostMetrics() {
  return invoke<Record<string, unknown>>("fetch_host_metrics");
}

export function fetchTaskTree(tgid: number) {
  return invoke<Record<string, unknown>>("fetch_task_tree", { tgid });
}

/** Runs a REPL line via the agent. Rejects when the command logically fails (`ok: false`) or on transport errors. */
export function executeCmd(commandLine: string) {
  return invoke<{ ok: boolean; output: string; error_message: string }>("execute_cmd", {
    commandLine,
  });
}

export function listTracepoints(prefix: string, maxEntries: number) {
  return invoke<string[]>("list_tracepoints_cmd", { prefix, maxEntries });
}

export function listKprobes(prefix: string, maxEntries: number) {
  return invoke<string[]>("list_kprobes_cmd", { prefix, maxEntries });
}

export function listUprobes(binaryPath: string, prefix: string, maxEntries: number) {
  return invoke<string[]>("list_uprobes_cmd", {
    binaryPath,
    prefix,
    maxEntries,
  });
}

export type CompileDiagnostic = {
  path: string;
  line: number;
  column: number;
  severity: string;
  message: string;
};

export type CompileHookResult = {
  ok: boolean;
  error_message: string;
  hook_id: string;
  probe_point: string;
  diagnostics: CompileDiagnostic[];
  compiler_output: string;
};

export function compileHook(source: string, programName: string, limit = 0) {
  return invoke<CompileHookResult>("compile_hook", { source, programName, limit });
}

export type ValidateCompileSourceResult = {
  ok: boolean;
  error_message: string;
  diagnostics: CompileDiagnostic[];
  compiler_output: string;
};

export function validateCompileSource(source: string) {
  return invoke<ValidateCompileSourceResult>("validate_compile_source", { source });
}

export type HookMapDescriptor = {
  name: string;
  map_type: string;
  key_size: number;
  value_size: number;
  max_entries: number;
};

export type ListHookMapsResult = {
  ok: boolean;
  error_message: string;
  maps: HookMapDescriptor[];
};

export function listHookMaps(hookId: string) {
  return invoke<ListHookMapsResult>("list_hook_maps_cmd", { hookId });
}

export type MapEntryHex = {
  key_hex: string;
  value_hex: string;
};

export type ReadHookMapResult = {
  ok: boolean;
  error_message: string;
  entries: MapEntryHex[];
};

export function readHookMap(hookId: string, mapName: string, maxEntries: number) {
  return invoke<ReadHookMapResult>("read_hook_map_cmd", { hookId, mapName, maxEntries });
}
