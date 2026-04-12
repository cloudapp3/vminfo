package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/VPSMarket/vminfo/pkg/vminfo"
)

func TestRunVersionText(t *testing.T) {
	t.Setenv("TERM", "dumb")

	originalVersion := vminfo.Version
	originalCommit := vminfo.Commit
	originalBuildTime := vminfo.BuildTime
	originalChannel := vminfo.Channel
	t.Cleanup(func() {
		vminfo.Version = originalVersion
		vminfo.Commit = originalCommit
		vminfo.BuildTime = originalBuildTime
		vminfo.Channel = originalChannel
	})

	vminfo.Version = "v0.1.0"
	vminfo.Commit = "abc1234"
	vminfo.BuildTime = "2026-04-12T13:00:00Z"
	vminfo.Channel = "stable"

	var stdout strings.Builder
	var stderr strings.Builder
	if err := Run(context.Background(), []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run(version) error = %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"vminfo v0.1.0",
		"commit: abc1234",
		"built:  2026-04-12T13:00:00Z",
		"channel: stable",
		"schema: v1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output missing %q\noutput:\n%s", want, got)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
}

func TestRunVersionJSONViaGlobalFlag(t *testing.T) {
	t.Setenv("TERM", "dumb")

	originalVersion := vminfo.Version
	originalCommit := vminfo.Commit
	originalBuildTime := vminfo.BuildTime
	originalChannel := vminfo.Channel
	t.Cleanup(func() {
		vminfo.Version = originalVersion
		vminfo.Commit = originalCommit
		vminfo.BuildTime = originalBuildTime
		vminfo.Channel = originalChannel
	})

	vminfo.Version = "v0.2.0"
	vminfo.Commit = "deadbee"
	vminfo.BuildTime = "2026-04-12T15:00:00Z"
	vminfo.Channel = "nightly"

	var stdout strings.Builder
	var stderr strings.Builder
	if err := Run(context.Background(), []string{"--version", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run(--version --json) error = %v", err)
	}

	var meta vminfo.AppMetadata
	if err := json.Unmarshal([]byte(stdout.String()), &meta); err != nil {
		t.Fatalf("json.Unmarshal error = %v\nbody=%s", err, stdout.String())
	}
	if meta.Name != vminfo.AppName {
		t.Fatalf("meta.Name = %q, want %q", meta.Name, vminfo.AppName)
	}
	if meta.Version != "v0.2.0" {
		t.Fatalf("meta.Version = %q, want v0.2.0", meta.Version)
	}
	if meta.Commit != "deadbee" {
		t.Fatalf("meta.Commit = %q, want deadbee", meta.Commit)
	}
	if meta.BuildTime != "2026-04-12T15:00:00Z" {
		t.Fatalf("meta.BuildTime = %q", meta.BuildTime)
	}
	if meta.Channel != "nightly" {
		t.Fatalf("meta.Channel = %q, want nightly", meta.Channel)
	}
	if meta.SchemaVersion != vminfo.DefaultSchemaVersion {
		t.Fatalf("meta.SchemaVersion = %q, want %q", meta.SchemaVersion, vminfo.DefaultSchemaVersion)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
}

func TestHelpTextMentionsVersion(t *testing.T) {
	help := helpText()
	for _, want := range []string{
		"vminfo version         show app version",
		"vminfo version --json  show app metadata as JSON",
		"vminfo --version       show app version",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help text missing %q\nhelp:\n%s", want, help)
		}
	}
}
