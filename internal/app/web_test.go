package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestRunWebReturnsServerStartError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWeb(ctx, &stdout, &stderr, i18n.New("en"), "127.0.0.1:99999", 10*time.Millisecond, false, true, "", false)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected server start error, got nil")
		}
		if !strings.Contains(err.Error(), "web server error") {
			t.Fatalf("expected web server error, got %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("runWeb did not return after server start failure")
	}
}

func TestResolveWebTokenBareFlag(t *testing.T) {
	token, generated, err := resolveWebToken("", true)
	if err != nil {
		t.Fatalf("resolveWebToken returned error: %v", err)
	}
	if !generated {
		t.Fatal("expected bare --token to be marked as generated")
	}
	if token == "" {
		t.Fatal("expected auto-generated token to be non-empty")
	}
	if strings.Contains(token, " ") {
		t.Fatalf("expected URL-safe token, got %q", token)
	}
}

func TestResolveWebTokenFixed(t *testing.T) {
	token, generated, err := resolveWebToken("my-fixed-token", false)
	if err != nil {
		t.Fatalf("resolveWebToken returned error: %v", err)
	}
	if generated {
		t.Fatal("expected fixed token not to be marked as generated")
	}
	if token != "my-fixed-token" {
		t.Fatalf("expected fixed token to round-trip, got %q", token)
	}
}

func TestResolveWebTokenEmpty(t *testing.T) {
	token, generated, err := resolveWebToken("", false)
	if err != nil {
		t.Fatalf("resolveWebToken returned error: %v", err)
	}
	if generated {
		t.Fatal("expected no generation when autoGenerate is false")
	}
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestResolveRequestedWebTokenNotRequested(t *testing.T) {
	token, generated, err := resolveRequestedWebToken("", false)
	if err != nil {
		t.Fatalf("resolveRequestedWebToken returned error: %v", err)
	}
	if generated {
		t.Fatal("expected no generation when --token is not requested")
	}
	if token != "" {
		t.Fatalf("expected empty token when --token is not requested, got %q", token)
	}
}

func TestResolveRequestedWebTokenBareFlag(t *testing.T) {
	token, generated, err := resolveRequestedWebToken("", true)
	if err != nil {
		t.Fatalf("resolveRequestedWebToken returned error: %v", err)
	}
	if !generated {
		t.Fatal("expected bare --token to generate a token")
	}
	if token == "" {
		t.Fatal("expected generated token to be non-empty")
	}
}

func TestWebDashboardListenLinesIncludeToken(t *testing.T) {
	lines := webDashboardListenLines("127.0.0.1:20021", nil, "secret-token")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if got := lines[0]; got != "Web dashboard: http://127.0.0.1:20021/?token=secret-token" {
		t.Fatalf("unexpected listen line: %q", got)
	}
}
