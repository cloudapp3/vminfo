package app

import (
	"strings"
	"testing"
)

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

func TestWebDashboardListenLinesIncludeToken(t *testing.T) {
	lines := webDashboardListenLines("127.0.0.1:20021", nil, "secret-token")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if got := lines[0]; got != "Web dashboard: http://127.0.0.1:20021/?token=secret-token" {
		t.Fatalf("unexpected listen line: %q", got)
	}
}
