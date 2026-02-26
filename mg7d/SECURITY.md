# Security

## Reporting vulnerabilities

**Do not open public issues for security vulnerabilities.**

Please report security issues via **GitHub Security Advisories**:

1. Go to the repository on GitHub.
2. Click **Security** → **Advisories** → **Report a vulnerability**.
3. Describe the issue and steps to reproduce.

We will respond and coordinate disclosure. Do not disclose the vulnerability publicly until it has been addressed.

## Supported versions

We release security fixes for the current release line. Upgrade to the latest version to receive fixes.

## Security-related behavior (Phase 0–3)

- The agent reads a local log file and (optionally) connects to 7DTD via telnet. Ensure config files and telnet passwords are not exposed (e.g. restrict file permissions, do not commit secrets).
- The HTTP server exposes `/metrics` and `/healthz` only; API auth (`auth_token`) is not enforced in Phase 0–3.
- Run the agent with least privilege (dedicated user, read-only access to the log path, network only to telnet if needed).
