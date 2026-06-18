---
title: summary
description: Collect one runtime snapshot as text or JSON.
---

# `vminfo summary`

Collects one snapshot of the current host state.

## Usage

```bash
vminfo summary
vminfo summary --json
vminfo summary --interval 1s
```

## Output

- Text mode prints a human-readable host summary.
- JSON mode prints a `vminfo.Snapshot` object with `static` and `stats` fields.

## When to use it

- quick terminal checks
- shell scripts
- CI diagnostics
- automation that needs a single sample

## Example

```bash
vminfo summary --json
```

## Related

- [watch](/commands/watch)
- [HTTP API](/api)
- [Go library](/library/)
