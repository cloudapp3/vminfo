---
title: Overview
description: What vminfo does and which host metrics it collects.
---

# Overview

vminfo is a cross-platform host runtime information toolkit for quick local diagnostics.

## What it gives you

- **TUI** — a full-screen terminal dashboard
- **JSON output** — script-friendly snapshots and watch streams
- **Web dashboard** — a lightweight browser UI with REST and WebSocket endpoints
- **Go library** — importable metrics collection and embedded TUI APIs

## Collected metrics

vminfo collects:

- CPU usage, per-core usage, CPU frequency
- memory and swap usage
- disk usage and disk I/O
- network totals, speeds, TCP/UDP counts
- per-interface traffic, errors, and drops
- process lists and process metadata
- temperature readings
- uptime and host metadata

## Public Go types

- `StaticInfo`
- `RuntimeStats`
- `Snapshot`
- `ProcessInfo`
- `AppMetadata`

## Common entry points

- `vminfo`
- `vminfo info`
- `vminfo summary`
- `vminfo watch`
- `vminfo --web`
- `vminfo ps`
- `vminfo kill <pid>`
- `vminfo update`

## Good fits

vminfo is a good fit when you want:

- a single binary for fast host inspection
- a readable terminal UI for interactive work
- JSON for scripts, CI, or automation
- a browser dashboard without running a monitoring stack
- a Go library you can embed in another CLI
