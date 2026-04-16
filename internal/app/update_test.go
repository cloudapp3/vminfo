package app

import (
	"bytes"
	"context"
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
	downloadCalled      bool
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
	return s.checkResult, s.checkErr
}

func (s *stubUpdateClient) DownloadAndInstall(_ context.Context, _ *updater.Release, progress io.Writer) error {
	s.downloadCalled = true
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
