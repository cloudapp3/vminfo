---
title: kill
description: Send SIGTERM to a Linux process by PID.
---

# `vminfo kill`

Sends `SIGTERM` to a Linux process.

::: warning
This command terminates a process. Make sure the PID is correct before running it.
:::

## Usage

```bash
vminfo kill 1234
```

## Notes

- Linux-only
- returns unsupported stubs on non-Linux builds
- may require permission if the target process belongs to another user

## Related

- [ps](/commands/ps)
