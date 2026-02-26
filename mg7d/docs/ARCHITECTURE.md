# mg7d Agent — Architecture (Phase 0–3)

## Invariants (non-negotiable)

1. **No telnet spam**: One persistent connection per instance; command rate limited (token bucket).
2. **Bounded RAM**: Ring buffers only; no unbounded history.
3. **Reversible changes**: Every automated setting change has a deterministic baseline restore.
4. **Hysteresis + cooldown**: Policies use cooldowns and stability windows to avoid flapping.
5. **Audit**: Every action produces an audit event (in-memory ring; structured logging).

## Dataflow

```
Log file → Tailer → lines (chan) → Parser → Snapshot (atomic)
                                              ↓
                         Metrics (Prometheus)  Policy Engine → Actions → Applier → Telnet
                         /healthz              ↓
                                         Audit ring
```

## Concurrency model (per instance)

- **Tailer goroutine**: Reads log file, survives rotation, emits complete lines on channel. Uses fsnotify + optional poll; no busy-spin.
- **Parser goroutine**: Consumes lines, parses "Time:" lines, updates atomic snapshot, updates metrics, runs policy engine, enqueues actions to applier.
- **Telnet goroutine**: Maintains one connection, drain loop for server output, send loop with rate limiter and circuit breaker.
- **Applier goroutine**: Consumes action queue, sends commands via telnet, records audit events.
- **HTTP server**: Serves GET /metrics (Prometheus text format) and GET /healthz (200 ok). Single listen address.

## How invariants are enforced in code

- **No telnet spam:** One `telnet.Client` per instance; token bucket in `takeToken()`; command queue bounded (e.g. 64); circuit breaker after N send failures.
- **Bounded RAM:** `util.Ring` for FPS samples and audit events; logtail uses a bounded line channel; no unbounded slices.
- **Reversible changes:** Baseline map in config and applier; `RestoreBaseline` action sends setpref for each baseline entry.
- **Hysteresis + cooldown:** FPS guard uses `cooldown_seconds` between steps and `restore_stable_seconds` before restore; state (throttled, lastStep, restoreAt) avoids flapping.
- **Audit:** Applier calls `audit.Append()` on queued, sent, success, failure; audit ring is fixed size.

## Failure modes handled (Phase 0–3)

- **Log rotation:** Tailer detects truncation or new inode and reopens the file; backoff if file is missing.
- **Partial lines:** Tailer buffers incomplete lines and only emits complete lines.
- **Telnet disconnect:** Client reconnects with exponential backoff; send loop exits and Run() re-establishes connection.
- **Malformed / non-Time lines:** Parser returns ok=false and is skipped; no crash.
- **Config invalid:** Validate() fails fast at startup before any goroutines.
- **Metrics server:** Serves /metrics and /healthz; shutdown on context cancel.

## Components

- **internal/logtail**: Rotation-safe, partial-line-safe tailer; bounded channel.
- **internal/parser**: "Time:" line → Snapshot; resilient to order and missing tokens.
- **internal/state**: Atomic snapshot store (atomic.Value); audit ring (fixed-size).
- **internal/metrics**: Prometheus gauges (mg7d_fps, mg7d_players, mg7d_chunks, mg7d_entities, mg7d_zombies, mg7d_heap_mb, mg7d_rss_mb) with instance label.
- **internal/api**: HTTP server exposing /metrics and /healthz.
- **internal/telnet**: One connection, token-bucket rate limit, exponential backoff reconnect, circuit breaker.
- **internal/actions**: Action types (SetGamePref, Say, RestoreBaseline, Noop); applier with bounded queue and baseline.
- **internal/policy**: Engine + FPS Guard (ring of FPS samples, throttle steps, restore after stable window).

## Configuration

YAML with `instances[]`, each: `name`, `log_path`, `telnet`, `policy.fps_guard`, `actions.throttle_profiles` and `actions.baseline`. Config validation is strict; fail fast on invalid. See [CONFIG.md](CONFIG.md).

## Definition of done (Phase 0–3)

- `make test` and CI pass.
- Agent run against a replay log shows metrics updating.
- Policy triggers throttle and restore in replay tests.
- Telnet holds one stable session and respects rate limiter.
