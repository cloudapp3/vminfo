# vminfo

> One command to see everything running on your machine — CPU, memory, disk, network, processes — in a polished terminal UI, JSON, or browser dashboard.

[![CI](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml/badge.svg)](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/cloudapp3/vminfo.svg)](https://pkg.go.dev/github.com/cloudapp3/vminfo)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

中文说明：[docs/README.zh-CN.md](docs/README.zh-CN.md)

## Demo

![vminfo demo](docs/assets/tui-demo.gif)

> Screens may vary slightly by terminal width, font, and theme.

| TUI overview | Web dashboard |
| --- | --- |
| ![vminfo overview refreshed](docs/assets/tui-overview-refreshed.png) | ![vminfo web dashboard](docs/assets/web-dashboard.png) |

| Processes | Help |
| --- | --- |
| ![vminfo processes](docs/assets/tui-processes.png) | ![vminfo help](docs/assets/tui-help.png) |

## Quick start

```bash
# 1. Install
go install github.com/cloudapp3/vminfo/cmd/vminfo@latest

# 2. Run — interactive TUI
vminfo

# 3. Or get a JSON snapshot for scripting
vminfo summary --json
```

That's it. No config files, no daemons, no setup.

## Install from GitHub Release

### One-line install (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

By default, the script installs `vminfo` into `~/.local/bin`.

### Install a specific version

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash -s -- --version v0.1.0
```

### Install to a custom directory

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash -s -- --dir /usr/local/bin
```

## What it does

vminfo gives you instant visibility into any host:

- **TUI** — full-screen, live-updating terminal dashboard with overview and process views
- **JSON** — machine-readable output for scripts, CI, monitoring pipelines
- **Web dashboard** — browser-based UI with REST and WebSocket endpoints (`vminfo --web`)
- **Go library** — import `github.com/cloudapp3/vminfo` and embed collection in your own tools

Collected metrics: CPU (per-core), memory, swap, disk, disk I/O, network, load, TCP/UDP counts, network interface totals/errors/drops, process list, temperatures, uptime, and host metadata.

## Network & Load panel

- **Load-aware coloring** — 1m / 5m / 15m load values are colored by `load / CPU cores`, with mini bars in wide layouts
- **Traffic split** — total throughput is separated from the per-interface table for faster scanning
- **Interface prioritization** — active interfaces and warning interfaces sort before idle bridges / veth devices
- **Noise reduction** — idle interfaces fold on narrow layouts, while public/private IPs and errors/drops stay visually distinct
- **Web parity** — the web dashboard mirrors the same network semantics: totals, sorting, IP styling, and interface warnings

## Commands

```bash
vminfo                 # launch TUI
vminfo info            # TUI alias
vminfo summary         # one runtime snapshot (text)
vminfo summary --json  # one runtime snapshot (JSON)
vminfo watch           # stream snapshots continuously
vminfo watch --json    # stream JSON lines
vminfo watch --count 1 # single sample then exit
vminfo --web           # web dashboard on 127.0.0.1:20021
vminfo --web --tui     # web + TUI together
vminfo --web --bind 0.0.0.0 --port 8080
vminfo ps              # Linux-only process list
vminfo ps --json       # processes as JSON
vminfo ps --sort mem   # sort by cpu|mem|pid|name
vminfo kill <pid>      # SIGTERM a process (Linux)
vminfo version --json  # build metadata
vminfo --lang zh       # switch UI language
```

Built-in languages: `en`, `zh`, `de`, `es`, `fr`, `ja`, `ko`, `pt`, `ru`.

## Web dashboard

```bash
vminfo --web                      # default: 127.0.0.1:20021
vminfo --web --bind 0.0.0.0 --port 8080 --interval 1s
```

When binding to all interfaces, startup output now shows friendlier URLs instead of only `0.0.0.0`:

```text
Web dashboard:
  Local  http://127.0.0.1:20021
  Public http://203.0.113.10:20021   # when a public IPv4 is present
  LAN    http://192.168.1.23:20021   # fallback when only a LAN IPv4 is present
```

Web mode keeps stdout quiet during normal browsing: routine HTTP request logs and WebSocket connect/disconnect logs are suppressed, while real startup and error messages are still shown.

Endpoints:

- `GET /healthz` — health check
- `GET /api/v1/snapshot` — current snapshot JSON
- `GET /ws` — live WebSocket stream

Recent web UI polish:

- Larger overall type scale for easier browser reading
- Resource progress bars use grouped spacing and segmented tracks
- CPU right-side block is vertically centered in the Resources card
- Per-core CPU bars are larger and the extra `avg` footer has been removed

## TUI controls

| Key | Action |
| --- | --- |
| `q` / `ctrl+c` | Quit |
| `?` | Toggle help |
| `p` | Pause / resume |
| `+` / `-` | Adjust interval |
| `r` | Refresh now |
| `tab` | Switch overview / processes |
| `↑` / `↓` | Move selection |
| `s` | Cycle sort |
| `t` | Tree view |
| `/` | Filter processes |
| `k` | SIGTERM selected process |
| `enter` / `y` | Confirm kill |
| `esc` / `n` | Cancel |

Status badges: `LIVE` · `PAUSED` · `LOADING` · `ERROR` · `STALE`

## Library usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/cloudapp3/vminfo"
)

func main() {
    ctx := context.Background()

    static, _ := vminfo.CollectStatic(ctx)
    stats, _ := vminfo.CollectStats(ctx, vminfo.Options{SampleInterval: time.Second})

    fmt.Println(static.Hostname, stats.CPU)
}
```

Exported types: `StaticInfo` · `RuntimeStats` · `ProcessInfo` · `Snapshot` · `AppMetadata`

## Platform support

| Capability | Linux | macOS | Windows |
| --- | --- | --- | --- |
| `summary` / `watch` | ✅ | ✅ | ✅ |
| TUI | ✅ | ✅ | ✅ |
| Web dashboard | ✅ | ✅ | ✅ |
| `ps` / `kill` | ✅ | ⚠️ stub | ⚠️ stub |

TUI requires a real TTY. `ps` and `kill` are Linux-only by design.

## Build from source

```bash
git clone https://github.com/cloudapp3/vminfo.git
cd vminfo
go build -ldflags "\
  -X github.com/cloudapp3/vminfo.Version=v0.1.0 \
  -X github.com/cloudapp3/vminfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/cloudapp3/vminfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/cloudapp3/vminfo.Channel=stable" \
  ./cmd/vminfo
```

## Development

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
go vet ./...
go run ./cmd/vminfo summary --json
```

## Documentation

- [docs/CHANGELOG.md](docs/CHANGELOG.md)

## License

[MIT](LICENSE)
