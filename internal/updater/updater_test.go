package updater

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// writeFreshCache seeds a cache entry checked just now with the given latest
// version, as the background notification check would.
func writeFreshCache(t *testing.T, dir, latest string) {
	t.Helper()
	if err := WriteCacheAt(dir, CacheFile{
		LastCheck:     time.Now(),
		LatestVersion: latest,
	}); err != nil {
		t.Fatalf("WriteCacheAt: %v", err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// TestCheckForUpdateUsesFreshCache pins the notification-check behavior: a
// cache entry within TTL answers without hitting the API.
func TestCheckForUpdateUsesFreshCache(t *testing.T) {
	dir := t.TempDir()
	writeFreshCache(t, dir, "v0.3.0")

	var called bool
	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		CurrentVer: "0.3.0",
		CacheDir:   dir,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			return jsonResponse(`{"tag_name":"v0.4.0"}`), nil
		})},
	})

	res, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if called {
		t.Error("expected cached answer without API call")
	}
	if res.LatestVersion != "0.3.0" || res.UpdateAvailable {
		t.Errorf("expected cached latest 0.3.0 with no update, got %+v", res)
	}
}

// TestCheckForUpdateSkipCacheIgnoresFreshCache pins the install path: SkipCache
// must resolve the real latest release even when the cache is fresh.
func TestCheckForUpdateSkipCacheIgnoresFreshCache(t *testing.T) {
	dir := t.TempDir()
	writeFreshCache(t, dir, "v0.3.0")

	var called bool
	u := New(Config{
		Repo:       "cloudapp3/vminfo",
		CurrentVer: "0.3.0",
		CacheDir:   dir,
		SkipCache:  true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			if !strings.HasSuffix(req.URL.Path, "/releases/latest") {
				t.Errorf("unexpected request path %s", req.URL.Path)
			}
			return jsonResponse(`{"tag_name":"v0.4.0"}`), nil
		})},
	})

	res, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if !called {
		t.Fatal("expected a live API check despite fresh cache")
	}
	if res.LatestVersion != "0.4.0" || !res.UpdateAvailable {
		t.Errorf("expected latest 0.4.0 with update available, got %+v", res)
	}

	// The refreshed answer must also update the cache for later checks.
	cache, err := ReadCacheAt(dir)
	if err != nil {
		t.Fatalf("ReadCacheAt: %v", err)
	}
	if cache.LatestVersion != "v0.4.0" {
		t.Errorf("expected cache refreshed to v0.4.0, got %q", cache.LatestVersion)
	}
}
