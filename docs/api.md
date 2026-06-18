---
title: HTTP API
description: Read-only HTTP API and WebSocket endpoints exposed by the vminfo web dashboard.
---

# HTTP API

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
- `/healthz` remains public
- `/`, `/api/v1/*`, and `/ws` require the token or auth cookie
- token-protected mode does not expose permissive `Access-Control-Allow-Origin: *`
- WebSocket requests must use the same browser origin as the dashboard host

## Endpoints

### `GET /healthz`

Public health check for the web process.

```json
{
  "status": "ok",
  "ws_clients": 0
}
```

### `GET /api/v1/snapshot`

Returns the current full dashboard snapshot.

```json
{
  "timestamp": "2026-06-14T12:00:00Z",
  "system": {},
  "cpu": {},
  "memory": {},
  "disk": {},
  "network": {},
  "load": {},
  "processes": {},
  "health": {}
}
```

### `GET /api/v1/cpu`

Returns CPU totals, per-core usage, and short in-memory CPU history.

### `GET /api/v1/memory`

Returns memory and swap totals, usage, availability, and percentages.

### `GET /api/v1/disk`

Returns filesystem usage and disk I/O rates.

### `GET /api/v1/network`

Returns network throughput, TCP/UDP connection counts, and interface counters.

### `GET /api/v1/processes`

Returns the hydrated process list.

Supported query parameters:

| Parameter | Description |
| --- | --- |
| `filter` | Case-insensitive match against PID, PPID, name, command, user, or state |
| `q` | Alias for `filter` |
| `sort` | `cpu`, `mem`, `pid`, or `name`; defaults to `cpu` |
| `limit` | Maximum number of returned rows; `0` or omitted means no limit |

Example:

```bash
curl 'http://127.0.0.1:20021/api/v1/processes?filter=ssh&sort=mem&limit=10'
```

Response shape:

```json
{
  "total": 128,
  "list": [
    {
      "pid": 1234,
      "ppid": 1,
      "name": "sshd",
      "user": "root",
      "cpu_percent": 0.1,
      "mem_percent": 0.2,
      "rss": 12345678,
      "status": "S",
      "command": "sshd: user@pts/0",
      "threads": 1,
      "nice": 0,
      "uptime": 3600,
      "started_at_unix": 1781434800
    }
  ]
}
```

### `GET /api/v1/system`

Returns host metadata, OS/kernel/arch, CPU model/core count, and uptime.

### `GET /api/v1/health`

Returns the lightweight health score and warnings used by the dashboard.

```json
{
  "score": 90,
  "warnings": [
    {
      "level": "warning",
      "code": "disk_high",
      "message": "disk usage is 88.5%"
    }
  ]
}
```

### `GET /ws`

WebSocket stream of full dashboard snapshots.

- sends the latest snapshot immediately after connection
- streams refreshed snapshots as the collector updates
- in token-protected mode, the request must authenticate and pass same-origin checks

## See also

- [Web dashboard guide](/guide/web-dashboard)
- [Command reference](/commands/)
