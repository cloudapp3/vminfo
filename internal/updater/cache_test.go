package updater

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCheckForUpdateReadsCustomCacheDir(t *testing.T) {
	cacheDir := t.TempDir()
	if err := WriteCacheAt(cacheDir, CacheFile{
		LastCheck:     time.Now(),
		LatestVersion: "v1.2.3",
	}); err != nil {
		t.Fatalf("WriteCacheAt returned error: %v", err)
	}

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected network request: %s", req.URL.String())
			return nil, nil
		}),
	}

	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		CurrentVer: "v1.0.0",
		CacheDir:   cacheDir,
		CacheTTL:   time.Hour,
		HTTPClient: client,
	})

	result, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate returned error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected update to be available, got %+v", result)
	}
	if result.LatestVersion != "1.2.3" {
		t.Fatalf("expected latest version 1.2.3, got %q", result.LatestVersion)
	}
}

func TestCheckForUpdateWritesCustomCacheDir(t *testing.T) {
	cacheDir := t.TempDir()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/releases/latest") {
				t.Fatalf("unexpected request path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.4.0","assets":[]}`)),
			}, nil
		}),
	}

	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		CurrentVer: "v1.0.0",
		CacheDir:   cacheDir,
		HTTPClient: client,
	})

	result, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate returned error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected update to be available, got %+v", result)
	}

	path := filepath.Join(cacheDir, cacheFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected cache file at %s: %v", path, err)
	}
	if !strings.Contains(string(data), `"latest_version":"v1.4.0"`) {
		t.Fatalf("unexpected cache contents: %s", string(data))
	}
}

func TestCheckForUpdateDevBuildFetchesLatestRelease(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v2.0.0","assets":[]}`)),
			}, nil
		}),
	}

	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		CurrentVer: "dev",
		CacheDir:   t.TempDir(),
		HTTPClient: client,
	})

	result, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate returned error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected dev build check to report an available release, got %+v", result)
	}
	if result.LatestVersion != "2.0.0" {
		t.Fatalf("expected latest version 2.0.0, got %q", result.LatestVersion)
	}
}
