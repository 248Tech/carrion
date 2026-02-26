# Configuration reference (Phase 0–3)

This document describes the YAML configuration schema actually used by the mg7d agent. Validation is strict: invalid or missing required fields cause the agent to fail at startup.

## Top-level keys

| Key         | Type     | Required | Description                    |
|------------|----------|----------|--------------------------------|
| `instances`| array    | yes      | List of 7DTD server instances. |
| `api`      | object   | no       | HTTP API server settings.      |
| `metrics`  | object   | no       | Prometheus metrics settings.  |

**Note:** In Phase 0–3 the agent uses only the **first** instance in `instances`. Additional entries are accepted for future multi-instance support.

---

## `instances[]`

Each element must have:

| Key         | Type   | Required | Description |
|------------|--------|----------|-------------|
| `name`     | string | yes      | Instance identifier; used as Prometheus label `instance="<name>"`. |
| `log_path` | string | yes      | Absolute or relative path to the 7DTD game log (e.g. `output_log.txt`). |
| `telnet`   | object | no       | Telnet connection and safety settings. If `host`/`port` are empty or zero, telnet and policy actions are not applied. |
| `policy`   | object | no       | Policy configuration (e.g. FPS guardrail). |
| `actions`  | object | no       | Throttle profiles and baseline for RestoreBaseline. |

### `instances[].telnet`

| Key                 | Type    | Default | Description |
|---------------------|---------|---------|-------------|
| `host`              | string  | —       | Telnet host (e.g. `127.0.0.1`). |
| `port`              | int     | —       | Telnet port (e.g. `8081`). |
| `password`          | string  | `""`    | Telnet password; empty if auth disabled. |
| `rate_limit_per_sec`| float   | `2.0`   | Max commands per second (token bucket). Applied in code if omitted or ≤ 0. |

Reconnect backoff and circuit breaker are not in config; they use internal defaults (e.g. 2s–60s backoff, circuit break after 3 failures).

### `instances[].policy.fps_guard`

| Key                     | Type    | Description |
|-------------------------|---------|-------------|
| `enabled`               | bool    | Turn FPS guardrail on/off. |
| `threshold_low`         | float   | FPS below this (for `require_low_samples` in window) triggers throttle. |
| `threshold_restore`     | float   | FPS above this, sustained for `restore_stable_seconds`, allows restore. |
| `require_low_samples`  | int     | Number of low-FPS samples in the window to trigger. |
| `sample_window_samples`| int     | FPS ring buffer size (samples). |
| `restore_stable_seconds`| float   | Seconds FPS must stay ≥ `threshold_restore` before emitting RestoreBaseline. |
| `cooldown_seconds`      | float   | Minimum seconds between throttle steps. |
| `delta_spike_threshold` | float   | Reserved (spike detection). |
| `spike_window_seconds`  | float   | Reserved. |
| `throttle_profile`      | string  | Name of profile in `actions.throttle_profiles`. |

### `instances[].actions`

| Key                 | Type  | Description |
|---------------------|-------|-------------|
| `baseline`          | map   | Pref name → value for RestoreBaseline (e.g. `MaxSpawnedZombies: "50"`). |
| `throttle_profiles` | map   | Named throttle profiles; each has `steps[]`. |

### `instances[].actions.throttle_profiles.<name>.steps[]`

| Key    | Type   | Description |
|--------|--------|-------------|
| `pref` | string | Game pref name (e.g. `MaxSpawnedZombies`). |
| `value`| string | Value to set (e.g. `"30"`). |

Steps are applied in order as FPS stays low and cooldown allows (first step on trigger, next steps on subsequent cooldown intervals).

---

## `api`

| Key          | Type   | Default         | Description |
|--------------|--------|-----------------|-------------|
| `listen`     | string | `127.0.0.1:9090`| HTTP listen address. |
| `auth_token` | string | `""`            | Reserved; not enforced in Phase 0–3. |

---

## `metrics`

| Key     | Type   | Default   | Description |
|---------|--------|-----------|-------------|
| `enable`| bool   | —         | If true, HTTP server runs and exposes `/metrics` and `/healthz`. |
| `path`  | string | `/metrics`| Path for Prometheus scrape. |

---

## Annotated example (single instance)

```yaml
instances:
  - name: default
    log_path: /var/log/7dtd/output_log.txt
    telnet:
      host: 127.0.0.1
      port: 8081
      password: ""
      rate_limit_per_sec: 2.0
    policy:
      fps_guard:
        enabled: true
        threshold_low: 25
        threshold_restore: 40
        require_low_samples: 3
        sample_window_samples: 60
        restore_stable_seconds: 120
        cooldown_seconds: 60
        throttle_profile: default
    actions:
      baseline:
        MaxSpawnedZombies: "50"
      throttle_profiles:
        default:
          steps:
            - pref: MaxSpawnedZombies
              value: "30"
            - pref: MaxSpawnedZombies
              value: "20"

api:
  listen: 127.0.0.1:9090
  auth_token: ""

metrics:
  enable: true
  path: /metrics
```

---

## Validation behavior

- At least one instance is required.
- Each instance must have `name` and `log_path`.
- If `api.listen` is empty, it is set to `127.0.0.1:9090`.
- If `metrics.path` is empty, it is set to `/metrics`.
- If `telnet.rate_limit_per_sec` is missing or ≤ 0, the telnet client uses `2.0` in code.

Invalid config causes the agent to exit with an error at startup.
