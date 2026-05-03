// Package tui exposes the interactive terminal UI used by the vminfo CLI.
package tui

import (
	"context"
	"io"
	"os"
	"strings"

	internali18n "github.com/cloudapp3/vminfo/internal/i18n"
	internaltui "github.com/cloudapp3/vminfo/internal/tui"
)

// Options configures the interactive terminal UI.
type Options struct {
	// Stdout receives the rendered terminal UI. Defaults to os.Stdout.
	Stdout io.Writer
	// Stdin provides keyboard input. Defaults to os.Stdin.
	Stdin io.Reader
	// Lang selects the UI language. Examples: "en", "zh", "zh-CN".
	// Empty means auto-detect from VMINFO_LANG, LC_ALL, or LANG.
	Lang string
}

// Run starts the vminfo interactive terminal UI.
func Run(ctx context.Context, opts Options) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	lang := normalizeLang(opts.Lang)
	if lang == "" {
		lang = internali18n.Detect()
	}

	return internaltui.RunWithOptions(ctx, internaltui.RunOptions{
		Stdout: stdout,
		Stdin:  stdin,
		TR:     internali18n.New(lang),
	})
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	lang = strings.ReplaceAll(lang, "_", "-")
	if idx := strings.Index(lang, "-"); idx > 0 {
		lang = lang[:idx]
	}
	return lang
}
