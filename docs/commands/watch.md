---
title: watch
description: Stream runtime snapshots continuously, either as text or JSON Lines.
---

# `vminfo watch`

Streams snapshots continuously.

## Usage

```bash
vminfo watch
vminfo watch --json
vminfo watch --count 1
vminfo watch --interval 2s
```

## Output

- Text mode prints timestamped snapshots.
- JSON mode prints JSON Lines with `collected_at`, `static`, and `stats`.

## Good for

- terminal monitoring
- log pipelines
- CI checks
- simple sampling loops

## Example

```bash
vminfo watch --json --count 3
```

## Related

- [summary](/commands/summary)
- [ps](/commands/ps)
