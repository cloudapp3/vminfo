---
layout: home
hero:
  name: vminfo
  text: Terminal system monitor, web dashboard, and Go library
  tagline: Cross-platform host runtime visibility for Linux, macOS, and Windows — from TUI, JSON output, browser dashboard, or embeddable Go APIs.
  image:
    src: /logo.svg
    alt: vminfo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View Commands
      link: /commands/
    - theme: alt
      text: HTTP API
      link: /api
    - theme: alt
      text: GitHub
      link: https://github.com/cloudapp3/vminfo

features:
  - icon: 🖥️
    title: Polished TUI
    details: Live terminal dashboard for CPU, memory, disk, network, load, and processes.
    link: /guide/tui-controls
  - icon: 📦
    title: Script-friendly JSON
    details: Export snapshots and watch streams for automation, CI, and diagnostics.
    link: /commands/summary
  - icon: 🌐
    title: Lightweight Web Dashboard
    details: Start a read-only browser dashboard with REST and WebSocket endpoints.
    link: /guide/web-dashboard
  - icon: 🧩
    title: Embeddable Go APIs
    details: Import vminfo collection APIs or embed the interactive TUI into your own Go CLI.
    link: /library/
  - icon: ⚡
    title: Zero-config Runtime Visibility
    details: No daemon, no database, no central server. Install and inspect the local host quickly.
    link: /guide/installation
  - icon: 🔒
    title: Token-aware Web Mode
    details: Protect remote dashboard access with token, cookie, CORS, and WebSocket origin checks.
    link: /api
---

## Install in one command

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

Then launch the terminal UI:

```bash
vminfo
```

Or get a JSON snapshot for scripts:

```bash
vminfo summary --json
```

## Why vminfo

vminfo is built for developers, SREs, DevOps engineers, and server operators who need low-friction visibility into a machine.

Use it when you need to:

- inspect CPU, memory, disk, network, load, and processes from a terminal
- export machine-readable JSON for scripts and automation
- open a browser dashboard on a server with `vminfo --web`
- embed host metrics collection into another Go tool
- diagnose a single host without running a full monitoring platform

## Screenshots

<div class="vminfo-screenshot-grid">
  <img src="./assets/tui-overview-refreshed.png" alt="vminfo TUI overview" />
  <img src="./assets/web-dashboard.png" alt="vminfo web dashboard" />
  <img src="./assets/tui-processes.png" alt="vminfo process view" />
  <img src="./assets/tui-help.png" alt="vminfo help overlay" />
</div>

![vminfo demo GIF](./assets/tui-demo.gif)

## Quick links

- [Quick start](/guide/quick-start)
- [Installation](/guide/installation)
- [Deployment](/guide/deployment)
- [Command reference](/commands/)
- [HTTP API](/api)
- [Go Library](/library/)
- [中文文档](/zh/)
