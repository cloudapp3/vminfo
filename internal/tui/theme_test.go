package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestDefaultThemeKeepsClassicPalette pins the startup look: tokyonight with
// vminfo's original hard-coded dark values.
func TestDefaultThemeKeepsClassicPalette(t *testing.T) {
	if activeTheme.Name != "tokyonight" {
		t.Fatalf("expected default theme tokyonight, got %s", activeTheme.Name)
	}
	want := map[string]lipgloss.AdaptiveColor{
		"CText":        {Dark: "#c0caf5", Light: "#343b58"},
		"CBorder":      {Dark: "#3b4261", Light: "#c4c8da"},
		"CBlue":        {Dark: "#7aa2f7", Light: "#2c5fb5"},
		"CPink":        {Dark: "#ff79c6", Light: "#c33b8d"},
		"CBrightGreen": {Dark: "#00ff87", Light: "#1f8a3d"},
		"CCritical":    {Dark: "#ff5555", Light: "#c61b1b"},
		"CBadgeBg":     {Dark: "#1a1b26", Light: "#e1e2e7"},
	}
	got := map[string]lipgloss.AdaptiveColor{
		"CText":        CText,
		"CBorder":      CBorder,
		"CBlue":        CBlue,
		"CPink":        CPink,
		"CBrightGreen": CBrightGreen,
		"CCritical":    CCritical,
		"CBadgeBg":     CBadgeBg,
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s = %+v, want %+v", name, got[name], w)
		}
	}
}

func TestSetThemeUpdatesActiveSlots(t *testing.T) {
	t.Cleanup(func() { SetTheme("tokyonight") })

	SetTheme("Nord")
	if activeTheme.Name != "nord" {
		t.Fatalf("expected theme names to be case-insensitive, got %s", activeTheme.Name)
	}
	if CText.Dark != "#eceff4" {
		t.Errorf("expected CText to follow nord, got %+v", CText)
	}

	SetTheme("no-such-theme")
	if activeTheme.Name != "nord" {
		t.Errorf("unknown theme must be ignored, got %s", activeTheme.Name)
	}
}

func TestCycleThemeFollowsThemeOrder(t *testing.T) {
	t.Cleanup(func() { SetTheme("tokyonight") })

	SetTheme("tokyonight")
	CycleTheme()
	if activeTheme.Name != "catppuccin" {
		t.Fatalf("expected catppuccin after tokyonight, got %s", activeTheme.Name)
	}

	SetTheme("monochrome")
	CycleTheme()
	if activeTheme.Name != "dracula" {
		t.Fatalf("expected cycle to wrap to dracula, got %s", activeTheme.Name)
	}
}

func TestInitThemeEnvWinsOverPersisted(t *testing.T) {
	t.Cleanup(func() { SetTheme("tokyonight") })

	t.Setenv("VMINFO_THEME", "gruvbox")
	InitTheme("nord")
	if activeTheme.Name != "gruvbox" {
		t.Fatalf("expected env theme gruvbox, got %s", activeTheme.Name)
	}

	t.Setenv("VMINFO_THEME", "")
	InitTheme("nord")
	if activeTheme.Name != "nord" {
		t.Fatalf("expected persisted theme nord, got %s", activeTheme.Name)
	}

	InitTheme("")
	if activeTheme.Name != "nord" {
		t.Fatalf("expected empty preference to keep current theme, got %s", activeTheme.Name)
	}
}

func TestThemeNamesCoversAllThemes(t *testing.T) {
	names := ThemeNames()
	if len(names) != len(Themes) {
		t.Fatalf("expected %d names, got %d", len(Themes), len(names))
	}
	for _, n := range names {
		if _, ok := Themes[n]; !ok {
			t.Errorf("ThemeNames returned unknown theme %s", n)
		}
	}
}
