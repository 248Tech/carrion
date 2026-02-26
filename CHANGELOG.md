# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

- Nothing yet.

## [v0.1.0] — Phase 0–3 (2025-02-26)

### Added

- **Phase 0:** Repo layout, Go module (Go 1.22), Makefile (build, test, lint, clean), CI workflow (build, test, golangci-lint).
- **Phase 1:** Rotation-safe log tailer; parser for "Time:" lines; atomic snapshot store; Prometheus metrics (mg7d_fps, mg7d_players, mg7d_chunks, mg7d_entities, mg7d_zombies, mg7d_heap_mb, mg7d_rss_mb) with instance label; GET /metrics and GET /healthz.
- **Phase 2:** Telnet client — single persistent connection, token-bucket rate limit, reconnect with backoff, circuit breaker, bounded command queue.
- **Phase 3:** Policy engine and FPS Guardrail — throttle stepwise on sustained low FPS, restore baseline after stability window; baseline and throttle profiles in config; action applier and audit ring.
- YAML config with instances, telnet, policy.fps_guard, actions.throttle_profiles and actions.baseline; strict validation.
- Docs: ARCHITECTURE, CONFIG, OPERATIONS, ROADMAP. Example config in configs/example.agent.yaml.
- Replay fixture testdata/replay_fps.log and tests for logtail, parser, policy.

### Invariants (non-negotiable)

- No telnet spam; bounded RAM (ring buffers); reversible changes; hysteresis/cooldown; every action audited.

[Unreleased]: https://github.com/mg7d/mg7d/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/mg7d/mg7d/releases/tag/v0.1.0
