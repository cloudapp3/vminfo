## Summary

Describe the problem and the focused change that solves it.

## Affected surface

- [ ] CLI commands or flags
- [ ] Terminal UI
- [ ] Web dashboard, REST API, or WebSocket
- [ ] Public Go library
- [ ] Collector or platform behavior
- [ ] Installer, updater, CI, or release packaging
- [ ] Documentation only

## Validation

List the commands you ran and the platforms you tested.

```text
go test ./...
go test -race ./...
go vet ./...
```

## Compatibility and security

- [ ] Existing CLI JSON output remains compatible, or the intended change is documented.
- [ ] Public Go APIs remain compatible, or the intended breaking change is documented.
- [ ] Cross-platform commands retain non-Linux support; `ps` / `kill` retain unsupported stubs where required.
- [ ] Network, file I/O, token, and user-input paths include appropriate timeout and error handling.
- [ ] Logs, screenshots, fixtures, and examples contain no secrets or sensitive host data.

## Documentation and visuals

- [ ] README, documentation, tests, and examples were updated when behavior changed.
- [ ] TUI or web changes include a screenshot or terminal capture when useful.
- [ ] No unrelated files or user changes were reverted.

Related issue:
