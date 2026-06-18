---
title: vminfo 中文文档
description: vminfo 是面向 Linux、macOS 和 Windows 的跨平台主机运行时信息工具，支持终端 UI、JSON 输出、Web 仪表盘与 Go 库。
---

# vminfo 中文文档

vminfo 是一个跨平台主机运行时信息工具，可以通过终端 UI、JSON 输出、浏览器仪表盘或 Go API 快速查看 CPU、内存、磁盘、网络、负载与进程。

## 快速安装

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

## 启动 TUI

```bash
vminfo
```

## 输出 JSON

```bash
vminfo summary --json
```

## 启动 Web 仪表盘

```bash
vminfo --web
```

继续阅读：

- [快速开始](/zh/quick-start)
- [命令参考](/zh/commands)
- [HTTP API](/api)
- [Go Library](/library/)
