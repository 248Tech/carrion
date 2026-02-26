# Contributing to mg7d

Thank you for considering contributing. Please keep the project’s invariants and architecture in mind.

## Invariants (do not break)

1. **No telnet spam** — Single persistent connection per instance; bounded command rate.
2. **Bounded RAM** — Ring buffers only; no unbounded history.
3. **Reversible changes** — Every automated change has a deterministic baseline restore.
4. **Hysteresis + cooldown** — Policies must avoid flapping (cooldowns, stability windows).
5. **Audit** — Every action must produce an audit event.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for how these are enforced in code.

## Before submitting

1. **Format:** Run `make fmt` (or `gofmt -s -w .`).
2. **Tests:** Run `make test` (or `go test ./... -count=1`). Optionally use `-race`.
3. **Lint:** Run `make lint` if the repo uses golangci-lint.

CI runs these on push/PR; ensure they pass locally.

## Adding log fixtures

- Put sample log files or snippets under `testdata/`.
- Add or extend tests in the relevant package (e.g. `internal/logtail`, `internal/parser`, `internal/policy`) that read the fixture and assert expected behavior.
- Do not commit huge or sensitive logs; keep fixtures minimal and synthetic where possible.

## Scope

Phase 0–3 is implemented and stable. Do not change public behavior or APIs for Phase 0–3 unless required for correctness. New features (e.g. Phase 4+) should be discussed in an issue first.

## Code of conduct

This project adheres to the Contributor Covenant v2.1. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
