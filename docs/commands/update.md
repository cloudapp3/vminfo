---
title: update
description: Check for or install the latest tagged vminfo release.
---

# `vminfo update`

Checks GitHub Releases and, when possible, installs the newest tagged build.

## Usage

```bash
vminfo update
vminfo update --check
vminfo update --version v0.1.0
```

## Notes

- `--check` only checks for updates.
- `--version` checks or installs a specific release tag.
- A dev build cannot self-update without `--version`.
- On Windows, install mode is unsupported because the binary cannot replace itself in place.

## Example

```bash
vminfo update --check
```

## Related

- [Installation](/guide/installation)
- [Changelog](/changelog)
