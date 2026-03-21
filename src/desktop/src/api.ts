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
  attach_point: string;
  diagnostics: CompileDiagnostic[];
  compiler_output: string;
};

export function compileHook(source: string, attach: string, programName: string) {
  return invoke<CompileHookResult>("compile_hook", { source, attach, programName });
}
