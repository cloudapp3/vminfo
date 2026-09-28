package tui

import (
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("VMINFO_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	if got := LoadConfig().Theme; got != "" {
		t.Fatalf("expected empty config for a fresh path, got %q", got)
	}
	if err := SaveConfig(PersistedConfig{Theme: "nord"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if got := LoadConfig().Theme; got != "nord" {
		t.Fatalf("expected persisted theme nord, got %q", got)
	}
}

func TestConfigPathsHonorsEnvOverride(t *testing.T) {
	t.Setenv("VMINFO_CONFIG", "/tmp/alt.json")
	file, dir := ConfigPaths()
	if file != "/tmp/alt.json" {
		t.Errorf("expected env override, got file %q", file)
	}
	if dir != "" {
		t.Errorf("expected empty dir for env-redirected file, got %q", dir)
	}
}
