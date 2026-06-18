---
title: ps
description: Linux-only process listing with filters, tree view, watch mode, and JSON output.
---

# `vminfo ps`

Linux-only process listing and filtering.

## Usage

```bash
vminfo ps
vminfo ps nginx
vminfo ps --filter ssh
vminfo ps --tree
vminfo ps --watch
vminfo ps --limit 20
vminfo ps --json
vminfo ps --sort mem
```

## Options

| Flag | Description |
| --- | --- |
| positional filter | Filter by name, user, PID, command, or state |
| `--filter` | Explicit process filter |
| `--tree` | Render a process tree |
| `--watch` | Refresh continuously |
| `--count` | Number of samples when used with `--watch` |
| `--interval` | Refresh interval for watch mode |
| `--limit` | Limit the number of rows |
| `--json` | Output JSON |
| `--sort` | Sort by `cpu`, `mem`, `pid`, or `name` |

## Notes

- The default sort is `cpu`.
- JSON output returns an array of process objects.
- `--watch --json` returns JSON Lines with a `collected_at` timestamp.
- Non-Linux builds keep unsupported stubs for this command.

## Example

```bash
vminfo ps --filter ssh --sort mem --limit 20
```

## Related

- [kill](/commands/kill)
- [HTTP API](/api)
