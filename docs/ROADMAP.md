# Roadmap

## Phase 0 – Foundations — **Done**

- Repo layout, Go module, CI (build, test, lint).
- Makefile (build, test, lint, clean).

## Phase 1 – Observability MVP — **Done**

- Log tailer (rotation-safe, partial-line safe).
- Parser for “Time:” lines; atomic snapshot store.
- Prometheus metrics (mg7d_fps, mg7d_players, mg7d_chunks, mg7d_entities, mg7d_zombies, mg7d_heap_mb, mg7d_rss_mb) and GET /metrics, GET /healthz.

## Phase 2 – Telnet client — **Done**

- Single persistent connection, rate limiting (token bucket), reconnect with backoff, circuit breaker, command queue.

## Phase 3 – Policy engine v1 (FPS Guardrail) — **Done**

- Policy pipeline (snapshot → actions); FPS Guard with throttle steps and restore after stability; baseline and RestoreBaseline; applier and audit ring.

---

## Planned / optional (not implemented)

- **Phase 4:** Multi-instance agent (use all entries in `instances[]`).
- **Phase 5:** `mg7d-ctl` status/control endpoints and CLI commands.
- **Phase 6:** Additional policies (e.g. player-cap, entity guardrails).
- **Phase 7:** API auth (e.g. enforce `auth_token` for /metrics or admin endpoints).
- **Phase 8:** Persistent audit log (file or external sink).
- **Phase 9:** Distribution packaging (e.g. .deb, .rpm).
- **Phase 10:** Optional web UI or dashboards.

These are summary bullets only; scope and order may change.
