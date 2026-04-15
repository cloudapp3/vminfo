# vminfo

> One command to see everything running on your machine — CPU, memory, disk, network, processes — in a polished terminal UI, JSON, or browser dashboard.

[![CI](https://github.com/VPSMarket/vminfo/actions/workflows/ci.yml/badge.svg)](https://github.com/VPSMarket/vminfo/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/VPSMarket/vminfo.svg)](https://pkg.go.dev/github.com/VPSMarket/vminfo)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

中文说明：[README.zh-CN.md](README.zh-CN.md)

## Demo

![vminfo demo](docs/assets/tui-demo.gif)

![vminfo overview](docs/assets/tui-overview.png)

## Quick start

```bash
# 1. Install
go install github.com/VPSMarket/vminfo/cmd/vminfo@latest

# 2. Run — interactive TUI
vminfo

# 3. Or get a JSON snapshot for scripting
vminfo summary --json
```

That's it. No config files, no daemons, no setup.

## What it does

vminfo gives you instant visibility into any host:

- **TUI** — full-screen, live-updating terminal dashboard with overview and process views
- **JSON** — machine-readable output for scripts, CI, monitoring pipelines
- **Web dashboard** — browser-based UI with REST and WebSocket endpoints (`vminfo --web`)
- **Go library** — import `github.com/VPSMarket/vminfo` and embed collection in your own tools

Collected metrics: CPU (per-core), memory, swap, disk, disk I/O, network, load, TCP/UDP counts, process list, temperatures, uptime, and host metadata.

## Commands

```bash
vminfo                 # launch TUI
vminfo info            # TUI alias
vminfo summary         # one runtime snapshot (text)
vminfo summary --json  # one runtime snapshot (JSON)
vminfo watch           # stream snapshots continuously
vminfo watch --json    # stream JSON lines
vminfo watch --count 1 # single sample then exit
vminfo --web           # web dashboard on 127.0.0.1:9990
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
vminfo --web                      # default: 127.0.0.1:9990
vminfo --web --bind 0.0.0.0 --port 9990 --interval 1s
```

Endpoints:

- `GET /healthz` — health check
- `GET /api/v1/snapshot` — current snapshot JSON
- `GET /ws` — live WebSocket stream

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

    "github.com/VPSMarket/vminfo"
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
git clone https://github.com/VPSMarket/vminfo.git
cd vminfo
go build -ldflags "\
  -X github.com/VPSMarket/vminfo.Version=v0.1.0 \
  -X github.com/VPSMarket/vminfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/VPSMarket/vminfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/VPSMarket/vminfo.Channel=stable" \
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

- [CHANGELOG.md](CHANGELOG.md)
- [docs/DESIGN-vminfo-bootstrap.md](docs/DESIGN-vminfo-bootstrap.md)

## License

[MIT](LICENSE)
