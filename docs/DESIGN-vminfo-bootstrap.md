# DESIGN-vminfo-bootstrap

## 背景

需要从 `vmpulse cli info` 中拆出一个独立开源项目 `vminfo`，目标不是简单 demo，而是：

1. 可嵌入的 Go 库
2. 正式可用的 CLI
3. 默认进入 TUI
4. 保留 Linux-only 进程列表与 `SIGTERM`
5. 暂不纳入 `bench sysinfo`

## 冻结命令面

```bash
vminfo                 # 默认进入 TUI
vminfo info            # TUI 别名
vminfo --web           # 启动 Web dashboard
vminfo --web --tui     # Web dashboard + TUI
vminfo summary         # 单次采样
vminfo watch           # 周期观察
vminfo ps              # 进程列表
vminfo kill <pid>      # Linux 发送 SIGTERM
vminfo version         # 版本信息
vminfo version --json  # 应用元数据 JSON
```

## 包边界

```text
cmd/vminfo            CLI 入口
internal/app          命令路由与参数解析
internal/collector    周期采样与快照聚合
internal/tui          Bubble Tea / Lipgloss 视图层
internal/web          内嵌 Web dashboard
*.go                  对外公开的采集/进程库（`package vminfo`）
```

## 实施批次（全部完成）

### 第 1 批 ✓ — Bootstrap

- 初始化独立 Go module
- 初始化 `cmd/vminfo` 入口
- 初始化 CLI 命令路由
- 冻结命令面

### 第 2 批 ✓ — 采集层

- 建立模块根目录 `github.com/VPSMarket/vminfo` 的类型与采集 API
- 实现静态信息采集（OS、内核、架构、CPU、内存、磁盘、虚拟化）
- 实现动态指标采集（CPU 使用率、负载、内存/交换/磁盘用量、网络 I/O、TCP/UDP、进程数、运行时间）
- 采样式 CPU 与网络速度计算

### 第 3 批 ✓ — 进程管理

- Linux `ps/kill` 实现（基于 gopsutil/process）
- 非 Linux 提供 unsupported stub（编译通过，运行时返回错误）
- 进程信息：PID、PPID、CPU%、MEM%、RSS、用户、状态、名称

### 第 4 批 ✓ — 非 TUI CLI

- `summary` — 单次采样，支持 `--json`、`--interval`
- `watch` — 持续输出，支持 `--json`、`--interval`、`--count`
- `ps` — 进程列表，支持 `--json`、`--sort`（cpu|mem|pid|name）
- `kill` — 发送 SIGTERM
- `version` / `--version` — 版本信息，支持 `--json`
- 应用元数据输出（name, version, commit, buildTime, channel）
- `-ldflags` 版本注入机制

### 第 5 批 ✓ — TUI

- Bubble Tea + Lipgloss TUI 实现
- 概览视图：静态信息 + 运行时指标，宽屏左右分栏
- 进程视图：列表 + 排序（cpu/mem/pid/name）+ 选择 + SIGTERM 确认
- 状态指示（LIVE/PAUSED/LOADING/ERROR/STALE）
- 每 3 秒自动刷新，支持暂停/恢复
- 帮助界面

### 第 6 批 ✓ — Web dashboard

- `--web` 启动内嵌 HTTP dashboard
- `--bind` / `--port` 控制监听地址，默认 `127.0.0.1:9990`
- `--interval` 控制 Web 模式采样刷新频率
- `--tui` 支持 Web dashboard 与终端 TUI 同时运行
- 提供 `/healthz`、`/api/v1/snapshot` 与 `/ws`

### 基础设施 ✓

- GitHub Actions：format / test / vet / smoke / cross-build
- 单元测试（version/help 输出验证）
- 文档、截图、GIF、JSON 示例补齐

## 后续可能方向

- Bench / sysinfo 集成
- 更多平台支持（macOS 进程管理）
- 配置文件支持
- 远程监控上报
- i18n
