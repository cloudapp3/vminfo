package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRunMCPRejectsArguments(t *testing.T) {
	err := runMCP(context.Background(), strings.NewReader(""), &bytes.Buffer{}, []string{"unexpected"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("runMCP error = %v, want ErrUsage", err)
	}
}

func TestRunMCPHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := runMCP(context.Background(), strings.NewReader(""), &stdout, []string{"--help"}, i18n.New("en")); err != nil {
		t.Fatalf("runMCP returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "vminfo mcp") || !strings.Contains(stdout.String(), "stdio") {
		t.Fatalf("unexpected MCP help: %q", stdout.String())
	}
}

func TestRunWithIOMCPStopsOnEOF(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := RunWithIO(context.Background(), []string{"mcp"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("RunWithIO returned error: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("MCP EOF output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunWithIOMCPProtocolRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	defer func() {
		_ = clientReader.Close()
		_ = serverWriter.Close()
		_ = serverReader.Close()
		_ = clientWriter.Close()
	}()

	var stderr bytes.Buffer
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- RunWithIO(ctx, []string{"mcp"}, serverReader, serverWriter, &stderr)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "vminfo-app-test", Version: "v1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{
		Reader: clientReader,
		Writer: clientWriter,
	}, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		t.Fatalf("list MCP tools: %v", err)
	}
	got := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range []string{
		"get_system_snapshot",
		"list_processes",
		"resolve_dns",
		"check_port",
		"ping_host",
		"lookup_ip",
		"get_version",
	} {
		if !got[name] {
			_ = session.Close()
			t.Fatalf("MCP tool %q is not advertised: %v", name, got)
		}
	}

	if err := session.Close(); err != nil {
		t.Fatalf("close MCP client: %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("RunWithIO returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("MCP server did not stop after client close: %v", ctx.Err())
	}
	if stderr.Len() != 0 {
		t.Fatalf("MCP wrote diagnostics to stderr: %q", stderr.String())
	}
}

func TestHelpTextIncludesMCP(t *testing.T) {
	if got := helpText(i18n.New("en")); !strings.Contains(got, "vminfo mcp") {
		t.Fatalf("help text does not include MCP command: %q", got)
	}
}
