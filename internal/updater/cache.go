package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheFileName = "update-check.json"

// CacheFile represents the on-disk update check cache.
type CacheFile struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

// CacheDir returns the default cache directory for vminfo.
// Priority: XDG_CACHE_HOME/vminfo > ~/.cache/vminfo
func CacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "vminfo")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "vminfo")
	}
	return filepath.Join(home, ".cache", "vminfo")
}

func cacheFilePath(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = CacheDir()
	}
	return filepath.Join(dir, cacheFileName)
}

// ReadCache reads the cache file from disk. Returns zero-value CacheFile if not found.
func ReadCache() (CacheFile, error) {
	return ReadCacheAt("")
}

// ReadCacheAt reads the cache file from a specific directory. Empty dir falls
// back to CacheDir().
func ReadCacheAt(dir string) (CacheFile, error) {
	path := cacheFilePath(dir)
	data, err := os.ReadFile(path)
	if err != nil {
		return CacheFile{}, nil
	}
	var cf CacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return CacheFile{}, nil
	}
	return cf, nil
}

// WriteCache writes the cache file to disk, creating the directory if needed.
func WriteCache(cf CacheFile) error {
	return WriteCacheAt("", cf)
}

// WriteCacheAt writes the cache file to a specific directory. Empty dir falls
// back to CacheDir().
func WriteCacheAt(dir string, cf CacheFile) error {
	path := cacheFilePath(dir)
	dir = filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ShouldCheck returns true if the cache is expired (past TTL) or missing.
func ShouldCheck(cf CacheFile, ttl time.Duration) bool {
	if cf.LastCheck.IsZero() {
		return true
	}
	return time.Since(cf.LastCheck) > ttl
}
