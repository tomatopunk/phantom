# Near-term maturity inventory

Prioritized gaps between **capabilities that exist** and **quality / usability** we want next. Order reflects impact on interactive debugging and agent integrations.

1. **REPL command dispatch & errors** — `commandExecutor` used a large verb `switch` (hard to extend, high cyclomatic complexity). Command failures should stay consistent (`ExecuteResponse.ok` / `error_message`) across entry points.
2. **MCP vs REPL parity** — Some MCP tools treated `Execute` failures as JSON-RPC success with text bodies; others as errors. Callers should see a **uniform** failure mode aligned with gRPC/REPL semantics.
3. **Rust client ergonomics** — CLI and Desktop repeated `if !resp.ok` handling; a shared helper on `ExecuteResponse` reduces drift and clarifies empty error messages from the agent.
4. **Desktop command path** — Surfacing agent command failures as Tauri `Err` matches other commands (metrics, task tree) and lets the UI use a single error path.
5. **Hook attach / compile path** — `hook attach` and `CompileAndAttach` already share `tryCompileAttachHook`; further splits (quota defer, attach helpers) are optional once dispatch and MCP are stable.
6. **`list` on non-Linux** — Kernel symbol listing returns a descriptive message with `ok: true` on unsupported platforms; changing that would break cross-platform tests without build-tagged expectations.

This list tracks [roadmap.md](roadmap.md) Near-term themes; update both when priorities shift.
