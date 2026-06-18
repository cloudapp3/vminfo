---
title: Platform Support
description: Which features work on Linux, macOS, and Windows.
---

# Platform Support

| Capability | Linux | macOS | Windows |
| --- | --- | --- | --- |
| `summary` / `watch` | ✅ | ✅ | ✅ |
| TUI | ✅ | ✅ | ✅ |
| Web dashboard | ✅ | ✅ | ✅ |
| `ps` / `kill` | ✅ | ⚠️ stub | ⚠️ stub |
| `update --check` | ✅ | ✅ | ✅ |
| `update` install | ✅ | ✅ | ⚠️ check-only |

Notes:

- TUI requires a real TTY.
- `ps` and `kill` are Linux-only by design.
- Non-Linux builds keep unsupported stubs for process features.
- Windows can check for updates, but self-replacement is unsupported.
