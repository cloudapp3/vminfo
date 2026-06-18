---
title: Go Library
description: Embed vminfo as a Go library for host metrics collection and terminal UI integration.
---

# Go Library

vminfo exposes public packages for collecting host metrics and embedding the terminal UI.

## Packages

- `github.com/cloudapp3/vminfo`
- `github.com/cloudapp3/vminfo/tui`

## Exported types

- `StaticInfo`
- `RuntimeStats`
- `Snapshot`
- `ProcessInfo`
- `AppMetadata`

## Common entry points

- `CollectStatic`
- `CollectStats`
- `CollectAll`
- `Metadata`
- `tui.Run`

## Related

- [Collect metrics](/library/collect)
- [Embed the TUI](/library/embed-tui)
