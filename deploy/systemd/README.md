# systemd deployment

- **mg7d-agent.service** — Example unit for the agent. Config path is passed as first argument (no `--config` flag).
  - Copy to `/etc/systemd/system/`, then `systemctl daemon-reload` and `systemctl enable --now mg7d-agent`.
  - Adjust `ExecStart` (binary path and config path), and optionally `User`, `WorkingDirectory`, and hardening options.
- See [docs/OPERATIONS.md](../../docs/OPERATIONS.md) for paths and troubleshooting.
