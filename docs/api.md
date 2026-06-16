# vminfo HTTP API

`vminfo --web` starts a lightweight, read-only HTTP API and dashboard.

Default address:

```bash
vminfo --web
# http://127.0.0.1:20021
```

Custom address:

```bash
vminfo --web --bind 0.0.0.0 --port 8080
```

## Authentication

By default, the dashboard and API are local and unauthenticated.

When `--token` is enabled:

```bash
vminfo --web --token
vminfo --web --token my-secret
```

- `/healthz` remains public for local health probes.
- `/`, `/api/v1/*`, and `/ws` require either `?token=...` or the auth cookie set after a successful token visit.
- Token-protected mode does not expose permissive `Access-Control-Allow-Origin: *`.
- WebSocket requests must use the same browser origin as the dashboard host.

## Endpoints

### `GET /healthz`

Public health check for the web process.

Example response:

```json
{
  "status": "ok",
  "ws_clients": 0
}
```

### `GET /api/v1/snapshot`

Returns the current full dashboard snapshot.

Top-level shape:

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

Returns aggregate filesystem usage and disk I/O rates.

### `GET /api/v1/network`

Returns total network throughput, TCP/UDP connection counts, and interface counters.

### `GET /api/v1/processes`

Returns the hydrated process list.

Supported query parameters:

| Parameter | Description |
| --- | --- |
| `filter` | Case-insensitive match against PID, PPID, name, command, user, or state. |
| `q` | Alias for `filter`. |
| `sort` | `cpu`, `mem`, `pid`, or `name`. Defaults to `cpu`. |
| `limit` | Maximum number of returned rows. `0` or omitted means no limit. |

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

`total` is the total process count before API-side filtering and limiting; `list` is the returned page after filtering, sorting, and limiting.

### `GET /api/v1/system`

Returns host metadata, OS/kernel/arch, CPU model/core count, and uptime.

### `GET /api/v1/health`

Returns the lightweight health score and warnings used by the dashboard Health Summary card.

Example response:

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

The score is intentionally simple and explainable:

- starts at `100`
- subtracts `10` for each warning
- subtracts `20` for each critical warning
- floors at `0`

### `GET /ws`

WebSocket stream of full dashboard snapshots.

- Sends the latest snapshot immediately after connection.
- Streams refreshed snapshots as the collector updates.
- In token-protected mode, the request must authenticate and pass same-origin checks.

