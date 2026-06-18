---
title: Web Dashboard
description: Start the read-only HTTP API and browser dashboard, including token auth and websocket access.
---

# Web Dashboard

`vminfo --web` starts a lightweight, read-only HTTP API and dashboard.

## Start the server

```bash
vminfo --web
```

Default address:

```text
http://127.0.0.1:20021
```

Custom address:

```bash
vminfo --web --bind 0.0.0.0 --port 8080 --interval 1s
```

You can also launch the TUI alongside the dashboard:

```bash
vminfo --web --tui
```

## Authentication

By default, the dashboard and API are local and unauthenticated.

When `--token` is enabled:

```bash
vminfo --web --token
vminfo --web --token my-secret
```

- bare `--token` auto-generates a URL-safe token
- `--token my-secret` uses a fixed token
- the first successful `/?token=...` visit sets a cookie for later requests
- `/healthz` remains public for local probes
- `/`, `/api/v1/*`, and `/ws` require the token or auth cookie
- token-protected mode does not expose permissive `Access-Control-Allow-Origin: *`
- WebSocket upgrades require the browser origin to match the dashboard host

::: warning Remote access
When binding to `0.0.0.0`, enable `--token` unless the dashboard is only exposed to a trusted private network.
:::

## Endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Public health check |
| `GET /api/v1/snapshot` | Full runtime snapshot |
| `GET /api/v1/cpu` | CPU data |
| `GET /api/v1/memory` | Memory and swap data |
| `GET /api/v1/disk` | Filesystem and disk I/O data |
| `GET /api/v1/network` | Network totals and interface data |
| `GET /api/v1/processes` | Process list |
| `GET /api/v1/system` | Host metadata |
| `GET /api/v1/health` | Health score and warnings |
| `GET /ws` | Live snapshot stream |

See the full [HTTP API reference](/api) for payload examples and query parameters.
