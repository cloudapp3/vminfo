# vminfo - cross-platform terminal system monitor, web dashboard, and Go library

> A single-binary system monitoring toolkit for Linux, macOS, and Windows. Inspect CPU, memory, disk, network, and load in a live terminal UI, export JSON for automation, open a browser dashboard, or embed host metrics in Go. No background agent or configuration is required for local monitoring.

[![CI](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml/badge.svg)](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/cloudapp3/vminfo?display_name=tag)](https://github.com/cloudapp3/vminfo/releases/latest)
[![GitHub Downloads](https://img.shields.io/github/downloads/cloudapp3/vminfo/total.svg)](https://github.com/cloudapp3/vminfo/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/cloudapp3/vminfo.svg)](https://pkg.go.dev/github.com/cloudapp3/vminfo)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Documentation: [vminfo documentation](https://vminfo.bestcheapvps.org) · [中文说明](https://vminfo.bestcheapvps.org/zh/) · [HTTP API reference](https://vminfo.bestcheapvps.org/api) · [Docs source](https://github.com/cloudapp3/vmdocs)

[Preview](#preview) · [Quick start](#quick-start) · [Why vminfo](#why-vminfo) · [Commands](#commands) · [Platform support](#platform-support) · [FAQ](#faq) · [Contributing](#contributing)

## Preview

![vminfo terminal system monitor demo](https://raw.githubusercontent.com/cloudapp3/vmdocs/main/sites/vminfo/docs/assets/tui-demo.gif)

> The demo cycles through the overview, Linux process view, and help. Screens may vary slightly by terminal width, font, and theme.

| Web dashboard | Linux process view |
| --- | --- |
| ![vminfo browser system monitoring dashboard](https://raw.githubusercontent.com/cloudapp3/vmdocs/main/sites/vminfo/docs/assets/web-dashboard.png) | ![vminfo Linux process monitor](https://raw.githubusercontent.com/cloudapp3/vmdocs/main/sites/vminfo/docs/assets/tui-processes.png) |

## Quick start

```bash
# 1. Install - one-line script (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | sudo bash -s -- --dir /usr/local/bin

# Or install without sudo to an auto-detected user directory
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash

# 2. Run - interactive TUI
vminfo

# 3. Or get a JSON snapshot for scripting
vminfo summary --json
```

That's it. No background agent, config file, or setup is required for local monitoring. The installer downloads the matching GitHub Release and verifies its SHA-256 checksum by default.

The install script auto-selects a directory when `--dir` is not set: `/usr/local/bin` → `~/.local/bin` → `~/bin`.

### Other installation methods

| Method | Platforms | Instructions |
| --- | --- | --- |
| Release archive | Linux, macOS, Windows | Download the matching `.tar.gz` or `.zip` from [GitHub Releases](https://github.com/cloudapp3/vminfo/releases/latest). |
| Linux package | Debian/Ubuntu, Fedora/RHEL | Download the generated `.deb` or `.rpm` from [GitHub Releases](https://github.com/cloudapp3/vminfo/releases/latest). |
| Go toolchain | Linux, macOS, Windows | Run `go install github.com/cloudapp3/vminfo/cmd/vminfo@latest`. |

Windows release builds are available for `amd64`. Extract `vminfo.exe` from the release ZIP and place it in a directory on your `PATH`.

Custom Linux/macOS install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash -s -- --dir /opt/bin
```

If you install to a custom directory such as `/opt/bin`, make sure that directory is in your `PATH`, or symlink the binary into `/usr/local/bin`:

```bash
sudo ln -sf /opt/bin/vminfo /usr/local/bin/vminfo
```

## Why vminfo

vminfo is built for developers, SREs, DevOps engineers, and server operators who want fast, low-friction visibility into host metrics without installing a monitoring stack.

Use vminfo when you need to:

- monitor CPU, memory, disk, network, and load from a live terminal dashboard
- inspect and manage Linux processes without switching tools
- export stable JSON snapshots or JSON Lines for scripts, CI, and automation
- open a lightweight browser dashboard with `vminfo --web`
- embed host metrics collection or the TUI into your own Go tools

The same binary provides four interfaces:

- **Terminal UI** - full-screen, live-updating overview and process views
- **JSON and text CLI** - one-shot or streaming output for automation
- **Web dashboard** - browser UI with REST and WebSocket endpoints
- **Go library** - public collection APIs plus an embeddable TUI package

Collected metrics include CPU per core, memory, swap, disk, disk I/O, network, load, TCP/UDP counts, TCP state distribution, conntrack usage, interface rates, processes, temperatures, uptime, and host metadata.

If vminfo is useful on your machines, [star the repository](https://github.com/cloudapp3/vminfo) to help other operators discover it.

## vminfo compared with other system monitors

| Project | Primary use | Where vminfo differs |
| --- | --- | --- |
| htop / btop | Interactive local process and resource monitoring | vminfo also provides scriptable JSON, a browser dashboard, and embeddable Go APIs from one binary. |
| Glances | Python-based terminal, export, API, and web monitoring | vminfo focuses on a self-contained Go binary and public Go packages. |
| Netdata | Always-on monitoring agent and web observability platform | vminfo can run on demand without installing a background service. |
| gopsutil | Go library for host and process metrics | vminfo adds a ready-to-run CLI, TUI, web dashboard, and opinionated output contracts. |

These tools serve different operational needs. See the [detailed comparison guide](https://vminfo.bestcheapvps.org/compare/) before choosing for production use.

## Cross-platform monitoring features

- **Host resources** - CPU, memory, swap, disk, disk I/O, temperature, uptime, and metadata
- **Network visibility** - total throughput, per-interface rates, IP addresses, TCP/UDP counts, TCP states, and Linux conntrack usage
- **Host health** - resource score and focused warnings exposed in both the dashboard and `/api/v1/health`
- **Network diagnostics** - DNS, TCP port, TCP/ICMP ping, and public IP/ASN/geo lookup commands
- **International UI** - built-in English, Chinese, German, Spanish, French, Japanese, Korean, Portuguese, and Russian translations

<details>
<summary>Network and host-health details</summary>

### Network & Load panel

- **Load-aware coloring** - 1m / 5m / 15m load values are colored by `load / CPU cores`, with mini bars in wide layouts
- **Traffic split** - total throughput is separated from the per-interface table for faster scanning
- **Interface prioritization** - active interfaces sort before idle bridges / veth devices
- **Noise reduction** - idle interfaces fold on narrow layouts, while public/private IPs stay visually distinct
- **Web parity** - the web dashboard mirrors the same network semantics: totals, sorting, and IP styling
- **Connection states** - TCP sockets broken down by state (ESTABLISHED, TIME_WAIT, SYN_RECV, …) in both TUI and web
- **Conntrack usage** - current/max `nf_conntrack` entries with a saturation gauge (Linux)

### Network health warnings

The host health score (`/api/v1/health`, health panel) includes network signals, so a quietly degrading link is no longer invisible:

- `network_errors` - sustained per-interface error rate (events/s, not cumulative counters)
- `network_drops` - sustained packet-drop rate
- `tcpconn_high` - unusually high TCP socket count (≥5000 warn / ≥20000 critical)
- `conntrack_high` - conntrack table filling up (≥85% warn / ≥95% critical)

Rates - not raw counters - gate `network_errors` / `network_drops`, so a long-lived total does not keep an otherwise-healthy host flagged.

</details>

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
vminfo --web --token   # auto-generate a dashboard token
vminfo --web --token secret-token
vminfo --web --tui     # web + TUI together
vminfo --web --bind 0.0.0.0 --port 8080 --token
vminfo ps              # Linux-only process list
vminfo ps nginx        # filter by name, user, pid, or command
vminfo ps --filter ssh # explicit process filter for scripts
vminfo ps --tree       # render a process tree
vminfo ps --watch      # refresh process table continuously
vminfo ps --limit 20   # show the first 20 rows after sort/filter
vminfo ps --json       # processes as JSON
vminfo ps --sort mem   # sort by cpu|mem|pid|name
vminfo kill <pid>      # SIGTERM a process (Linux)
vminfo net dns vminfo.bestcheapvps.org            # resolve a domain (system or --server)
vminfo net port vminfo.bestcheapvps.org 443       # test TCP port connectivity / latency
vminfo net ping vminfo.bestcheapvps.org --tcp-port 443   # TCP ping (default; cross-platform)
vminfo net ping vminfo.bestcheapvps.org --mode icmp      # real ICMP ping (needs privileges)
vminfo net ip                         # your public IP + ASN / geo
vminfo net ip 8.8.8.8                 # lookup a specific IP
vminfo update          # check + install the latest tagged release
vminfo update --check  # check without installing
vminfo update --version vX.Y.Z
vminfo --lang zh       # switch UI language
```

Built-in languages: `en`, `zh`, `de`, `es`, `fr`, `ja`, `ko`, `pt`, `ru`.

`net` subcommand flags may appear before or after the target, so both
`net ping --tcp-port 443 example.com` and `net ping example.com --tcp-port 443`
are accepted. CLI ping count is limited to 1-100 and probe timeouts must be
positive and no greater than 10 seconds.

## Platform support

| Capability | Linux | macOS | Windows |
| --- | --- | --- | --- |
| `summary` / `watch` | ✅ | ✅ | ✅ |
| TUI | ✅ | ✅ | ✅ |
| Web dashboard | ✅ | ✅ | ✅ |
| `ps` / `kill` | ✅ | ⚠️ stub | ⚠️ stub |
| `update --check` | ✅ | ✅ | ✅ |
| `update` install | ✅ | ✅ | ⚠️ check-only |

TUI requires a real TTY. `ps` and `kill` are Linux-only by design.

## Web dashboard

```bash
vminfo --web                      # default: 127.0.0.1:20021
vminfo --web --token             # auto-generate a token and print a ready-to-open URL
vminfo --web --token my-token    # use a fixed token
vminfo --web --bind 0.0.0.0 --port 8080 --interval 1s --token
```

Add `--token` when you want to protect the dashboard in a browser:

- `--token some-value` uses that exact token
- bare `--token` auto-generates a URL-safe token
- the first successful `/?token=...` visit sets a cookie, so later page/API/WebSocket requests can continue without keeping the token in the address bar

Loopback binds (`127.0.0.1`, `::1`, or `localhost`) may run without a token.
Any non-loopback bind, including `0.0.0.0`, requires an explicit `--token` or
bare `--token` request. An empty `--bind=` is rejected rather than interpreted
as all interfaces.

The built-in server uses HTTP. A token controls access but does not encrypt the
connection. For remote access, put vminfo behind an HTTPS reverse proxy or use
an SSH tunnel; do not expose its HTTP port directly to an untrusted network.

When binding to all interfaces with the required token, startup output shows
friendlier, ready-to-open URLs instead of only `0.0.0.0`:

```text
Web dashboard:
  Local  http://127.0.0.1:20021/?token=example-token
  Public http://203.0.113.10:20021/?token=example-token   # when a public IPv4 is present
  LAN    http://192.168.1.23:20021/?token=example-token   # fallback when only a LAN IPv4 is present
```

Token-protected printed URLs always include `?token=...` so they can be opened directly.

Web mode keeps stdout quiet during normal browsing: routine HTTP request logs and WebSocket connect/disconnect logs are suppressed, while real startup and error messages are still shown.

The web server always enforces browser same-origin access:

- when a token is enabled, dashboard pages, JSON APIs, and `/ws` require the token or auth cookie
- permissive `Access-Control-Allow-Origin: *` is never exposed
- REST requests and WebSocket upgrades with an `Origin` header must match the dashboard host
- unauthenticated loopback mode only accepts `localhost` or loopback-IP Host headers, preventing DNS rebinding
- native clients without an `Origin` header remain supported

Endpoints:

- `GET /api/v1/snapshot` — current snapshot JSON
- `GET /api/v1/processes` — process list with optional `filter` / `q`, `sort`, and `limit` query parameters
- `GET /api/v1/health` — lightweight health score and resource warnings
- `GET /ws` — live WebSocket stream
- `POST /api/v1/net/diag` — run a same-origin network diagnostic (`{"action":"dns|port|ping|ip","target":...}`); ping count is limited to 10 and per-probe timeout to 3 seconds

The dashboard ships with switchable themes (Auto / Neon / Light / Terminal / Synthwave) from the header; "Auto" follows the OS color scheme. JetBrains Mono is **embedded**, so the dashboard stays self-contained and works offline with no external font requests.

## Go library

Collect host metrics from your own Go program:

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

Launch the same interactive terminal UI from another Go CLI:

```go
package main

import (
    "context"
    "log"

    vminfotui "github.com/cloudapp3/vminfo/tui"
)

func main() {
    if err := vminfotui.Run(context.Background(), vminfotui.Options{Lang: "en"}); err != nil {
        log.Fatal(err)
    }
}
```

`tui.Options` also accepts custom `Stdin` and `Stdout` streams for embedded CLIs and tests.

Public packages: [github.com/cloudapp3/vminfo](https://pkg.go.dev/github.com/cloudapp3/vminfo) · [github.com/cloudapp3/vminfo/tui](https://pkg.go.dev/github.com/cloudapp3/vminfo/tui)

Exported collection types: `StaticInfo` · `RuntimeStats` · `ProcessInfo` · `Snapshot` · `AppMetadata`

## Self-update

Release builds can update themselves from GitHub Releases:

```bash
vminfo update
vminfo update --check
vminfo update --version vX.Y.Z
```

See the [changelog](https://vminfo.bestcheapvps.org/changelog) for release-specific changes.

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
| `K` | Show / hide Linux kernel threads |
| `enter` / `y` | Confirm kill |
| `esc` / `n` | Cancel |

Status badges: `LIVE` · `PAUSED` · `LOADING` · `ERROR` · `STALE`

## FAQ

### Does vminfo require a daemon or configuration file?

No background service or configuration file is required for the TUI, `summary`, `watch`, or network diagnostics. Web mode starts a foreground HTTP server only when you request `vminfo --web`.

### Does vminfo require root privileges?

Normal monitoring commands do not require root. Installing into a protected directory, sending signals to other users' processes, and ICMP ping may require elevated OS permissions.

### Which features work on Windows and macOS?

The TUI, `summary`, `watch`, web dashboard, and update checks are cross-platform. `ps` and `kill` are Linux-only, and Windows self-update is currently check-only. See [Platform support](#platform-support).

### Can I use vminfo in scripts and CI?

Yes. Use `vminfo summary --json` for one snapshot or `vminfo watch --json` for a stream of JSON Lines.

### How should I access the web dashboard remotely?

Keep the default loopback bind when possible. Non-loopback binds require a token, but the built-in server is HTTP only; use an HTTPS reverse proxy or SSH tunnel for remote access.

### Is vminfo a replacement for htop or btop?

It overlaps with local resource and Linux process monitoring, but also targets JSON automation, browser access, and Go embedding. Choose based on the workflow summarized in [the comparison](#vminfo-compared-with-other-system-monitors).

### How do I update vminfo?

Run `vminfo update` from a tagged Linux or macOS release build. Use `vminfo update --check` to inspect available versions without installing.

## Community & Support

- Read the [documentation](https://vminfo.bestcheapvps.org) or [support guide](SUPPORT.md)
- Join the [VMPulse Telegram group](https://t.me/VMPulse) for usage questions and feedback
- [Open a structured issue](https://github.com/cloudapp3/vminfo/issues/new/choose) for a reproducible bug or feature request
- Follow [SECURITY.md](SECURITY.md) to report a vulnerability privately
- Browse [GitHub Releases](https://github.com/cloudapp3/vminfo/releases) for binaries, packages, checksums, and release notes

## Contributing

Contributions are welcome - bug reports, feature ideas, documentation improvements, tests, platform compatibility fixes, and pull requests.

If you want to help:

1. [Open an issue](https://github.com/cloudapp3/vminfo/issues/new/choose) to discuss a bug, feature, or non-trivial change
2. Read [CONTRIBUTING.md](CONTRIBUTING.md)
3. Fork the repository and make a focused change
4. Run `go test ./...`, `go test -race ./...`, and `go vet ./...`
5. Open a pull request

Questions before opening a PR? Join [Telegram](https://t.me/VMPulse).

## Build from source

```bash
git clone https://github.com/cloudapp3/vminfo.git
cd vminfo
go build -ldflags "\
  -X github.com/cloudapp3/vminfo.Version=vX.Y.Z \
  -X github.com/cloudapp3/vminfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/cloudapp3/vminfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/cloudapp3/vminfo.Channel=stable" \
  ./cmd/vminfo
```

## Development

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/vminfo summary --json
```

## Documentation

- [Documentation website](https://vminfo.bestcheapvps.org)
- [Chinese documentation](https://vminfo.bestcheapvps.org/zh/)
- [HTTP API reference](https://vminfo.bestcheapvps.org/api)
- [Changelog](https://vminfo.bestcheapvps.org/changelog)
- [Feature benchmark and roadmap](https://vminfo.bestcheapvps.org/roadmap/feature-benchmark)
- [Documentation source repository](https://github.com/cloudapp3/vmdocs)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## License

[MIT](LICENSE)
