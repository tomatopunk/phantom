# Near-term maturity inventory

Prioritized gaps between **capabilities that exist** and **quality / usability**. Order reflects impact on interactive debugging and agent integrations.

## Addressed (baseline)

1. **REPL command dispatch & errors** — Table-driven dispatch in [`lib/agent/server/executor_dispatch.go`](../lib/agent/server/executor_dispatch.go); failures remain `ExecuteResponse.ok` / `error_message` at the gRPC layer.
2. **MCP vs REPL parity** — [`lib/agent/mcp/backend.go`](../lib/agent/mcp/backend.go) `ExecuteCommandLine` maps `ok: false` to JSON-RPC errors for command-style tools.
3. **Rust client ergonomics** — [`lib/phantom-client`](../lib/phantom-client/src/lib.rs) `ExecuteResponse::into_result()` for CLI / Tauri.
4. **Desktop command path** — [`src/desktop/src-tauri/src/lib.rs`](../src/desktop/src-tauri/src/lib.rs) `execute_cmd` uses `into_result()`; [`src/desktop/src/api.ts`](../src/desktop/src/api.ts) documents invoke rejection on logical failure; UI call sites aligned.
5. **Desktop structure** — [`src/desktop/src/App.tsx`](../src/desktop/src/App.tsx) composes feature panels: [`AppHeader.tsx`](../src/desktop/src/components/AppHeader.tsx), [`MetricsDimensionPanel.tsx`](../src/desktop/src/components/MetricsDimensionPanel.tsx), [`DiscoverPanel.tsx`](../src/desktop/src/components/DiscoverPanel.tsx), [`CmdReplBlock.tsx`](../src/desktop/src/components/CmdReplBlock.tsx), [`EventsStreamPanel.tsx`](../src/desktop/src/components/EventsStreamPanel.tsx), [`AppFooter.tsx`](../src/desktop/src/components/AppFooter.tsx); shared types in [`src/desktop/src/app/types.ts`](../src/desktop/src/app/types.ts) and event helpers in [`src/desktop/src/app/eventUtils.ts`](../src/desktop/src/app/eventUtils.ts).
6. **Stream lifecycle** — [`src/desktop/src-tauri/src/lib.rs`](../src/desktop/src-tauri/src/lib.rs) `start_capture` reconnects `stream_events` after stream end or message error, with exponential backoff (capped). Frontend no longer clears capture state on transient `debug-event-error` (see [`App.tsx`](../src/desktop/src/App.tsx) listener); user stops via **Stop capture** or disconnect.

## Active / deferred

7. **Hook attach / compile path** — `hook attach` and `CompileAndAttach` share `tryCompileAttachHook`; attach failures prefix `attach failed:` ([`executor_compile_rpc.go`](../lib/agent/server/executor_compile_rpc.go)). Further internal helpers optional.
8. **`list` on non-Linux** — Kernel symbol listing returns a descriptive message with `ok: true` on unsupported platforms; changing that would need build-tagged tests.

This list tracks [roadmap.md](roadmap.md) Near-term themes; update both when priorities shift.
