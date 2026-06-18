---
title: Quick Start
description: Install vminfo and inspect host runtime metrics from TUI, JSON, or web dashboard.
---

# Quick Start

vminfo gives you fast host runtime visibility from the terminal, JSON output, browser dashboard, or Go APIs.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

With sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | sudo bash
```

## Launch the TUI

```bash
vminfo
```

## Print a JSON snapshot

```bash
vminfo summary --json
```

## Start the web dashboard

```bash
vminfo --web
```

Default address:

```text
http://127.0.0.1:20021
```

## Next steps

- Read the [command reference](/commands/)
- Open the [web dashboard guide](/guide/web-dashboard)
- Use the [HTTP API](/api)
- Embed the [Go library](/library/)
