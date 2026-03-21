# Operations and troubleshooting

## Deployment

### Systemd

1. Build and install the agent binary, e.g. to `/usr/local/bin/phantom-agent`.
2. Copy or adapt [deploy/systemd/phantom-agent.service](../deploy/systemd/phantom-agent.service).
3. Enable and start:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable phantom-agent
   sudo systemctl start phantom-agent
   ```

4. Optionally set `PHANTOM_TOKEN` in the service file or use an environment file.

### Configuration

Agent is configured via code (see `server.DefaultConfig()` and `server.Config`). Typical knobs:

- `ListenAddr` — gRPC listen address (default `:9090`).
- `Token` — if set, all RPCs must send `Authorization: Bearer <token>`.
- `HealthAddr` — if set, HTTP server serves `GET /health` (e.g. `:8080`).
- `RateLimit`, `RateBurst` — per-session rate limit.
- `MaxBreak`, `MaxTrace`, `MaxHooks` — per-session quotas.
- `Audit` — optional audit logger (e.g. `server.NewAuditLog(os.Stderr)`).

## Troubleshooting

### CLI: "missing -agent" / connection errors

Pass the agent address: `phantom-cli --agent host:9090` (or `-a host:9090`).

### CLI: "connect: ... connection refused"

- Agent is not running on the given host/port.
- Firewall or network blocks the port.

### CLI: "session not found"

- Session was closed (e.g. `CloseSession` or agent restart).
- Reconnect and call `OpenSession` again to get a new session.

### Agent: "listen ... address already in use"

Another process is using the listen port. Change `ListenAddr` or stop the other process.

### Agent: eBPF load fails (LoadFromFile)

- Build eBPF on Linux: `make build-bpf`.
- Ensure the .o path is correct and the binary was built for the same architecture.
- Kernel may need CAP_BPF, CAP_PERFMON, CAP_SYS_ADMIN for load/attach.

### Rate limited / quota

Increase `RateLimit`/`RateBurst` or `MaxBreak`/`MaxTrace`/`MaxHooks` in config, or reduce usage per session.

### Health check

If `HealthAddr` is set, `curl http://<agent>:<port>/health` should return 200 OK. Use this for load balancers or readiness probes.

### Prometheus metrics

If `MetricsAddr` is set (e.g. `-metrics :9091` or `PHANTOM_METRICS=:9091`), the agent exposes `GET /metrics` with counters `phantom_commands_total`, `phantom_events_total` and gauge `phantom_sessions_active`.
