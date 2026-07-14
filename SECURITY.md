# Security Policy

## Supported versions

Security fixes are applied to the latest tagged release and the `main` branch.
Older releases may not receive backported fixes.

| Version | Supported |
| --- | --- |
| Latest tagged release | Yes |
| `main` | Yes, for the next release |
| Older releases | No |

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability.

Use [GitHub Private Vulnerability Reporting](https://github.com/cloudapp3/vminfo/security/advisories/new) to send the maintainers:

- the affected version, platform, and component
- a clear description of the impact
- reproduction steps or a proof of concept
- any known mitigations

Please omit secrets, access tokens, private hostnames, and unrelated system data. The maintainers will confirm the report, investigate it, and coordinate disclosure and release timing with the reporter.

## Web dashboard security

The dashboard binds to loopback by default. Any non-loopback bind requires a token, but the built-in server uses HTTP and does not provide transport encryption. For remote access, use an HTTPS reverse proxy or SSH tunnel and do not expose the HTTP port directly to an untrusted network.

## Scope

Reports about the CLI, TUI, updater, installer, public Go packages, web dashboard, REST API, and WebSocket endpoint are in scope. General support questions and non-security bugs belong in the [issue tracker](https://github.com/cloudapp3/vminfo/issues/new/choose) or [SUPPORT.md](SUPPORT.md).
