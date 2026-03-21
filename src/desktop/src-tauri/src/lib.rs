use std::sync::Arc;

use phantom_client::{
    CompileAndAttachResponse, ExecuteResponse, GetHostMetricsResponse, GetTaskTreeResponse,
    PhantomClient,
};
use serde_json::{json, Value};
use tauri::{AppHandle, Emitter, State};
use tokio::sync::Mutex;
use tokio::task::JoinHandle;

#[derive(Clone)]
struct AppState {
    client: Arc<Mutex<Option<PhantomClient>>>,
    capture_task: Arc<Mutex<Option<JoinHandle<()>>>>,
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

fn execute_json(r: ExecuteResponse) -> Value {
    json!({
        "ok": r.ok,
        "output": r.output,
        "error_message": r.error_message,
    })
}

fn compile_json(r: CompileAndAttachResponse) -> Value {
    json!({
        "ok": r.ok,
        "error_message": r.error_message,
        "hook_id": r.hook_id,
        "attach_point": r.attach_point,
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
    let mut guard = state.client.lock().await;
    *guard = Some(c);
    Ok(sid)
}

#[tauri::command]
async fn disconnect_agent(state: State<'_, AppState>) -> Result<(), String> {
    stop_capture_inner(&state.capture_task).await;
    let mut guard = state.client.lock().await;
    if let Some(mut c) = guard.take() {
        let _ = c.close_session().await;
    }
    Ok(())
}

#[tauri::command]
async fn start_capture(app: AppHandle, state: State<'_, AppState>) -> Result<(), String> {
    stop_capture_inner(&state.capture_task).await;

    let stream = {
        let mut guard = state.client.lock().await;
        let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
        c.stream_events()
            .await
            .map_err(|e| format!("stream_events: {e}"))?
    };

    let app2 = app.clone();
    let capture_task = state.capture_task.clone();
    let h = tokio::spawn(async move {
        let mut stream = stream;
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
                }
                Ok(None) => break,
                Err(e) => {
                    let _ = app2.emit(
                        "debug-event-error",
                        json!({ "message": e.to_string() }),
                    );
                    break;
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
    let mut guard = state.client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .get_host_metrics()
        .await
        .map_err(|e| format!("get_host_metrics: {e}"))?;
    Ok(host_metrics_json(r))
}

#[tauri::command]
async fn fetch_task_tree(state: State<'_, AppState>, tgid: u32) -> Result<Value, String> {
    let mut guard = state.client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .get_task_tree(tgid)
        .await
        .map_err(|e| format!("get_task_tree: {e}"))?;
    Ok(task_tree_json(r))
}

#[tauri::command]
async fn execute_cmd(state: State<'_, AppState>, command_line: String) -> Result<Value, String> {
    let mut guard = state.client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .execute(&command_line)
        .await
        .map_err(|e| format!("execute: {e}"))?;
    Ok(execute_json(r))
}

#[tauri::command]
async fn list_tracepoints_cmd(
    state: State<'_, AppState>,
    prefix: String,
    max_entries: u32,
) -> Result<Vec<String>, String> {
    let mut guard = state.client.lock().await;
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
    let mut guard = state.client.lock().await;
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
    let mut guard = state.client.lock().await;
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
    let mut guard = state.client.lock().await;
    let c = guard.as_mut().ok_or_else(|| "not connected".to_string())?;
    let r = c
        .compile_and_attach(&source, &attach, &program_name)
        .await
        .map_err(|e| format!("compile_and_attach: {e}"))?;
    Ok(compile_json(r))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let state = AppState {
        client: Arc::new(Mutex::new(None)),
        capture_task: Arc::new(Mutex::new(None)),
    };

    tauri::Builder::default()
        .manage(state)
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
        ])
        .run(tauri::generate_context!())
        .expect("error while running Phantom desktop");
}
