---
title: Command Reference
description: Overview of the vminfo CLI commands and common workflows.
---

# Command Reference

## Common commands

| Command | What it does |
| --- | --- |
| `vminfo` | Start the interactive TUI |
| `vminfo info` | Alias for the TUI |
| `vminfo summary` | Collect one snapshot |
| `vminfo watch` | Stream snapshots continuously |
| `vminfo --web` | Start the browser dashboard |
| `vminfo ps` | List local processes on Linux |
| `vminfo kill <pid>` | Send SIGTERM to a Linux process |
| `vminfo update` | Check for or install a release update |
| `vminfo --lang zh` | Switch the UI language |

## Cheat sheet

```bash
vminfo
vminfo info
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

Use the pages below for full details:

- [summary](/commands/summary)
- [watch](/commands/watch)
- [ps](/commands/ps)
- [kill](/commands/kill)
- [update](/commands/update)
