# vminfo — 终端系统监控工具、Web 仪表盘与 Go 库

> 面向 Linux、macOS 和 Windows 的跨平台系统监控工具。用终端 UI、JSON 输出或浏览器仪表盘，快速查看 CPU、内存、磁盘、网络、负载与进程。

[![CI](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml/badge.svg)](https://github.com/cloudapp3/vminfo/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/cloudapp3/vminfo.svg)](https://pkg.go.dev/github.com/cloudapp3/vminfo)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](../LICENSE)

English: [../README.md](../README.md)

[快速开始](#快速开始) · [预览](#预览) · [加入 Telegram 群组](https://t.me/VMPulse) · [提交 Issue](https://github.com/cloudapp3/vminfo/issues/new) · [参与贡献](#参与贡献)

## 快速开始

```bash
# 1. 一键安装（Linux/macOS）
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash

# 或者加 sudo 装到 /usr/local/bin
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | sudo bash

# 2. 运行 — 交互式 TUI
vminfo

# 3. 或者直接拿 JSON 快照用于脚本
vminfo summary --json
```

无需配置文件，无需守护进程，装完即用。

脚本按以下顺序自动选择安装目录：`/usr/local/bin` → `~/.local/bin` → `~/bin`。

其他安装方式：

```bash
# 安装到指定目录
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash -s -- --dir /opt/bin

# 从源码安装
go install github.com/cloudapp3/vminfo/cmd/vminfo@latest
```

如果你在使用过程中遇到问题、想提需求，或者希望交流想法，欢迎加入 [VMPulse Telegram 群组](https://t.me/VMPulse) 或直接 [提交 Issue](https://github.com/cloudapp3/vminfo/issues/new)。

## 适合谁 / 适用场景

vminfo 适合开发者、SRE、DevOps 工程师和服务器运维人员，用最低的接入成本快速查看主机实时指标。

适合这些场景：

- 在终端中实时查看 CPU、内存、磁盘、网络和负载
- 快速排查 Linux 进程占用情况
- 输出 JSON 供脚本、CI 或自动化流程消费
- 用 `vminfo --web` 打开轻量级浏览器监控面板
- 在自己的 Go 工具里嵌入主机指标采集能力

## 预览

![vminfo preview](assets/tui-demo.gif)

> 截图会随终端宽度、字体与主题略有差异。

| TUI 概览 | Web 仪表盘 |
| --- | --- |
| ![vminfo overview refreshed](assets/tui-overview-refreshed.png) | ![vminfo web dashboard](assets/web-dashboard.png) |

| 进程 | 帮助 |
| --- | --- |
| ![vminfo processes](assets/tui-processes.png) | ![vminfo help](assets/tui-help.png) |

## 它能做什么

vminfo 是一个跨平台系统监控工具，帮助你快速看清机器当前状态：

- **TUI** — 全屏、实时刷新的终端仪表盘，支持概览和进程视图
- **JSON** — 面向脚本、CI、监控流水线的机器可读输出
- **Web 仪表盘** — 浏览器 UI，带 REST 和 WebSocket 接口（`vminfo --web`）
- **Go 库** — 导入 `github.com/cloudapp3/vminfo` 做指标采集，或导入 `github.com/cloudapp3/vminfo/tui` 嵌入交互式终端 UI

采集指标：CPU（每核）、内存、交换区、磁盘、磁盘 I/O、网络、负载、TCP/UDP 连接数、网卡累计流量/错误/丢包、进程列表、温度、运行时间、主机元数据。

## Network & Load 面板

- **负载着色** — 1m / 5m / 15m 按 `load / CPU cores` 着色，宽屏下附带 mini bar
- **Traffic 分区** — 总吞吐与逐网卡表格分开显示，扫一眼就能看出当前总流量
- **接口优先级** — 活跃接口、异常接口优先展示，再排桥接 / docker / veth 等空闲接口
- **降噪处理** — 窄屏自动折叠空闲接口，同时保留公网/私网 IP 和 errors/drops 的视觉区分
- **Web 对齐** — Web 仪表盘也同步采用相同的网络语义：总流量、排序、公私网 IP 样式与网卡告警

## 命令一览

```bash
vminfo                 # 启动 TUI
vminfo info            # TUI 别名
vminfo summary         # 单次快照（文本）
vminfo summary --json  # 单次快照（JSON）
vminfo watch           # 持续输出快照
vminfo watch --json    # 持续输出 JSON Lines
vminfo watch --count 1 # 输出一条样本后退出
vminfo --web           # 在 127.0.0.1:20021 启动 Web 仪表盘
vminfo --web --token   # 自动生成仪表盘 token
vminfo --web --token secret-token
vminfo --web --tui     # Web + TUI 同时运行
vminfo --web --bind 0.0.0.0 --port 8080
vminfo ps              # Linux-only 进程列表
vminfo ps nginx        # 按名称、用户、PID 或命令过滤
vminfo ps --filter ssh # 面向脚本的显式过滤参数
vminfo ps --tree       # 渲染进程树
vminfo ps --watch      # 持续刷新进程表
vminfo ps --limit 20   # 排序 / 过滤后只显示前 20 行
vminfo ps --json       # 进程列表 JSON
vminfo ps --sort mem   # 按 cpu|mem|pid|name 排序
vminfo kill <pid>      # 向进程发送 SIGTERM（Linux）
vminfo update          # 检查并安装最新 tag 版本
vminfo update --check  # 只检查，不安装
vminfo update --version v0.1.0
vminfo --lang zh       # 切换 UI 语言
```

内置语言：`en`、`zh`、`de`、`es`、`fr`、`ja`、`ko`、`pt`、`ru`。

## Web 仪表盘

```bash
vminfo --web                      # 默认：127.0.0.1:20021
vminfo --web --token             # 自动生成 token，并打印可直接打开的 URL
vminfo --web --token my-token    # 使用固定 token
vminfo --web --bind 0.0.0.0 --port 8080 --interval 1s
```

如果你希望浏览器访问 Web 仪表盘时带鉴权，可以加上 `--token`：

- `--token some-value`：使用你指定的固定 token
- 裸写 `--token`：自动生成一个 URL-safe token
- 第一次成功访问 `/?token=...` 后会写入 cookie，后续页面 / API / WebSocket 请求不必一直把 token 留在地址栏里
- `GET /healthz` 仍保持公开，方便本地探针或健康检查继续使用

当使用 `0.0.0.0` 绑定全部网卡时，启动输出会显示更友好的可访问地址，而不是只打印 `0.0.0.0`：

```text
Web dashboard:
  Local  http://127.0.0.1:20021
  Public http://203.0.113.10:20021   # 机器存在公网 IPv4 时显示
  LAN    http://192.168.1.23:20021   # 没有公网 IPv4 时回退到局域网地址
```

开启 token 后，启动输出里的 URL 会自动带上 `?token=...`，可直接复制打开：

```text
Web dashboard: http://127.0.0.1:20021/?token=secret-token
```

Web 模式下普通浏览访问保持安静：默认不再输出 HTTP 访问日志和 WebSocket 连接/断开日志，只保留真正有用的启动信息和错误信息。

开启 Web 鉴权后，浏览器访问规则也会更严格：

- 仪表盘页面、JSON API 和 `/ws` 都需要 token 或已写入的 auth cookie
- token 保护模式下不会再暴露宽松的 `Access-Control-Allow-Origin: *`
- WebSocket 升级要求浏览器 `Origin` 与仪表盘 host 一致

接口：

- `GET /healthz` — 健康检查
- `GET /api/v1/snapshot` — 当前快照 JSON
- `GET /api/v1/processes` — 进程列表，支持 `filter` / `q`、`sort`、`limit` 查询参数
- `GET /api/v1/health` — 轻量健康评分和资源告警
- `GET /ws` — 实时 WebSocket 流

## 更新命令

Release 构建可直接从 GitHub Releases 自更新：

```bash
vminfo update
vminfo update --check
vminfo update --version v0.1.0
```

近期 Web UI 微调：

- 整体字体放大一档，更适合浏览器查看
- 资源进度条改为分段轨道，并增加组间留白
- Resources 卡片右侧 CPU 区整体垂直居中
- 每核 CPU 柱图进一步放大，并去掉额外的 `avg` 尾部块

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
| `K` | 显示 / 隐藏 Linux 内核线程 |
| `enter` / `y` | 确认终止 |
| `esc` / `n` | 取消 |

状态徽标：`LIVE` · `PAUSED` · `LOADING` · `ERROR` · `STALE`

## 支持作为库 import 使用

在自己的 Go 程序里采集主机指标：

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

在另一个 Go CLI 中启动同款交互式 TUI：

```go
package main

import (
    "context"
    "log"

    vminfotui "github.com/cloudapp3/vminfo/tui"
)

func main() {
    if err := vminfotui.Run(context.Background(), vminfotui.Options{Lang: "zh"}); err != nil {
        log.Fatal(err)
    }
}
```

`tui.Options` 也支持传入自定义 `Stdin` 和 `Stdout`，方便嵌入其他 CLI 或测试。

公共包：`github.com/cloudapp3/vminfo` · `github.com/cloudapp3/vminfo/tui`

采集相关导出类型：`StaticInfo` · `RuntimeStats` · `ProcessInfo` · `Snapshot` · `AppMetadata`

## 平台兼容

| 能力 | Linux | macOS | Windows |
| --- | --- | --- | --- |
| `summary` / `watch` | ✅ | ✅ | ✅ |
| TUI | ✅ | ✅ | ✅ |
| Web 仪表盘 | ✅ | ✅ | ✅ |
| `ps` / `kill` | ✅ | ⚠️ stub | ⚠️ stub |
| `update --check` | ✅ | ✅ | ✅ |
| `update` 安装 | ✅ | ✅ | ⚠️ 仅检查 |

TUI 需要真实 TTY。`ps` 和 `kill` 按设计仅 Linux 可用。

## 社区与支持

- 💬 加入 Telegram 群组：[t.me/VMPulse](https://t.me/VMPulse)
- 🐛 发现 bug 或想提新功能？[提交 Issue](https://github.com/cloudapp3/vminfo/issues/new)
- 📚 想先看说明文档？见 [文档](#文档)
- 🤝 想参与项目？从 [CONTRIBUTING.md](../CONTRIBUTING.md) 开始

你的反馈、问题报告和需求建议都会直接帮助 vminfo 的后续路线。

## 参与贡献

欢迎各种形式的贡献：bug 报告、功能建议、文档优化、测试补充、平台兼容修复，以及 Pull Request。

如果你想参与：

1. 先通过 [Issue](https://github.com/cloudapp3/vminfo/issues/new) 讨论 bug、功能或非平凡改动
2. 阅读 [CONTRIBUTING.md](../CONTRIBUTING.md)
3. Fork 仓库并提交聚焦改动
4. 运行 `go test ./...` 和 `go vet ./...`
5. 提交 Pull Request

如果你还不确定从哪里开始，也欢迎先加入 [Telegram](https://t.me/VMPulse) 交流。

## 从源码构建

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

## 开发

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
go vet ./...
go run ./cmd/vminfo summary --json
```

## 文档

- [../README.md](../README.md)
- [api.md](api.md)
- [feature-benchmark.md](feature-benchmark.md)
- [../CONTRIBUTING.md](../CONTRIBUTING.md)
- [CHANGELOG.md](CHANGELOG.md)

## License

[MIT](../LICENSE)
