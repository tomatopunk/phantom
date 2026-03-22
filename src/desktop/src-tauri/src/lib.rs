// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

use std::sync::Arc;
use std::time::Duration;

use phantom_client::{
    CompileAndAttachResponse, GetHostMetricsResponse, GetTaskTreeResponse, ListHookMapsResponse,
    PhantomClient, PreviewHookTemplateResponse, ReadHookMapResponse, ValidateCompileSourceResponse,
};
use serde_json::{json, Value};
use tauri::menu::{MenuBuilder, MenuItem, PredefinedMenuItem, SubmenuBuilder};
use tauri::{AppHandle, Emitter, State};
use tokio::sync::Mutex;
use tokio::task::JoinHandle;

#[derive(Clone)]
struct AppState {
    /// Unary RPCs (execute, compile, discovery, metrics). Never use for `stream_events`.
    exec_client: Arc<Mutex<Option<PhantomClient>>>,
    /// Dedicated clone for the capture loop so heavy streaming never contends with execute on the same mutex.
    stream_client: Arc<Mutex<Option<PhantomClient>>>,
    capture_task: Arc<Mutex<Option<JoinHandle<()>>>>,
}

/// Exponential backoff capped for stream reconnect (ms).
fn stream_backoff_ms(attempt: u32) -> u64 {
    const INITIAL: u64 = 400;
    const MAX: u64 = 30_000;
    let ms = INITIAL.saturating_mul(2u64.saturating_pow(attempt.min(12)));
    ms.min(MAX)
}

fn event_type_name(code: i32) -> &'static str {
    match code {
        1 => "BREAK_HIT",
        2 => "TRACE_SAMPLE",
        3 => "ERROR",
        4 => "STATE_CHANGE",
        _ => "UNSPECIFIED",
    }
}

fn host_metrics_json(r: GetHostMetricsResponse) -> Value {
    let cpus: Vec<Value> = r
        .cpus
        .iter()
        .map(|c| {
            json!({
                "label": c.label,
                "user": c.user,
                "nice": c.nice,
                "system": c.system,
                "idle": c.idle,
                "iowait": c.iowait,
                "irq": c.irq,
                "softirq": c.softirq,
                "steal": c.steal,
                "guest": c.guest,
                "guest_nice": c.guest_nice,
            })
        })
        .collect();
    let net_devs: Vec<Value> = r
        .net_devs
        .iter()
        .map(|n| {
            json!({
                "name": n.name,
                "rx_bytes": n.rx_bytes,
                "tx_bytes": n.tx_bytes,
                "rx_packets": n.rx_packets,
                "tx_packets": n.tx_packets,
                "rx_errors": n.rx_errors,
                "tx_errors": n.tx_errors,
                "rx_dropped": n.rx_dropped,
                "tx_dropped": n.tx_dropped,
            })
        })
        .collect();
    json!({
        "hostname": r.hostname,
        "loadavg_one": r.loadavg_one,
        "loadavg_five": r.loadavg_five,
        "loadavg_fifteen": r.loadavg_fifteen,
        "mem_total_kb": r.mem_total_kb,
        "mem_available_kb": r.mem_available_kb,
        "mem_buffers_kb": r.mem_buffers_kb,
        "mem_cached_kb": r.mem_cached_kb,
        "mem_swap_total_kb": r.mem_swap_total_kb,
        "mem_swap_free_kb": r.mem_swap_free_kb,
        "cpus": cpus,
        "net_devs": net_devs,
        "error_message": r.error_message,
    })
}

fn task_tree_json(r: GetTaskTreeResponse) -> Value {
    let tasks: Vec<Value> = r
        .tasks
        .iter()
        .map(|t| {
            json!({
                "tid": t.tid,
                "name": t.name,
                "state": t.state,
                "vm_peak_kb": t.vm_peak_kb,
                "vm_size_kb": t.vm_size_kb,
                "vm_rss_kb": t.vm_rss_kb,
                "vm_hwm_kb": t.vm_hwm_kb,
                "threads_count": t.threads_count,
            })
        })
        .collect();
    json!({
        "tgid": r.tgid,
        "tasks": tasks,
        "error_message": r.error_message,
    })
}

fn preview_template_json(r: PreviewHookTemplateResponse) -> Value {
    let diags: Vec<Value> = r
        .diagnostics
        .iter()
        .map(|d| {
            json!({
                "path": d.path,
                "line": d.line,
                "column": d.column,
                "severity": d.severity,
                "message": d.message,
            })
        })
        .collect();
    json!({
        "ok": r.ok,
        "error_message": r.error_message,
        "generated_source_c": r.generated_source_c,
        "compile_attempted": r.compile_attempted,
        "compile_ok": r.compile_ok,
        "compiler_output": r.compiler_output,
        "diagnostics": diags,
    })
}

fn validate_source_json(r: ValidateCompileSourceResponse) -> Value {
    let diags: Vec<Value> = r
        .diagnostics
        .iter()
        .map(|d| {
            json!({
                "path": d.path,
                "line": d.line,
                "column": d.column,
                "severity": d.severity,
                "message": d.message,
            })
        })
        .collect();
    json!({
        "ok": r.ok,
        "error_message": r.error_message,
        "diagnostics": diags,
        "compiler_output": r.compiler_output,
    })
}

fn list_hook_maps_json(r: ListHookMapsResponse) -> Value {
    let maps: Vec<Value> = r
        .maps
        .iter()
        .map(|m| {
            json!({
                "name": m.name,
                "map_type": m.map_type,
                "key_size": m.key_size,
                "value_size": m.value_size,
                "max_entries": m.max_entries,
            })
        })
        .collect();
    json!({ "ok": r.ok, "error_message": r.error_message, "maps": maps })
}

fn bytes_hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{:02x}", x)).collect()
}

fn read_hook_map_json(r: ReadHookMapResponse) -> Value {
    let entries: Vec<Value> = r
        .entries
        .iter()
        .map(|e| json!({ "key_hex": bytes_hex(&e.key), "value_hex": bytes_hex(&e.value) }))
        .collect();
    json!({ "ok": r.ok, "error_message": r.error_message, "entries": entries })
}

fn compile_json(r: CompileAndAttachResponse) -> Value {
    let diags: Vec<Value> = r
        .diagnostics
        .iter()
        .map(|d| {
            json!({
                "path": d.path,
                "line": d.line,
                "column": d.column,
                "severity": d.severity,
                "message": d.message,
            })
        })
        .collect();
    json!({
        "ok": r.ok,
        "error_message": r.error_message,
        "hook_id": r.hook_id,
        "attach_point": r.attach_point,
        "diagnostics": diags,
        "compiler_output": r.compiler_output,
    })
}

async fn stop_capture_inner(capture_task: &Arc<Mutex<Option<JoinHandle<()>>>>) {
    let mut g = capture_task.lock().await;
    if let Some(h) = g.take() {
        h.abort();
    }
}

#[tauri::command]
async fn connect_agent(
    state: State<'_, AppState>,
    agent: String,
    token: String,
) -> Result<String, String> {
    stop_capture_inner(&state.capture_task).await;
    let tok = if token.trim().is_empty() {
        None
    } else {
        Some(token)
    };
    let mut c = PhantomClient::connect(&agent, tok)
        .await
        .map_err(|e| format!("connect: {e}"))?;
    let sid = c
        .open_session("")
        .await
        .map_err(|e| format!("open_session: {e}"))?;
    let stream_peer = c.clone();
    let mut eg = state.exec_client.lock().await;
    let mut sg = state.stream_client.lock().await;
    *eg = Some(c);
    *sg = Some(stream_peer);
    Ok(sid)
}

#[tauri::command]
async fn disconnect_agent(state: State<'_, AppState>) -> Result<(), String> {
    stop_capture_inner(&state.capture_task).await;
    let mut sg = state.stream_client.lock().await;
    *sg = None;
    let mut eg = state.exec_client.lock().await;
    if let Some(mut c) = eg.take() {
        let _ = c.close_session().await;
    }
    Ok(())
}

#[tauri::command]
async fn start_capture(app: AppHandle, state: State<'_, AppState>) -> Result<(), String> {
    stop_capture_inner(&state.capture_task).await;

    let app2 = app.clone();
    let capture_task = state.capture_task.clone();
    let stream_client = state.stream_client.clone();
    let h = tokio::spawn(async move {
        let mut attempt = 0u32;
        loop {
            let stream_result = {
                let mut guard = stream_client.lock().await;
                let c = match guard.as_mut() {
                    Some(c) => c,
                    None => break,
                };
                c.stream_events().await
            };
            let mut stream = match stream_result {
                Ok(s) => {
                    attempt = 0;
                    s
                }
                Err(e) => {
                    let _ = app2.emit(
                        "debug-event-error",
                        json!({ "message": format!("stream_events: {e}") }),
                    );
                    tokio::time::sleep(Duration::from_millis(stream_backoff_ms(attempt))).await;
                    attempt = attempt.saturating_add(1);
                    continue;
                }
            };

            loop {
                match stream.message().await {
                    Ok(Some(ev)) => {
                        let pl = &ev.payload;
                        let take = pl.len().min(8192);
                        let payload_hex = hex::encode(&pl[..take]);
                        let truncated = pl.len() > take;
                        let payload_utf8 = String::from_utf8_lossy(&pl[..take]).to_string();
                        let j = json!({
                            "timestamp_ns": ev.timestamp_ns,
                            "session_id": ev.session_id,
                            "event_type": ev.event_type,
                            "event_type_name": event_type_name(ev.event_type),
                            "pid": ev.pid,
                            "tgid": ev.tgid,
                            "cpu": ev.cpu,
                            "probe_id": ev.probe_id,
                            "payload_hex": payload_hex,
                            "payload_truncated": truncated,
                            "payload_utf8": payload_utf8,
                        });
                        let _ = app2.emit("debug-event", j);
                        // Let unary RPCs (execute / info) get scheduled under high event rates.
                        tokio::task::yield_now().await;
                    }
                    Ok(None) => {
                        tokio::time::sleep(Duration::from_millis(stream_backoff_ms(attempt))).await;
                        attempt = attempt.saturating_add(1);
                        break;
                    }
                    Err(e) => {
                        let _ = app2.emit(
                            "debug-event-error",
                            json!({ "message": e.to_string() }),
                        );
                        tokio::time::sleep(Duration::from_millis(stream_backoff_ms(attempt))).await;
                        attempt = attempt.saturating_add(1);
                        break;
                    }
                }
            }
        }
        let mut g = capture_task.lock().await;
        *g = None;
    });

    *state.capture_task.lock().await = Some(h);
    Ok(())
}

#[tauri::command]
async fn stop_capture(state: State<'_, AppState>) -> Result<(), String> {
    stop_capture_inner(&state.capture_task).await;
    Ok(())
}

#[tauri::command]
async fn fetch_host_metrics(state: State<'_, AppState>) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .get_host_metrics()
        .await
        .map_err(|e| format!("get_host_metrics: {e}"))?;
    Ok(host_metrics_json(r))
}

#[tauri::command]
async fn fetch_task_tree(state: State<'_, AppState>, tgid: u32) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .get_task_tree(tgid)
        .await
        .map_err(|e| format!("get_task_tree: {e}"))?;
    Ok(task_tree_json(r))
}

#[tauri::command]
async fn execute_cmd(state: State<'_, AppState>, command_line: String) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .execute(&command_line)
        .await
        .map_err(|e| format!("execute: {e}"))?;
    let output = r.into_result().map_err(|msg| msg)?;
    Ok(json!({
        "ok": true,
        "output": output,
        "error_message": "",
    }))
}

#[tauri::command]
async fn list_tracepoints_cmd(
    state: State<'_, AppState>,
    prefix: String,
    max_entries: u32,
) -> Result<Vec<String>, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    c.list_tracepoints(&prefix, max_entries)
        .await
        .map_err(|e| format!("list_tracepoints: {e}"))
}

#[tauri::command]
async fn list_kprobes_cmd(
    state: State<'_, AppState>,
    prefix: String,
    max_entries: u32,
) -> Result<Vec<String>, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    c.list_kprobe_symbols(&prefix, max_entries)
        .await
        .map_err(|e| format!("list_kprobe_symbols: {e}"))
}

#[tauri::command]
async fn list_uprobes_cmd(
    state: State<'_, AppState>,
    binary_path: String,
    prefix: String,
    max_entries: u32,
) -> Result<Vec<String>, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    c.list_uprobe_symbols(&binary_path, &prefix, max_entries)
        .await
        .map_err(|e| format!("list_uprobe_symbols: {e}"))
}

#[tauri::command]
async fn compile_hook(
    state: State<'_, AppState>,
    source: String,
    attach: String,
    program_name: String,
) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .compile_and_attach(&source, &attach, &program_name)
        .await
        .map_err(|e| format!("compile_and_attach: {e}"))?;
    Ok(compile_json(r))
}

#[tauri::command]
async fn preview_hook_template(
    state: State<'_, AppState>,
    attach_point: String,
    sec_expression: String,
    code_snippet: String,
) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .preview_hook_template(&attach_point, &sec_expression, &code_snippet)
        .await
        .map_err(|e| format!("preview_hook_template: {e}"))?;
    Ok(preview_template_json(r))
}

#[tauri::command]
async fn validate_compile_source(state: State<'_, AppState>, source: String) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .validate_compile_source(&source)
        .await
        .map_err(|e| format!("validate_compile_source: {e}"))?;
    Ok(validate_source_json(r))
}

#[tauri::command]
async fn list_hook_maps_cmd(state: State<'_, AppState>, hook_id: String) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .list_hook_maps(&hook_id)
        .await
        .map_err(|e| format!("list_hook_maps: {e}"))?;
    Ok(list_hook_maps_json(r))
}

#[tauri::command]
async fn read_hook_map_cmd(
    state: State<'_, AppState>,
    hook_id: String,
    map_name: String,
    max_entries: u32,
) -> Result<Value, String> {
    let mut guard = state.exec_client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .read_hook_map(&hook_id, &map_name, max_entries)
        .await
        .map_err(|e| format!("read_hook_map: {e}"))?;
    Ok(read_hook_map_json(r))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let state = AppState {
        exec_client: Arc::new(Mutex::new(None)),
        stream_client: Arc::new(Mutex::new(None)),
        capture_task: Arc::new(Mutex::new(None)),
    };

    tauri::Builder::default()
        .manage(state)
        .setup(|app| {
            let phantom_export = MenuItem::with_id(
                app,
                "phantom_export",
                "Export JSONL…",
                true,
                Some("CmdOrCtrl+E"),
            )?;
            let phantom_clear = MenuItem::with_id(
                app,
                "phantom_clear",
                "Clear Events",
                true,
                None::<&str>,
            )?;
            let file_menu = SubmenuBuilder::new(app, "File")
                .item(&phantom_export)
                .item(&phantom_clear)
                .separator()
                .item(&PredefinedMenuItem::quit(app, None)?)
                .build()?;

            let edit_menu = SubmenuBuilder::new(app, "Edit")
                .cut()
                .copy()
                .paste()
                .separator()
                .select_all()
                .build()?;

            let phantom_settings = MenuItem::with_id(
                app,
                "phantom_settings",
                "Settings…",
                true,
                Some("CmdOrCtrl+,"),
            )?;
            let view_menu = SubmenuBuilder::new(app, "View").item(&phantom_settings).build()?;

            let phantom_about =
                MenuItem::with_id(app, "phantom_about", "About Phantom", true, None::<&str>)?;
            let help_menu = SubmenuBuilder::new(app, "Help")
                .item(&phantom_about)
                .build()?;

            let menu = MenuBuilder::new(app)
                .item(&file_menu)
                .item(&edit_menu)
                .item(&view_menu)
                .item(&help_menu)
                .build()?;

            app.set_menu(menu)?;
            Ok(())
        })
        .on_menu_event(|app, event| {
            let action = if event.id() == "phantom_export" {
                Some("export")
            } else if event.id() == "phantom_clear" {
                Some("clear")
            } else if event.id() == "phantom_settings" {
                Some("open_settings")
            } else if event.id() == "phantom_about" {
                Some("about")
            } else {
                None
            };
            if let Some(a) = action {
                let _ = app.emit("phantom-menu", json!({ "action": a }));
            }
        })
        .invoke_handler(tauri::generate_handler![
            connect_agent,
            disconnect_agent,
            start_capture,
            stop_capture,
            fetch_host_metrics,
            fetch_task_tree,
            execute_cmd,
            list_tracepoints_cmd,
            list_kprobes_cmd,
            list_uprobes_cmd,
            compile_hook,
            preview_hook_template,
            validate_compile_source,
            list_hook_maps_cmd,
            read_hook_map_cmd,
        ])
        .run(tauri::generate_context!())
        .expect("error while running Phantom desktop");
}
