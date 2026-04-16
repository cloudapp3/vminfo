package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

// DefaultCacheTTL is the default time between update checks.
const DefaultCacheTTL = 24 * time.Hour

// Config holds configuration for the updater.
type Config struct {
	Repo        string
	CurrentVer  string
	GitHubToken string
	CacheDir    string
	CacheTTL    time.Duration
	HTTPClient  *http.Client
}

// CheckResult is returned by check operations.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Release         *Release
}

// Updater handles checking, downloading, and applying updates.
type Updater struct {
	cfg Config
}

// New creates a new Updater with the given config.
func New(cfg Config) *Updater {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = DefaultCacheTTL
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = CacheDir()
	}
	return &Updater{cfg: cfg}
}

// CheckForUpdate queries the GitHub Releases API and compares versions.
// It uses the cache to avoid redundant API calls within CacheTTL.
func (u *Updater) CheckForUpdate(ctx context.Context) (*CheckResult, error) {
	current := stripVersionPrefix(u.cfg.CurrentVer)
	unknownCurrent := current == "" || current == "dev"

	// Check cache first
	cache, _ := ReadCacheAt(u.cfg.CacheDir)
	if !ShouldCheck(cache, u.cfg.CacheTTL) && cache.LatestVersion != "" {
		latest := stripVersionPrefix(cache.LatestVersion)
		return &CheckResult{
			CurrentVersion:  current,
			LatestVersion:   latest,
			UpdateAvailable: unknownCurrent || compareVersions(current, latest) < 0,
		}, nil
	}

	// Fetch from GitHub API
	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		// Return cached result on error if available
		if cache.LatestVersion != "" {
			latest := stripVersionPrefix(cache.LatestVersion)
			return &CheckResult{
				CurrentVersion:  current,
				LatestVersion:   latest,
				UpdateAvailable: unknownCurrent || compareVersions(current, latest) < 0,
			}, nil
		}
		return nil, err
	}

	latest := stripVersionPrefix(release.TagName)

	// Update cache
	_ = WriteCacheAt(u.cfg.CacheDir, CacheFile{
		LastCheck:     time.Now(),
		LatestVersion: release.TagName,
	})

	return &CheckResult{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: unknownCurrent || compareVersions(current, latest) < 0,
		Release:         release,
	}, nil
}

// CheckSpecificVersion checks for a specific version tag.
func (u *Updater) CheckSpecificVersion(ctx context.Context, tag string) (*CheckResult, error) {
	current := stripVersionPrefix(u.cfg.CurrentVer)
	target := stripVersionPrefix(tag)

	release, err := u.fetchReleaseByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	return &CheckResult{
		CurrentVersion:  current,
		LatestVersion:   stripVersionPrefix(release.TagName),
		UpdateAvailable: compareVersions(current, target) < 0,
		Release:         release,
	}, nil
}

// DownloadAndInstall downloads the archive for the current OS/arch,
// verifies the SHA-256 checksum, and atomically replaces the binary.
func (u *Updater) DownloadAndInstall(ctx context.Context, release *Release, progress io.Writer) error {
	archiveName := u.archiveName(release.TagName)
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		u.cfg.Repo, release.TagName, archiveName,
	)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vminfo-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download archive
	archivePath := tmpDir + "/" + archiveName
	fmt.Fprintf(progress, "downloading %s...\n", archiveName)
	if err := u.downloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Verify checksum
	fmt.Fprintf(progress, "verifying checksum...\n")
	checksums, err := u.fetchChecksums(ctx, release)
	if err != nil {
		return fmt.Errorf("cannot fetch checksums: %w", err)
	}
	expectedHash, ok := checksums[archiveName]
	if !ok {
		return fmt.Errorf("checksum not found for %s", archiveName)
	}
	ok, err = VerifySHA256(archivePath, expectedHash)
	if err != nil {
		return fmt.Errorf("checksum verification error: %w", err)
	}
	if !ok {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}

	// Extract binary
	fmt.Fprintf(progress, "installing...\n")
	extractedPath, err := ExtractBinaryFromTarGz(archivePath, "vminfo", tmpDir)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	if extractedPath == "" {
		return fmt.Errorf("binary not found in archive")
	}

	// Get current binary path
	selfPath, err := SelfPath()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}

	// Atomic replace
	if err := AtomicReplace(extractedPath, selfPath); err != nil {
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	return nil
}

func (u *Updater) archiveName(tag string) string {
	ver := stripVersionPrefix(tag)
	return fmt.Sprintf("vminfo-%s-%s-%s.tar.gz", ver, runtime.GOOS, runtime.GOARCH)
}
