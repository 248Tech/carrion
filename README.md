# mg7d

A safe, reversible autopilot/observability agent for 7 Days to Die dedicated servers (log tail → state → policy → telnet).

The agent tails the 7DTD game log, parses status lines into an atomic snapshot, exposes Prometheus metrics, and—optionally—runs a policy (FPS Guardrail) that throttles the server stepwise when FPS collapses and restores baseline after stability. All automated changes are reversible and audited; telnet is rate-limited and runs over a single persistent connection.

**Safety focus:** No telnet spam, bounded RAM (ring buffers), reversible changes, hysteresis/cooldown to avoid flapping, and every action recorded in an audit ring.

---

## Use cases (Phase 0–3)

- **Prometheus metrics from 7DTD log tailing** — FPS, players, chunks, entities, zombies, heap/RSS (gauges with `instance` label).
- **FPS Guardrail autopilot** — Stepwise throttle on sustained FPS collapse; restore baseline after a configurable stability window; cooldown between steps.
- **Stable single telnet session** — One persistent connection per agent, token-bucket rate limiting, reconnect with backoff, circuit breaker.
- **Replay tests** — `testdata/replay_fps.log` and tests in `internal/logtail` and `internal/parser` for fixture/replay validation.

---

## Non-negotiable invariants

1. **No telnet spam** — Single persistent connection per instance; bounded command rate (token bucket).
2. **Bounded RAM** — Ring buffers only; no unbounded history.
3. **Reversible changes** — Every automated setting change has a deterministic baseline restore.
4. **Hysteresis + cooldown** — Policies use cooldowns and stability windows to avoid flapping.
5. **Audit** — Every action produces an audit event (in-memory ring; structured logging).

---

## Requirements

- **Go:** 1.22 (see `go.mod`). Only needed for building from source.
- **OS:** Linux recommended (tested on Linux; may work on Windows/macOS for development).
- **7DTD:** Log path must be readable; if using policy/telnet, telnet must be enabled and reachable.
- **Prometheus:** Optional; scrape `/metrics` if you want to graph metrics.

---

## Quickstart

**Build from source:**

```bash
git clone https://github.com/mg7d/mg7d.git
cd mg7d
go mod download
make build
```

Binaries are produced in `./bin/agent` and `./bin/ctl`.

**Run the agent:**

```bash
# config path is first argument (default: config.yaml)
./bin/agent config.yaml
# or
./bin/agent /etc/mg7d/agent.yaml
```

**Verify endpoints:**

```bash
curl -s http://127.0.0.1:9090/healthz
# ok

curl -s http://127.0.0.1:9090/metrics
# Prometheus text format (mg7d_fps, mg7d_players, ...)
```

Ensure `config.yaml` has a valid `log_path` (create an empty file or point to a real 7DTD log). If `metrics.enable` is true, the server listens on `api.listen` (default `127.0.0.1:9090`).

**Replay / tests:** The repo does not ship a “replay mode” CLI flag. To validate behavior against a sample log, run the test suite (which uses `testdata/replay_fps.log`):

```bash
make test
```

---

## Configuration

- **Schema and defaults:** [docs/CONFIG.md](docs/CONFIG.md)
- **Example config:** [configs/example.agent.yaml](configs/example.agent.yaml)

Summary:

- **`instances[]`** — Each instance has `name`, `log_path`, `telnet`, `policy`, `actions`. The agent uses the **first** instance only in Phase 0–3. Metrics use `instance="<name>"`.
- **Telnet safety** — `rate_limit_per_sec` (default 2.0); reconnect backoff and circuit breaker use internal defaults.
- **FPS guard** — `threshold_low`, `threshold_restore`, `require_low_samples`, `sample_window_samples`, `restore_stable_seconds`, `cooldown_seconds`, `throttle_profile`. Throttle profiles define stepwise pref changes; `baseline` defines values for RestoreBaseline.

---

## Project layout

```
mg7d/
  cmd/agent/          # Agent entrypoint (config path as first arg)
  cmd/ctl/            # CLI stub (Phase 0–3)
  configs/            # Example config
  internal/
    api/              # HTTP server (/metrics, /healthz)
    config/           # YAML config load and validate
    logtail/          # Rotation-safe log tailer
    parser/           # "Time:" line → Snapshot
    state/             # Atomic snapshot store, audit ring
    metrics/           # Prometheus gauges
    telnet/            # Persistent client, rate limit, reconnect
    actions/           # Action types and applier
    policy/            # Engine + FPS guard
    util/               # Ring buffer
  deploy/systemd/      # systemd unit example
  docs/                # ARCHITECTURE, CONFIG, OPERATIONS, ROADMAP
  testdata/            # Replay fixture (replay_fps.log)
  go.mod, Makefile
```

---

## Contributing

- Run `make fmt`, `make test`, and `make lint` before submitting changes.
- Keep the five invariants; see [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- Log fixtures go in `testdata/`; add tests that consume them where appropriate.

---

## Security

- Do not open security vulnerabilities in public issues. See [SECURITY.md](SECURITY.md) for disclosure via GitHub Security Advisories.

---

## License

MIT. See [LICENSE](LICENSE).

---

**Repository (for maintainers):** Set the GitHub "About" description to: *A safe, reversible autopilot/observability agent for 7 Days to Die dedicated servers (log tail → state → policy → telnet).* Suggested topics: `go`, `observability`, `prometheus`, `game-server`, `7daystodie`, `telnet`, `automation`, `sre`, `systemd`.
