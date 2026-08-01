package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverInstructions = "Read-only local host diagnostics. Process command lines are omitted unless include_command is true. lookup_ip sends an explicit request to ip.bestcheapvps.org."

// NewServer builds a tools-only MCP server backed by local vminfo APIs.
func NewServer(backend Backend) *mcp.Server {
	if backend == nil {
		backend = nativeBackend{}
	}
	meta := backend.Metadata()
	server := mcp.NewServer(&mcp.Implementation{
		Name:       "vminfo",
		Title:      "vminfo host diagnostics",
		Version:    meta.Version,
		WebsiteURL: meta.Homepage,
	}, &mcp.ServerOptions{
		Instructions: serverInstructions,
		Capabilities: &mcp.ServerCapabilities{},
	})
	registerTools(server, backend)
	return server
}

// RunStdio serves one MCP client over the supplied input and output streams.
// The streams are deliberately not closed because they are normally os.Stdin
// and os.Stdout owned by the caller.
func RunStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	if stdin == nil {
		return fmt.Errorf("MCP stdin is required")
	}
	if stdout == nil {
		return fmt.Errorf("MCP stdout is required")
	}
	transport := &mcp.IOTransport{
		Reader: io.NopCloser(stdin),
		Writer: nopWriteCloser{Writer: stdout},
	}
	if err := NewServer(nil).Run(ctx, transport); err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			return nil
		}
		return err
	}
	return nil
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
