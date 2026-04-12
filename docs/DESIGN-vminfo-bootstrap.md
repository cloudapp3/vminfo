# DESIGN-vminfo-bootstrap

## 背景

需要从 `vmpulse cli info` 中拆出一个独立开源项目 `vminfo`，目标不是简单 demo，而是：

1. 可嵌入的 Go 库
2. 正式可用的 CLI
3. 默认进入 TUI
4. 保留 Linux-only 进程列表与 `SIGTERM`
5. 暂不纳入 `bench sysinfo`

## 第 1 批目标

本批只做 bootstrap，不迁移采集逻辑：

- 初始化独立 module
- 初始化 `cmd/vminfo` 入口
- 初始化 CLI 命令路由
- 冻结命令面，避免后续反复改名
- 为后续批次留出目录边界

## 冻结命令面

```bash
vminfo                 # 默认进入 TUI
vminfo info            # TUI 别名
vminfo summary         # 单次采样
vminfo watch           # 周期观察
vminfo ps              # 进程列表
vminfo kill <pid>      # Linux 发送 SIGTERM
```

## 包边界

```text
cmd/vminfo            CLI 入口
internal/app          命令路由与参数解析
pkg/vminfo            对外公开的采集/进程库（后续批次）
internal/tui          Bubble Tea / Lipgloss 视图层（后续批次）
```

## 本批非目标

- 不迁移 `vmpulse` 协议与上报逻辑
- 不迁移 `AppVersion` 业务字段
- 不迁移 i18n
- 不实现真实采集与 TUI，仅保留 placeholder

## 后续批次

### 第 2 批
- 建立 `pkg/vminfo` 类型与采集 API
- 迁移静态信息与动态指标采集

### 第 3 批
- 加入 Linux `ps/kill`
- 非 Linux 提供 unsupported stub

### 第 4 批
- 建立 `summary / watch / ps / kill` 非 TUI CLI
- 当前状态：已完成，其中 `watch` 支持 `--interval`、`--json`、`--count`

### 第 5 批
- 迁移 `info` TUI
- 去除对 `vmpulse` 私有 i18n / 业务模型的耦合
