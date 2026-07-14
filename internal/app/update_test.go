package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/cloudapp3/vminfo/internal/updater"
)

type stubUpdateClient struct {
	checkCalled         bool
	checkSpecificCalled bool
	checkSpecificTag    string
	checkSpecificResult *updater.CheckResult
	checkSpecificErr    error
	useSpecificResult   bool
	downloadCalled      bool
	downloadRelease     *updater.Release
	checkResult         *updater.CheckResult
	checkErr            error
	downloadErr         error
}

func (s *stubUpdateClient) CheckForUpdate(context.Context) (*updater.CheckResult, error) {
	s.checkCalled = true
	return s.checkResult, s.checkErr
}

func (s *stubUpdateClient) CheckSpecificVersion(_ context.Context, tag string) (*updater.CheckResult, error) {
	s.checkSpecificCalled = true
	s.checkSpecificTag = tag
	if s.useSpecificResult {
		return s.checkSpecificResult, s.checkSpecificErr
	}
	return s.checkResult, s.checkErr
}

func (s *stubUpdateClient) DownloadAndInstall(_ context.Context, release *updater.Release, progress io.Writer) error {
	s.downloadCalled = true
	s.downloadRelease = release
	if progress != nil {
		_, _ = progress.Write([]byte("installing...\n"))
	}
	return s.downloadErr
}

func TestRunUpdateCheckRoutesThroughUpdater(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	restoreRepo := vminfo.Repository
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
		vminfo.Repository = restoreRepo
	})

	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
	}
	newUpdateClient = func(cfg updater.Config) updateClient {
		if cfg.Repo != "cloudapp3/vminfo" {
			t.Fatalf("unexpected repo: %s", cfg.Repo)
		}
		return stub
	}
	vminfo.Version = "v1.0.0"
	vminfo.Repository = "https://github.com/cloudapp3/vminfo"

	var stdout bytes.Buffer
	if err := Run(context.Background(), []string{"update", "--check"}, &stdout, new(bytes.Buffer)); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !stub.checkCalled {
		t.Fatal("expected CheckForUpdate to be called")
	}
	if got := stdout.String(); !strings.Contains(got, "update available: v1.1.0 (current v1.0.0)") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunUpdateWrapsFlagErrorsAsUsage(t *testing.T) {
	err := runUpdate(context.Background(), new(bytes.Buffer), new(bytes.Buffer), []string{"--unknown"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("runUpdate error = %v, want ErrUsage", err)
	}
}

func TestRunUpdateNormalizesSpecificVersion(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
	})

	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.2.3",
			UpdateAvailable: true,
			Release:         &updater.Release{TagName: "v1.2.3"},
		},
	}
	newUpdateClient = func(cfg updater.Config) updateClient {
		return stub
	}
	vminfo.Version = "v1.0.0"

	var stdout bytes.Buffer
	if err := runUpdate(context.Background(), &stdout, new(bytes.Buffer), []string{"--check", "--version", "1.2.3"}, i18n.New("en")); err != nil {
		t.Fatalf("runUpdate returned error: %v", err)
	}
	if !stub.checkSpecificCalled {
		t.Fatal("expected CheckSpecificVersion to be called")
	}
	if stub.checkSpecificTag != "v1.2.3" {
		t.Fatalf("expected normalized tag v1.2.3, got %q", stub.checkSpecificTag)
	}
}

func TestRunUpdateInstallsAvailableRelease(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
	})

	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
			Release:         &updater.Release{TagName: "v1.1.0"},
		},
	}
	newUpdateClient = func(cfg updater.Config) updateClient {
		return stub
	}
	vminfo.Version = "v1.0.0"

	var stdout bytes.Buffer
	if err := runUpdate(context.Background(), &stdout, new(bytes.Buffer), nil, i18n.New("en")); err != nil {
		t.Fatalf("runUpdate returned error: %v", err)
	}
	if !stub.downloadCalled {
		t.Fatal("expected DownloadAndInstall to be called")
	}
	if got := stdout.String(); !strings.Contains(got, "updated successfully to v1.1.0") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunUpdateInstallsReleaseAfterCacheHit(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
	})

	release := &updater.Release{TagName: "v1.1.0"}
	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
		checkSpecificResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
			Release:         release,
		},
		useSpecificResult: true,
	}
	newUpdateClient = func(updater.Config) updateClient {
		return stub
	}
	vminfo.Version = "v1.0.0"

	var stdout bytes.Buffer
	if err := runUpdate(context.Background(), &stdout, new(bytes.Buffer), nil, i18n.New("en")); err != nil {
		t.Fatalf("runUpdate returned error: %v", err)
	}
	if !stub.checkCalled {
		t.Fatal("expected CheckForUpdate to be called")
	}
	if !stub.checkSpecificCalled {
		t.Fatal("expected CheckSpecificVersion to be called")
	}
	if stub.checkSpecificTag != "v1.1.0" {
		t.Fatalf("expected normalized tag v1.1.0, got %q", stub.checkSpecificTag)
	}
	if stub.downloadRelease != release {
		t.Fatalf("DownloadAndInstall received release %p, want %p", stub.downloadRelease, release)
	}
	if got := stdout.String(); !strings.Contains(got, "updated successfully to v1.1.0") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunUpdateRejectsInvalidCachedReleaseMetadata(t *testing.T) {
	tests := []struct {
		name            string
		checkResult     *updater.CheckResult
		specificResult  *updater.CheckResult
		specificErr     error
		wantSpecificTag string
		wantErr         string
	}{
		{
			name: "empty cached version",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				UpdateAvailable: true,
			},
			wantErr: "release metadata is unavailable for version \"\"",
		},
		{
			name: "specific lookup error",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			specificErr:     errors.New("offline"),
			wantSpecificTag: "v1.1.0",
			wantErr:         "failed to fetch release metadata for v1.1.0: offline",
		},
		{
			name: "empty specific result",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			wantSpecificTag: "v1.1.0",
			wantErr:         "failed to fetch release metadata for v1.1.0: empty result",
		},
		{
			name: "mismatched specific version",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			specificResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.2.0",
				UpdateAvailable: true,
				Release:         &updater.Release{TagName: "v1.2.0"},
			},
			wantSpecificTag: "v1.1.0",
			wantErr:         "returned version is v1.2.0",
		},
		{
			name: "missing release",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			specificResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			wantSpecificTag: "v1.1.0",
			wantErr:         "release metadata is unavailable",
		},
		{
			name: "mismatched release tag",
			checkResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
			specificResult: &updater.CheckResult{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
				Release:         &updater.Release{TagName: "v1.2.0"},
			},
			wantSpecificTag: "v1.1.0",
			wantErr:         "release tag is v1.2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreClient := newUpdateClient
			restoreVersion := vminfo.Version
			t.Cleanup(func() {
				newUpdateClient = restoreClient
				vminfo.Version = restoreVersion
			})

			stub := &stubUpdateClient{
				checkResult:         tt.checkResult,
				checkSpecificResult: tt.specificResult,
				checkSpecificErr:    tt.specificErr,
				useSpecificResult:   true,
			}
			newUpdateClient = func(updater.Config) updateClient {
				return stub
			}
			vminfo.Version = "v1.0.0"

			err := runUpdate(context.Background(), new(bytes.Buffer), new(bytes.Buffer), nil, i18n.New("en"))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("runUpdate error = %v, want substring %q", err, tt.wantErr)
			}
			if stub.checkSpecificTag != tt.wantSpecificTag {
				t.Fatalf("CheckSpecificVersion tag = %q, want %q", stub.checkSpecificTag, tt.wantSpecificTag)
			}
			if stub.downloadCalled {
				t.Fatal("DownloadAndInstall should not be called")
			}
		})
	}
}

func TestRunUpdateRechecksCachedVersionBeforeInstall(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
	})

	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
		checkSpecificResult: &updater.CheckResult{
			CurrentVersion:  "1.1.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: false,
			Release:         &updater.Release{TagName: "v1.1.0"},
		},
		useSpecificResult: true,
	}
	newUpdateClient = func(updater.Config) updateClient {
		return stub
	}
	vminfo.Version = "v1.0.0"

	var stdout bytes.Buffer
	if err := runUpdate(context.Background(), &stdout, new(bytes.Buffer), nil, i18n.New("en")); err != nil {
		t.Fatalf("runUpdate returned error: %v", err)
	}
	if stub.downloadCalled {
		t.Fatal("DownloadAndInstall should not be called when the refreshed result is current")
	}
	if got := stdout.String(); !strings.Contains(got, "already up to date: v1.1.0") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunUpdateCheckAllowsDevBuild(t *testing.T) {
	restoreClient := newUpdateClient
	restoreVersion := vminfo.Version
	t.Cleanup(func() {
		newUpdateClient = restoreClient
		vminfo.Version = restoreVersion
	})

	stub := &stubUpdateClient{
		checkResult: &updater.CheckResult{
			CurrentVersion:  "dev",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
	}
	newUpdateClient = func(cfg updater.Config) updateClient {
		return stub
	}
	vminfo.Version = "dev"

	var stdout bytes.Buffer
	if err := runUpdate(context.Background(), &stdout, new(bytes.Buffer), []string{"--check"}, i18n.New("en")); err != nil {
		t.Fatalf("runUpdate returned error: %v", err)
	}
	if !stub.checkCalled {
		t.Fatal("expected CheckForUpdate to be called")
	}
	if got := stdout.String(); !strings.Contains(got, "update available: v1.1.0 (current dev)") {
		t.Fatalf("unexpected output: %q", got)
	}
}
