package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const githubAPIBase = "https://api.github.com"

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a single release asset.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// fetchLatestRelease calls GET /repos/{owner}/{repo}/releases/latest
func (u *Updater) fetchLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, u.cfg.Repo)
	return u.fetchRelease(ctx, url)
}

// fetchReleaseByTag calls GET /repos/{owner}/{repo}/releases/tags/{tag}
func (u *Updater) fetchReleaseByTag(ctx context.Context, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", githubAPIBase, u.cfg.Repo, tag)
	return u.fetchRelease(ctx, url)
}

func (u *Updater) fetchRelease(ctx context.Context, url string) (*Release, error) {
	req, err := u.newGitHubRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("github api rate limited (set GITHUB_TOKEN to increase limit)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}
	return &release, nil
}

// downloadFile downloads a file from a URL to destPath.
func (u *Updater) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if u.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+u.cfg.GitHubToken)
	}
	resp, err := u.downloadClient().Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("download write failed: %w", err)
	}
	return nil
}

// fetchChecksums downloads and parses checksums.txt from the release assets,
// returning a map[filename]hash.
func (u *Updater) fetchChecksums(ctx context.Context, release *Release) (map[string]string, error) {
	var checksumsURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumsURL = asset.URL
			break
		}
	}
	if checksumsURL == "" {
		return nil, fmt.Errorf("checksums.txt not found in release assets")
	}

	// Download checksums via browser_download_url pattern
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/checksums.txt",
		u.cfg.Repo, release.TagName,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("failed to download checksums: status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}

	checksums := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		checksums[strings.TrimSpace(parts[1])] = strings.TrimSpace(parts[0])
	}
	return checksums, nil
}

// newGitHubRequest creates an http.Request with GitHub API headers.
func (u *Updater) newGitHubRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if u.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+u.cfg.GitHubToken)
	}
	return req, nil
}

func (u *Updater) client() *http.Client {
	if u.cfg.HTTPClient != nil {
		return u.cfg.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (u *Updater) downloadClient() *http.Client {
	if u.cfg.HTTPClient == nil {
		return &http.Client{
			Timeout:       5 * time.Minute,
			CheckRedirect: downloadRedirectPolicy(nil),
		}
	}

	cloned := *u.cfg.HTTPClient
	cloned.CheckRedirect = downloadRedirectPolicy(u.cfg.HTTPClient.CheckRedirect)
	return &cloned
}

func downloadRedirectPolicy(next func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		// Clear Authorization header when redirecting to different host.
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			req.Header.Del("Authorization")
		}
		if next != nil {
			return next(req, via)
		}
		return nil
	}
}
