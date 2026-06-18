---
title: 快速开始
description: 安装 vminfo，并从终端、JSON 或 Web 仪表盘查看主机指标。
---

# 快速开始

vminfo 可以让你快速查看主机运行时信息，适合终端、脚本和 Web 仪表盘三种场景。

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

加上 sudo：

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | sudo bash
```

## 启动 TUI

```bash
vminfo
```

## 输出 JSON 快照

```bash
vminfo summary --json
```

## 启动 Web 仪表盘

```bash
vminfo --web
```

默认地址：

```text
http://127.0.0.1:20021
```

## 下一步

- 查看 [命令参考](/zh/commands)
- 阅读 [HTTP API](/api)
- 了解 [Go Library](/library/)
- 看英文的 [Web Dashboard](/guide/web-dashboard)
