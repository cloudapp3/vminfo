package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cloudapp3/vminfo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerAdvertisesReadOnlyTools(t *testing.T) {
	backend := newFakeBackend()
	session := connectClient(t, NewServer(backend))
	capabilities := session.InitializeResult().Capabilities
	if capabilities.Tools == nil || capabilities.Logging != nil || capabilities.Prompts != nil || capabilities.Resources != nil {
		t.Fatalf("unexpected server capabilities: %+v", capabilities)
	}

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	gotNames := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		gotNames = append(gotNames, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q is not marked read-only", tool.Name)
		}
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Errorf("tool %q is missing an input or output schema", tool.Name)
		}
	}
	slices.Sort(gotNames)
	wantNames := []string{
		toolCheckPort,
		toolGetSystemSnapshot,
		toolGetVersion,
		toolListProcesses,
		toolLookupIP,
		toolPingHost,
		toolResolveDNS,
	}
	slices.Sort(wantNames)
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("tool names = %v, want %v", gotNames, wantNames)
	}
}

func TestServerReturnsStructuredVersion(t *testing.T) {
	backend := newFakeBackend()
	backend.metadata = vminfo.AppMetadata{Name: "vminfo", Version: "v9.8.7", Channel: "test"}
	session := connectClient(t, NewServer(backend))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: toolGetVersion})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool returned tool error: %+v", result.Content)
	}
	var got vminfo.AppMetadata
	decodeStructured(t, result.StructuredContent, &got)
	if got.Version != "v9.8.7" || got.Channel != "test" {
		t.Fatalf("version result = %+v", got)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1 JSON fallback", len(result.Content))
	}
}

func TestServerReturnsStructuredSnapshotWithEmptyCollections(t *testing.T) {
	backend := newFakeBackend()
	backend.snapshot.System.Hostname = "empty-host"
	session := connectClient(t, NewServer(backend))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: toolGetSystemSnapshot})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool returned tool error: %+v", result.Content)
	}
	var got map[string]any
	decodeStructured(t, result.StructuredContent, &got)
	if got["system"].(map[string]any)["hostname"] != "empty-host" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestServerReturnsToolErrorsToClient(t *testing.T) {
	session := connectClient(t, NewServer(newFakeBackend()))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      toolListProcesses,
		Arguments: map[string]any{"sort_by": "rss"},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("CallTool result IsError = false, want true")
	}
	if len(result.Content) != 1 || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "sort_by") {
		t.Fatalf("unexpected tool error content: %+v", result.Content)
	}
}

func TestRunStdioValidatesStreamsAndStopsOnEOF(t *testing.T) {
	if err := RunStdio(context.Background(), nil, &bytes.Buffer{}); err == nil {
		t.Fatal("RunStdio accepted nil stdin")
	}
	if err := RunStdio(context.Background(), strings.NewReader(""), nil); err == nil {
		t.Fatal("RunStdio accepted nil stdout")
	}
	var stdout bytes.Buffer
	if err := RunStdio(context.Background(), strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("RunStdio on EOF returned error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("RunStdio wrote unexpected output: %q", stdout.String())
	}
}

func connectClient(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect returned error: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "vminfo-test", Version: "v1"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect returned error: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func decodeStructured(t *testing.T, value any, dst any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
}
