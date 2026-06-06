# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.3.x   | :white_check_mark: |
| < 0.3   | :x:                |

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Please report security issues privately via GitHub Security Advisories:
https://github.com/wzhongyou/baize/security/advisories/new

You will receive a response within 48 hours. We will work with you to understand the scope, reproduce the issue, and prepare a fix.

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Environment details (OS, Go version, etc.)
- Potential impact

### Disclosure Policy

1. Reporter submits vulnerability privately
2. Maintainer acknowledges within 48 hours
3. Fix is developed and tested
4. CVE is requested if applicable
5. Public disclosure after patch is released

## Security Design

Baize is built with security as a core design principle:

- **OS-Native Sandbox** — macOS Seatbelt / Linux Bubblewrap for process isolation
- **Permission Pipeline** — deny-first model, all tool calls go through permission checks
- **Local-First** — API keys and session data stay on your machine
- **No Telemetry** — no usage data leaves your machine (by default)

See [docs/subsystems/permission.md](docs/subsystems/permission.md) for details on the permission system.
