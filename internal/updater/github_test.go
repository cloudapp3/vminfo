package updater

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadFileUsesConfiguredHTTPClient(t *testing.T) {
	var called bool
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("payload")),
			}, nil
		}),
	}

	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		HTTPClient: client,
	})

	dest := filepath.Join(t.TempDir(), "download.bin")
	if err := u.downloadFile(context.Background(), "https://example.com/download.bin", dest); err != nil {
		t.Fatalf("downloadFile returned error: %v", err)
	}
	if !called {
		t.Fatal("expected configured HTTP client to be used")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("unexpected downloaded contents: %q", string(data))
	}
}
