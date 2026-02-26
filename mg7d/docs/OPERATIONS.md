# Operations (Phase 0–3)

## systemd deployment

- **Binary:** Install to e.g. `/usr/local/bin/mg7d-agent` (and optionally `mg7d-ctl`).
- **Config:** Place YAML at e.g. `/etc/mg7d/agent.yaml`. Ensure the agent process can read it and the log path.
- **User:** Run as a dedicated user with read access to the 7DTD log and (if used) network access to telnet.
- **Unit:** Example unit is in `deploy/systemd/mg7d-agent.service`. Adjust paths to match your install.

```bash
sudo cp deploy/systemd/mg7d-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now mg7d-agent
```

---

## Prometheus scrape

Add a scrape config for the agent (metrics and healthz are on the same server):

```yaml
scrape_configs:
  - job_name: mg7d
    static_configs:
      - targets: ['127.0.0.1:9090']
    metrics_path: /metrics
```

To check readiness (optional):

```yaml
# If using a probe that expects HTTP 200
# GET http://127.0.0.1:9090/healthz → 200 ok
```

---

## Troubleshooting

### Telnet auth failures

- Ensure `telnet.password` in config matches the 7DTD server telnet password.
- If telnet is disabled on the server, leave `telnet.host` empty or set port to 0; the agent will run without telnet and without applying actions.

### Log path issues

- `log_path` must be readable by the process. Use absolute paths in production.
- If the file is missing at startup, the tailer will retry with backoff; ensure the path is correct and 7DTD is writing the log (e.g. `output_log.txt` in the game directory or as configured in 7DTD).

### Log rotation

- The tailer is rotation-safe (copytruncate and rename+recreate). No extra logrotate config is required beyond normal 7DTD log rotation.

### “Why no actions fired?”

- FPS guardrail only acts when FPS is below `threshold_low` for at least `require_low_samples` in the sample window.
- Telnet must be configured (host/port) and connected; if the connection is down, actions are queued but may fail until reconnect.
- Check logs for “applier enqueue failed” (queue full) or telnet errors.

---

## Replay tests and fixtures

- **Fixture:** `testdata/replay_fps.log` contains sample “Time:” lines.
- **Tests:** `go test ./internal/logtail/...` (integration test that replays the fixture), `go test ./internal/parser/...`, `go test ./internal/policy/...`.
- To add fixtures: add log snippets under `testdata/` and reference them from tests in the appropriate package (e.g. `internal/logtail`, `internal/parser`).

```bash
make test
# or
go test ./...
```
