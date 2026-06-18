---
title: Installation
description: Install vminfo with the shell installer, go install, or a custom binary directory.
---

# Installation

## Shell installer (Linux/macOS)

The installer script is the fastest path for Linux and macOS.

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash
```

With sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | sudo bash
```

The script auto-selects an install directory in this order:

1. `/usr/local/bin`
2. `~/.local/bin`
3. `~/bin`

If you want a fixed directory, pass `--dir`:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudapp3/vminfo/main/install.sh | bash -s -- --dir /opt/bin
```

The script also supports `--skip-verify` if you explicitly want to skip checksum verification.

## Go install

```bash
go install github.com/cloudapp3/vminfo/cmd/vminfo@latest
```

## Update

```bash
vminfo update
vminfo update --check
vminfo update --version v0.1.0
```

Notes:

- `vminfo update` installs the newest tagged release when you are running a tagged build
- `vminfo update --check` only checks for updates
- `vminfo update --version v0.1.0` checks or installs a specific release tag

## PATH troubleshooting

If `vminfo` is installed but not found, make sure your install directory is on `PATH`.

Common examples:

```bash
export PATH="$HOME/.local/bin:$PATH"
export PATH="$HOME/bin:$PATH"
```

## Uninstall

Remove the binary from the directory you installed into:

```bash
rm -f /usr/local/bin/vminfo
rm -f ~/.local/bin/vminfo
rm -f ~/bin/vminfo
```
