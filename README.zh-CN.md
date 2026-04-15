# vminfo

> 一条命令看清机器上的一切 — CPU、内存、磁盘、网络、进程 — 终端 UI、JSON 或浏览器仪表盘，随你选。

[![CI](https://github.com/VPSMarket/vminfo/actions/workflows/ci.yml/badge.svg)](https://github.com/VPSMarket/vminfo/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/VPSMarket/vminfo.svg)](https://pkg.go.dev/github.com/VPSMarket/vminfo)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

English: [README.md](README.md)

## 演示

![vminfo demo](docs/assets/tui-demo.gif)

![vminfo overview](docs/assets/tui-overview.png)

## 快速开始

```bash
# 1. 安装
go install github.com/VPSMarket/vminfo/cmd/vminfo@latest

# 2. 运行 — 交互式 TUI
vminfo

# 3. 或者直接拿 JSON 快照用于脚本
vminfo summary --json
```

无需配置文件，无需守护进程，装完即用。

## 它能做什么

vminfo 让你对任意主机一目了然：

- **TUI** — 全屏、实时刷新的终端仪表盘，支持概览和进程视图
- **JSON** — 面向脚本、CI、监控流水线的机器可读输出
- **Web 仪表盘** — 浏览器 UI，带 REST 和 WebSocket 接口（`vminfo --web`）
- **Go 库** — 导入 `github.com/VPSMarket/vminfo`，嵌入你自己的工具

采集指标：CPU（每核）、内存、交换区、磁盘、磁盘 I/O、网络、负载、TCP/UDP 连接数、进程列表、温度、运行时间、主机元数据。

## 命令一览

```bash
vminfo                 # 启动 TUI
vminfo info            # TUI 别名
vminfo summary         # 单次快照（文本）
vminfo summary --json  # 单次快照（JSON）
vminfo watch           # 持续输出快照
vminfo watch --json    # 持续输出 JSON Lines
vminfo watch --count 1 # 输出一条样本后退出
vminfo --web           # 在 127.0.0.1:9990 启动 Web 仪表盘
vminfo --web --tui     # Web + TUI 同时运行
vminfo --web --bind 0.0.0.0 --port 8080
vminfo ps              # Linux-only 进程列表
vminfo ps --json       # 进程列表 JSON
vminfo ps --sort mem   # 按 cpu|mem|pid|name 排序
vminfo kill <pid>      # 向进程发送 SIGTERM（Linux）
vminfo version --json  # 构建元数据
vminfo --lang zh       # 切换 UI 语言
```

内置语言：`en`、`zh`、`de`、`es`、`fr`、`ja`、`ko`、`pt`、`ru`。

## Web 仪表盘

```bash
vminfo --web                      # 默认：127.0.0.1:9990
vminfo --web --bind 0.0.0.0 --port 9990 --interval 1s
```

接口：

- `GET /healthz` — 健康检查
- `GET /api/v1/snapshot` — 当前快照 JSON
- `GET /ws` — 实时 WebSocket 流

## TUI 快捷键

| 按键 | 功能 |
| --- | --- |
| `q` / `ctrl+c` | 退出 |
| `?` | 切换帮助 |
| `p` | 暂停 / 恢复 |
| `+` / `-` | 调整刷新间隔 |
| `r` | 立即刷新 |
| `tab` | 切换概览 / 进程视图 |
| `↑` / `↓` | 移动选择 |
| `s` | 切换排序 |
| `t` | 树视图 |
| `/` | 过滤进程 |
| `k` | SIGTERM 选中进程 |
| `enter` / `y` | 确认终止 |
| `esc` / `n` | 取消 |

状态徽标：`LIVE` · `PAUSED` · `LOADING` · `ERROR` · `STALE`

## 库用法

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

导出类型：`StaticInfo` · `RuntimeStats` · `ProcessInfo` · `Snapshot` · `AppMetadata`

## 平台兼容

| 能力 | Linux | macOS | Windows |
| --- | --- | --- | --- |
| `summary` / `watch` | ✅ | ✅ | ✅ |
| TUI | ✅ | ✅ | ✅ |
| Web 仪表盘 | ✅ | ✅ | ✅ |
| `ps` / `kill` | ✅ | ⚠️ stub | ⚠️ stub |

TUI 需要真实 TTY。`ps` 和 `kill` 按设计仅 Linux 可用。

## 从源码构建

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

## 开发

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
go vet ./...
go run ./cmd/vminfo summary --json
```

## 文档

- [README.md](README.md)
- [CHANGELOG.md](CHANGELOG.md)
- [docs/DESIGN-vminfo-bootstrap.md](docs/DESIGN-vminfo-bootstrap.md)

## License

[MIT](LICENSE)
