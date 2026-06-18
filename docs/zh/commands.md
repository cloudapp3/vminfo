---
title: 命令参考
description: vminfo 中文命令说明、常用参数和示例。
---

# 命令参考

## 常用命令

| 命令 | 作用 |
| --- | --- |
| `vminfo` | 打开交互式 TUI |
| `vminfo info` | TUI 别名 |
| `vminfo summary` | 输出单次快照 |
| `vminfo watch` | 持续输出快照 |
| `vminfo --web` | 打开 Web 仪表盘 |
| `vminfo ps` | 列出 Linux 进程 |
| `vminfo kill <pid>` | 向 Linux 进程发送 SIGTERM |
| `vminfo update` | 检查或安装更新 |
| `vminfo --lang zh` | 切换中文界面 |

## 速查

```bash
vminfo
vminfo summary
vminfo summary --json
vminfo watch
vminfo watch --json
vminfo watch --count 1
vminfo --web
vminfo --web --token
vminfo --web --token secret-token
vminfo --web --tui
vminfo --web --bind 0.0.0.0 --port 8080
vminfo ps
vminfo ps nginx
vminfo ps --filter ssh
vminfo ps --tree
vminfo ps --watch
vminfo ps --limit 20
vminfo ps --json
vminfo ps --sort mem
vminfo kill <pid>
vminfo update
vminfo update --check
vminfo update --version v0.1.0
vminfo --lang zh
```

## 说明

- `summary`：单次快照，适合脚本和 CI。
- `watch`：连续采样，`--json` 时输出 JSON Lines。
- `ps`：Linux-only，支持过滤、树视图、watch、limit 和 JSON。
- `kill`：Linux-only，发送 `SIGTERM`。
- `update`：检查或安装最新 release，Windows 仅支持检查。
