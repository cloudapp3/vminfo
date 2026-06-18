---
title: Changelog
description: Release notes and recent product changes for vminfo.
---

# Changelog

## Unreleased

- Enhance `vminfo ps` on Linux with positional and `--filter` process search, `--limit`, tree output, `--watch` with `--count` / `--interval`, `AGE` / `COMMAND` columns, and JSON `command` / `started_at_unix` process fields.
- Add a lightweight web Health Summary card, `GET /api/v1/health`, and a filterable/sortable `GET /api/v1/processes` endpoint for process diagnostics.
- Add `docs/api.md` and `docs/roadmap/feature-benchmark.md` to document the HTTP API, competitor feature benchmarking, and a prioritized product roadmap for `vminfo`.
- Add public `github.com/cloudapp3/vminfo/tui` package for embedding the interactive terminal UI from other Go CLIs.
- Add configurable TUI runner options for stdin, stdout, and language selection while keeping existing internal CLI behavior.
- Improve Linux process collection by reading `/proc` directly, with better CPU and user-name resolution.
- Add a TUI toggle for showing or hiding kernel threads in the process list.

## v0.1.4 - 2026-05-03

- Fix web dashboard auth so `vminfo --web` no longer auto-generates or enables a token unless `--token` is explicitly requested.
- Add regression coverage for explicit vs implicit web token resolution.
