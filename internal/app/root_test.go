package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/cloudapp3/vminfo/internal/updater"
)

func TestParseGlobalOptionsWebFlagsAreOrderIndependent(t *testing.T) {
	opts, remaining, err := parseGlobalOptions([]string{
		"--interval", "750ms", "--port=8080", "--bind", "0.0.0.0", "--token", "secret", "--web",
	})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining args = %v, want none", remaining)
	}
	if !opts.web || opts.webPort != 8080 || opts.webBind != "0.0.0.0" || opts.webToken != "secret" {
		t.Fatalf("unexpected options: %+v", opts)
	}
	if opts.webInterval != 750*time.Millisecond {
		t.Fatalf("web interval = %s, want 750ms", opts.webInterval)
	}
}

func TestParseGlobalOptionsRejectsInvalidWebValues(t *testing.T) {
	for _, args := range [][]string{
		{"--web", "--port", "nope"},
		{"--web", "--port=0"},
		{"--web", "--bind="},
		{"--web", "--interval", "0s"},
		{"--bind", "127.0.0.1"},
	} {
		t.Run(args[1], func(t *testing.T) {
			if _, _, err := parseGlobalOptions(args); !errors.Is(err, ErrUsage) {
				t.Fatalf("parseGlobalOptions(%v) error = %v, want ErrUsage", args, err)
			}
		})
	}
}

func TestParseGlobalOptionsHonorsFlagTerminator(t *testing.T) {
	opts, remaining, err := parseGlobalOptions([]string{"ps", "--", "--token", "--web", "--interval", "1s"})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if opts.web || opts.webOptionSeen {
		t.Fatalf("options after -- were parsed as globals: %+v", opts)
	}
	want := []string{"ps", "--", "--token", "--web", "--interval", "1s"}
	if !slices.Equal(remaining, want) {
		t.Fatalf("remaining args = %v, want %v", remaining, want)
	}

	_, remaining, err = parseGlobalOptions([]string{"--", "ps", "--token"})
	if err != nil {
		t.Fatalf("parseGlobalOptions with leading terminator returned error: %v", err)
	}
	if want := []string{"ps", "--token"}; !slices.Equal(remaining, want) {
		t.Fatalf("remaining args after leading -- = %v, want %v", remaining, want)
	}
}

func TestValidateWebExposure(t *testing.T) {
	for _, bind := range []string{"127.0.0.1", "::1", "[::1]", "localhost"} {
		if err := validateWebExposure(bind, ""); err != nil {
			t.Fatalf("validateWebExposure(%q) returned error: %v", bind, err)
		}
	}
	if err := validateWebExposure("0.0.0.0", ""); !errors.Is(err, ErrUsage) {
		t.Fatalf("wildcard bind error = %v, want ErrUsage", err)
	}
	if err := validateWebExposure("0.0.0.0", "secret"); err != nil {
		t.Fatalf("token-protected wildcard bind returned error: %v", err)
	}
}

func TestParsePIDRejectsOverflow(t *testing.T) {
	if _, err := parsePID("4294967298"); !errors.Is(err, ErrUsage) {
		t.Fatalf("overflow PID error = %v, want ErrUsage", err)
	}
	if _, err := parsePID("0"); !errors.Is(err, ErrUsage) {
		t.Fatalf("zero PID error = %v, want ErrUsage", err)
	}
	pid, err := parsePID("42")
	if err != nil || pid != 42 {
		t.Fatalf("parsePID(42) = %d, %v", pid, err)
	}
}

func TestRunRejectsUnsafeOrUnknownWebArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--web", "--bind", "0.0.0.0", "--silent"},
		{"--web", "--unexpected"},
	} {
		err := Run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, ErrUsage) {
			t.Fatalf("Run(%v) error = %v, want ErrUsage", args, err)
		}
	}
}

func TestRunWrapsFlagErrorsAsUsage(t *testing.T) {
	err := Run(context.Background(), []string{"summary", "--bogus"}, &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Run error = %v, want ErrUsage", err)
	}
}

func TestSummaryTextRemovesTerminalControlPayloads(t *testing.T) {
	staticInfo := vminfo.StaticInfo{
		Hostname:  "safe-host\x1b]0;hostname-payload\a",
		Platform:  "linux\x1b]0;platform-payload\a",
		OSVersion: "12\x1b]0;version-payload\a",
		Kernel:    "6.1\x1b]0;kernel-payload\a",
		Arch:      "amd64\x1b]0;arch-payload\a",
		CPUModel:  "example-cpu\x1b]0;cpu-payload\a",
		CPUCores:  4,
	}
	payloads := []string{
		"hostname-payload",
		"platform-payload",
		"version-payload",
		"kernel-payload",
		"arch-payload",
		"cpu-payload",
	}

	var summary bytes.Buffer
	if err := writeSummary(&summary, staticInfo, vminfo.RuntimeStats{}, i18n.New("en")); err != nil {
		t.Fatalf("writeSummary returned error: %v", err)
	}
	assertNoTerminalPayloads(t, summary.String(), payloads)
	for _, want := range []string{"safe-host", "linux 12", "6.1", "amd64", "example-cpu"} {
		if !strings.Contains(summary.String(), want) {
			t.Fatalf("summary output %q does not contain sanitized value %q", summary.String(), want)
		}
	}

	var watch bytes.Buffer
	if err := writeWatchSnapshot(&watch, time.Unix(0, 0).UTC(), staticInfo, vminfo.RuntimeStats{}, i18n.New("en")); err != nil {
		t.Fatalf("writeWatchSnapshot returned error: %v", err)
	}
	assertNoTerminalPayloads(t, watch.String(), payloads[:3])
	if !strings.Contains(watch.String(), "host=safe-host os=linux 12") {
		t.Fatalf("watch output does not contain sanitized host and OS: %q", watch.String())
	}
}

func assertNoTerminalPayloads(t *testing.T, output string, payloads []string) {
	t.Helper()
	for _, payload := range payloads {
		if strings.Contains(output, payload) {
			t.Fatalf("output exposed terminal control payload %q: %q", payload, output)
		}
	}
}

func TestBackgroundUpdateCleanupDoesNotWaitForever(t *testing.T) {
	restoreClient := newUpdateClient
	t.Cleanup(func() { newUpdateClient = restoreClient })

	client := &blockingUpdateClient{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		exited:  make(chan struct{}),
	}
	newUpdateClient = func(updater.Config) updateClient { return client }

	cleanup := startBackgroundUpdateCheck(
		context.Background(),
		io.Discard,
		i18n.New("en"),
		vminfo.AppMetadata{Version: "1.0.0"},
	)
	select {
	case <-client.entered:
	case <-time.After(time.Second):
		t.Fatal("background update check did not start")
	}

	started := time.Now()
	cleanup()
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cleanup blocked for %s", elapsed)
	}

	close(client.release)
	select {
	case <-client.exited:
	case <-time.After(time.Second):
		t.Fatal("background update check did not exit after release")
	}
}

type blockingUpdateClient struct {
	entered chan struct{}
	release chan struct{}
	exited  chan struct{}
}

func (c *blockingUpdateClient) CheckForUpdate(ctx context.Context) (*updater.CheckResult, error) {
	close(c.entered)
	<-c.release
	close(c.exited)
	return nil, ctx.Err()
}

func (*blockingUpdateClient) CheckSpecificVersion(context.Context, string) (*updater.CheckResult, error) {
	return nil, errors.New("unexpected CheckSpecificVersion call")
}

func (*blockingUpdateClient) DownloadAndInstall(context.Context, *updater.Release, io.Writer) error {
	return errors.New("unexpected DownloadAndInstall call")
}
