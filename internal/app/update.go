package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/cloudapp3/vminfo/internal/updater"
)

type updateClient interface {
	CheckForUpdate(ctx context.Context) (*updater.CheckResult, error)
	CheckSpecificVersion(ctx context.Context, tag string) (*updater.CheckResult, error)
	DownloadAndInstall(ctx context.Context, release *updater.Release, progress io.Writer) error
}

var newUpdateClient = func(cfg updater.Config) updateClient {
	return updater.New(cfg)
}

func runUpdate(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n")
		fmt.Fprintf(fs.Output(), "  vminfo update [--check] [--version <tag>]\n\n")
		fmt.Fprintf(fs.Output(), "Options:\n")
		fs.PrintDefaults()
	}

	var checkOnly bool
	var targetVersion string
	fs.BoolVar(&checkOnly, "check", false, tr.T("check for updates without installing"))
	fs.StringVar(&targetVersion, "version", "", tr.T("install or inspect a specific release tag"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("%w: update does not accept positional args", ErrUsage)
	}

	meta := vminfo.Metadata()
	if !checkOnly && targetVersion == "" && strings.EqualFold(strings.TrimSpace(meta.Version), "dev") {
		return fmt.Errorf("self-update requires a tagged release build; current version is %q (use --version vX.Y.Z to install a specific release)", meta.Version)
	}

	client := newUpdateClient(updater.Config{
		Repo:        defaultUpdateRepo(meta),
		CurrentVer:  meta.Version,
		GitHubToken: updateTokenFromEnv(),
		CacheDir:    updater.CacheDir(),
	})

	targetTag := normalizeReleaseTag(targetVersion)
	var (
		result *updater.CheckResult
		err    error
	)
	if targetTag != "" {
		result, err = client.CheckSpecificVersion(ctx, targetTag)
	} else {
		result, err = client.CheckForUpdate(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if result == nil {
		return fmt.Errorf("failed to check for updates: empty result")
	}

	if checkOnly {
		return writeUpdateCheck(stdout, result, targetTag != "", tr)
	}
	if targetTag == "" && result.UpdateAvailable && result.Release == nil {
		latestTag := normalizeReleaseTag(result.LatestVersion)
		if latestTag == "" || strings.EqualFold(latestTag, "dev") {
			return fmt.Errorf("failed to install update: release metadata is unavailable for version %q", result.LatestVersion)
		}

		result, err = client.CheckSpecificVersion(ctx, latestTag)
		if err != nil {
			return fmt.Errorf("failed to fetch release metadata for %s: %w", latestTag, err)
		}
		if result == nil {
			return fmt.Errorf("failed to fetch release metadata for %s: empty result", latestTag)
		}
		if normalizeReleaseTag(result.LatestVersion) != latestTag {
			return fmt.Errorf("failed to fetch release metadata for %s: returned version is %s", latestTag, formatReleaseTag(result.LatestVersion))
		}
		if result.Release == nil {
			return fmt.Errorf("failed to install update: release metadata is unavailable")
		}
		if normalizeReleaseTag(result.Release.TagName) != latestTag {
			return fmt.Errorf("failed to fetch release metadata for %s: release tag is %s", latestTag, formatReleaseTag(result.Release.TagName))
		}
	}

	if !result.UpdateAvailable {
		if targetTag != "" && normalizeReleaseTag(result.CurrentVersion) == normalizeReleaseTag(result.LatestVersion) {
			_, err := fmt.Fprintf(stdout, tr.T("already on requested version: %s")+"\n", formatReleaseTag(result.LatestVersion))
			return err
		}
		if targetTag != "" {
			_, err := fmt.Fprintf(stdout, tr.T("requested version %s is not newer than current %s")+"\n", formatReleaseTag(result.LatestVersion), formatReleaseTag(result.CurrentVersion))
			return err
		}
		_, err := fmt.Fprintf(stdout, tr.T("already up to date: %s")+"\n", formatReleaseTag(result.CurrentVersion))
		return err
	}

	if result.Release == nil {
		return fmt.Errorf("failed to install update: release metadata is unavailable")
	}

	if _, err := fmt.Fprintf(stdout, tr.T("updating from %s to %s")+"\n", formatReleaseTag(result.CurrentVersion), formatReleaseTag(result.LatestVersion)); err != nil {
		return err
	}
	if err := client.DownloadAndInstall(ctx, result.Release, stdout); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}
	_, err = fmt.Fprintf(stdout, tr.T("updated successfully to %s")+"\n", formatReleaseTag(result.LatestVersion))
	return err
}

func writeUpdateCheck(w io.Writer, result *updater.CheckResult, specific bool, tr *i18n.Translator) error {
	if result == nil {
		return fmt.Errorf("empty update check result")
	}
	switch {
	case result.UpdateAvailable && specific:
		_, err := fmt.Fprintf(w, tr.T("target release available: %s (current %s)")+"\n", formatReleaseTag(result.LatestVersion), formatReleaseTag(result.CurrentVersion))
		return err
	case result.UpdateAvailable:
		_, err := fmt.Fprintf(w, tr.T("update available: %s (current %s)")+"\n", formatReleaseTag(result.LatestVersion), formatReleaseTag(result.CurrentVersion))
		return err
	case specific && normalizeReleaseTag(result.CurrentVersion) == normalizeReleaseTag(result.LatestVersion):
		_, err := fmt.Fprintf(w, tr.T("already on requested version: %s")+"\n", formatReleaseTag(result.LatestVersion))
		return err
	case specific:
		_, err := fmt.Fprintf(w, tr.T("requested version %s is not newer than current %s")+"\n", formatReleaseTag(result.LatestVersion), formatReleaseTag(result.CurrentVersion))
		return err
	default:
		_, err := fmt.Fprintf(w, tr.T("already up to date: %s")+"\n", formatReleaseTag(result.CurrentVersion))
		return err
	}
}

func defaultUpdateRepo(meta vminfo.AppMetadata) string {
	repo := strings.TrimSpace(meta.Repository)
	if repo == "" {
		return "cloudapp3/vminfo"
	}
	if strings.Contains(repo, "://") {
		if parsed, err := url.Parse(repo); err == nil {
			repo = parsed.Path
		}
	}
	repo = strings.TrimSuffix(strings.Trim(repo, "/"), ".git")
	if repo == "" {
		return "cloudapp3/vminfo"
	}
	return repo
}

func updateTokenFromEnv() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GH_TOKEN"))
}

func normalizeReleaseTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "dev") {
		return v
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func formatReleaseTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	if strings.EqualFold(v, "dev") {
		return v
	}
	return normalizeReleaseTag(v)
}
