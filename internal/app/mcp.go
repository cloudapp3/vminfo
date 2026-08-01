package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/cloudapp3/vminfo/internal/mcpserver"
)

func runMCP(ctx context.Context, stdin io.Reader, stdout io.Writer, args []string, tr *i18n.Translator) error {
	if len(args) == 1 && isHelpAlias(args[0]) {
		_, err := io.WriteString(stdout, mcpHelpText(tr))
		return err
	}
	if len(args) != 0 {
		return fmt.Errorf("%w: mcp does not accept arguments: %s", ErrUsage, strings.Join(args, " "))
	}
	if err := mcpserver.RunStdio(ctx, stdin, stdout); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}
	return nil
}

func mcpHelpText(tr *i18n.Translator) string {
	return strings.Join([]string{
		tr.T("Usage:"),
		"  vminfo mcp  " + tr.T("start read-only MCP server over stdio"),
		"",
		tr.T("MCP mode reserves stdout for protocol messages and stops when the client disconnects."),
	}, "\n") + "\n"
}
