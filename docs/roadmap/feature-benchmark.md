---
title: Feature Benchmark and Roadmap
description: Feature comparison and product direction notes for vminfo.
---

# Feature Benchmark and Roadmap

This document records vminfo's feature tradeoffs after comparing similar open-source projects. It is a planning reference for issues and PR review, not a promise list.

## Product position

vminfo should stay within these boundaries:

- **Single binary, low dependency, low configuration** — install and use immediately without a daemon, database, or central server.
- **Local, real-time diagnostics first** — TUI, CLI, JSON, Web/API all focus on seeing the current machine quickly.
- **Scriptable and embeddable** — keep the JSON schema and Go public API stable for CI, automation, and other Go tools.
- **Lightweight web, not a monitoring platform** — borrow the instant readability of Netdata, but not the full cloud, multi-node, long-retention stack.
- **Clear platform semantics** — `summary` / `watch` stay cross-platform; `ps` / `kill` remain Linux-only.

## Reference projects

| Project | Representative strengths | What vminfo can borrow |
| --- | --- | --- |
| [btop](https://github.com/aristocratos/btop) | High-quality TUI, resource graphs, process filtering, tree view, signal actions, themes | Improve TUI visuals and process interaction |
| [bottom / btm](https://github.com/ClementTsang/bottom) | Cross-platform terminal monitoring, widget focus, layout config, history windows | Add focus/basic modes and better narrow/wide layouts |
| [Glances](https://github.com/nicolargo/glances) | TUI, web, REST API, JSON/CSV/stdout, client/server, MCP | Strengthen script output and API docs |
| [procs](https://github.com/dalance/procs) | Modern `ps` replacement, search, tree, watch, IO, Docker info | Make `vminfo ps` a practical troubleshooting entrypoint |
| [htop](https://github.com/htop-dev/htop) | Mature process operations, sorting, filtering, kill/renice, command display | Align TUI process page with familiar shortcuts |
| [Netdata](https://github.com/netdata/netdata) | Zero-config realtime web, automatic discovery, clear summaries | Borrow the “one glance” web summary without platform complexity |
| [gopsutil](https://github.com/shirou/gopsutil) / [psutil](https://github.com/giampaolo/psutil) | Stable cross-platform collection APIs | Keep the high-level Go API stable and well documented |

## Recommended roadmap

### P0: Highest value

1. **Enhance `vminfo ps`**
   - `vminfo ps <keyword>`: filter by process name, user, PID, or command line.
   - `vminfo ps --filter <keyword>`: explicit filter for scripts.
   - `vminfo ps --tree`: tree view.
   - `vminfo ps --watch`: continuous refresh with `--count` and `--interval`.
   - `vminfo ps --limit 20`: limit rows after sort/filter.
   - `vminfo ps --sort cpu|mem|pid|name`: keep the current sort set.

2. **Add process detail views**
   - Show PID, PPID, user, command line, CPU, RSS, and start time in the TUI.
   - Keep the Web process table and API filterable and sortable.

3. **Add a web problem summary**
   - Show top CPU, memory, disk, and network warning signals.
   - Keep it read-only, stateless, and lightweight.

### P1: Automation and integration

4. **Field selection and tabular output**
   - Add user-selectable field subsets for summary and watch.
   - Keep field names aligned with JSON.

5. **Short history windows**
   - Keep recent CPU, memory, network, and disk I/O samples in memory.
   - Avoid databases and default disk writes.

6. **Go library examples**
   - Add more import examples for single snapshots, continuous sampling, custom JSON, and embedded TUI.

### P2: Useful, but not core

7. **MCP server**
   - Expose snapshot and process tools for AI assistants.

8. **Container identification**
   - Show Docker/containerd identifiers in `ps`, TUI, and Web views.

9. **Lightweight health scoring**
   - Emit a transparent `score` and `warnings` set.

## Not recommended

- Long-term data storage or databases
- Cloud accounts or multi-node control planes
- Heavy plugin systems
- Default Prometheus/Influx exporters
- A large frontend app for the dashboard

## Delivery rules

- If CLI behavior, JSON output, Web API, or exported Go APIs change, update README, docs, and tests together.
- Keep `ps` / `kill` Linux-only.
- Keep web remote access guarded by token, cookie, CORS, and WebSocket origin rules.
