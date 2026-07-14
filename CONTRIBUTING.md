# Contributing to vminfo

Thanks for helping improve vminfo. Bug reports, feature ideas, documentation improvements, tests, platform compatibility fixes, and focused pull requests are all welcome.

中文或英文 issue / PR 都欢迎。If you want to ask a quick question first, join the [VMPulse Telegram group](https://t.me/VMPulse) or [open an issue](https://github.com/cloudapp3/vminfo/issues/new).

## Ways to contribute

- Report bugs or regressions
- Request features or CLI / TUI / web UX improvements
- Improve documentation, examples, or onboarding
- Add tests or fix platform-specific issues
- Improve cross-platform behavior while keeping compatibility in mind

## Before you start

- Prefer opening an issue first for bugs, feature requests, or non-trivial changes
- For small documentation fixes, feel free to open a PR directly
- Keep changes small and focused
- Avoid adding new dependencies unless there is a strong reason and prior discussion
- Do not revert unrelated user changes in the working tree

## Project layout

- `github.com/cloudapp3/vminfo` — public collection / process library
- `internal/app` — CLI routing, flags, and focused command tests
- `internal/tui` — Bubble Tea / Lipgloss TUI
- `cmd/vminfo` — program entrypoint

## Compatibility expectations

- `summary` / `watch` should remain cross-platform
- `ps` / `kill` are Linux-only; non-Linux implementations should keep unsupported stubs
- Keep CLI JSON output compatible unless a breaking change is explicitly intended
- Keep `github.com/cloudapp3/vminfo` reusable as a public Go library
- If command behavior or exported APIs change, update tests, `README.md`, and the documentation site in `cloudapp3/vmdocs`

## Local development

```bash
git clone https://github.com/cloudapp3/vminfo.git
cd vminfo
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/vminfo version
go run ./cmd/vminfo summary --json
go run ./cmd/vminfo watch --count 1
go run ./cmd/vminfo ps # Linux-only
```

Notes:

- TUI needs a real TTY; for non-interactive verification, prefer `summary`, `watch`, or `version`
- Run `gofmt -w` on any modified Go files before submitting

## Pull request checklist

- [ ] The change is focused and clearly described
- [ ] Modified Go files have been formatted with `gofmt -w`
- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` passes
- [ ] README / docs / tests were updated when behavior changed
- [ ] No unrelated files were reverted

## PR tips

- Link the related issue when possible
- Include screenshots or terminal captures for TUI / web changes
- Mention which platform(s) you tested on

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
