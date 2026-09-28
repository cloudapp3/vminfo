package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PersistedConfig holds TUI preferences saved between sessions.
type PersistedConfig struct {
	Theme string `json:"theme,omitempty"`
}

// ConfigPaths returns the TUI preferences file path and the vminfo-owned
// directory containing it. Dir is empty when VMINFO_CONFIG redirected the
// file elsewhere (only the file is vminfo-owned then) or when the platform
// has no config location.
func ConfigPaths() (file, dir string) {
	if v := os.Getenv("VMINFO_CONFIG"); v != "" {
		return v, ""
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", ""
	}
	return filepath.Join(base, "vminfo", "config.json"), filepath.Join(base, "vminfo")
}

func configPath() string {
	file, _ := ConfigPaths()
	return file
}

func LoadConfig() PersistedConfig {
	var cfg PersistedConfig
	p := configPath()
	if p == "" {
		return cfg
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func SaveConfig(cfg PersistedConfig) error {
	p := configPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
