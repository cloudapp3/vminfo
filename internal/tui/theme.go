package tui

import (
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Theme ────────────────────────────────────────────────────────────

// Theme is one full palette for the dashboard. Every color is adaptive so
// each theme carries dark- and light-terminal variants, like vmbench.
type Theme struct {
	Name string

	// Base palette
	Text   lipgloss.AdaptiveColor // primary text
	Dim    lipgloss.AdaptiveColor // muted labels / secondary text
	Border lipgloss.AdaptiveColor // panel borders / separators
	Muted  lipgloss.AdaptiveColor // idle bar parts / dim glyphs
	Info   lipgloss.AdaptiveColor // key hints

	// Accent colors
	Cyan        lipgloss.AdaptiveColor
	Green       lipgloss.AdaptiveColor
	Yellow      lipgloss.AdaptiveColor
	Red         lipgloss.AdaptiveColor
	Purple      lipgloss.AdaptiveColor
	Blue        lipgloss.AdaptiveColor
	Pink        lipgloss.AdaptiveColor // upload / Tx
	BrightGreen lipgloss.AdaptiveColor // download / Rx

	// 4-tier threshold colors (0-50 / 50-75 / 75-90 / 90-100)
	Ok       lipgloss.AdaptiveColor
	Warn     lipgloss.AdaptiveColor
	Alert    lipgloss.AdaptiveColor
	Critical lipgloss.AdaptiveColor

	// Backgrounds
	BadgeBg   lipgloss.AdaptiveColor // header badge background
	SurfaceBg lipgloss.AdaptiveColor // process column header background
	OddBg     lipgloss.AdaptiveColor // zebra row odd background
	EvenBg    lipgloss.AdaptiveColor // zebra row even background
	SelectBg  lipgloss.AdaptiveColor // selected row background
}

func ac(dark, light string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Dark: dark, Light: light}
}

// Themes mirrors vmbench's built-in set. The "tokyonight" dark values are
// vminfo's original hard-coded palette, so the default look is unchanged.
var Themes = map[string]Theme{
	"tokyonight": {
		Name:   "tokyonight",
		Text:   ac("#c0caf5", "#343b58"),
		Dim:    ac("#565f89", "#6c7186"),
		Border: ac("#3b4261", "#c4c8da"),
		Muted:  ac("#6c6c6c", "#8990b3"),
		Info:   ac("#5fafff", "#2c5fb5"),

		Cyan:        ac("#7dcfff", "#0a8ab5"),
		Green:       ac("#9ece6a", "#587a39"),
		Yellow:      ac("#e0af68", "#a07028"),
		Red:         ac("#f7768e", "#c43955"),
		Purple:      ac("#bb9af7", "#7748b8"),
		Blue:        ac("#7aa2f7", "#2c5fb5"),
		Pink:        ac("#ff79c6", "#c33b8d"),
		BrightGreen: ac("#00ff87", "#1f8a3d"),

		Ok:       ac("#00ff87", "#1f8a3d"),
		Warn:     ac("#ffd700", "#a8810b"),
		Alert:    ac("#ffaf5f", "#c46c00"),
		Critical: ac("#ff5555", "#c61b1b"),

		BadgeBg:   ac("#1a1b26", "#e1e2e7"),
		SurfaceBg: ac("#2a2a3e", "#d5d6db"),
		OddBg:     ac("#1a1a2e", "#e1e2e7"),
		EvenBg:    ac("#16213e", "#e9eaf0"),
		SelectBg:  ac("#2d4f67", "#bcd2f0"),
	},

	"dracula": {
		Name:   "dracula",
		Text:   ac("#f8f8f2", "#282a36"),
		Dim:    ac("#6272a4", "#7c7f9b"),
		Border: ac("#44475a", "#bcbcbc"),
		Muted:  ac("#44475a", "#9aa0b5"),
		Info:   ac("#8be9fd", "#0aa6c2"),

		Cyan:        ac("#8be9fd", "#0aa6c2"),
		Green:       ac("#50fa7b", "#1f8a3d"),
		Yellow:      ac("#f1fa8c", "#a07f00"),
		Red:         ac("#ff5555", "#c61b1b"),
		Purple:      ac("#bd93f9", "#6c4ad9"),
		Blue:        ac("#bd93f9", "#6c4ad9"),
		Pink:        ac("#ff79c6", "#c33b8d"),
		BrightGreen: ac("#50fa7b", "#1f8a3d"),

		Ok:       ac("#50fa7b", "#1f8a3d"),
		Warn:     ac("#f1fa8c", "#a07f00"),
		Alert:    ac("#ffb86c", "#c46c00"),
		Critical: ac("#ff5555", "#c61b1b"),

		BadgeBg:   ac("#282a36", "#f8f8f2"),
		SurfaceBg: ac("#343746", "#eaeaea"),
		OddBg:     ac("#2b2d3a", "#eaeaea"),
		EvenBg:    ac("#31343f", "#f2f2f0"),
		SelectBg:  ac("#44475a", "#d8dcf5"),
	},

	"catppuccin": {
		Name:   "catppuccin",
		Text:   ac("#cdd6f4", "#4c4f69"),
		Dim:    ac("#7f849c", "#6c6f85"),
		Border: ac("#45475a", "#bcc0cc"),
		Muted:  ac("#6c7086", "#9a9cb0"),
		Info:   ac("#89dceb", "#04a5e5"),

		Cyan:        ac("#89dceb", "#04a5e5"),
		Green:       ac("#a6e3a1", "#40a02b"),
		Yellow:      ac("#f9e2af", "#df8e1d"),
		Red:         ac("#f38ba8", "#d20f39"),
		Purple:      ac("#cba6f7", "#8839ef"),
		Blue:        ac("#89b4fa", "#1e66f5"),
		Pink:        ac("#f5c2e7", "#ea76cb"),
		BrightGreen: ac("#a6e3a1", "#40a02b"),

		Ok:       ac("#a6e3a1", "#40a02b"),
		Warn:     ac("#f9e2af", "#df8e1d"),
		Alert:    ac("#fab387", "#fe640b"),
		Critical: ac("#f38ba8", "#d20f39"),

		BadgeBg:   ac("#1e1e2e", "#eff1f5"),
		SurfaceBg: ac("#313244", "#e6e9ef"),
		OddBg:     ac("#262637", "#e6e9ef"),
		EvenBg:    ac("#2a2a3c", "#eceef4"),
		SelectBg:  ac("#45475a", "#dce0e8"),
	},

	"nord": {
		Name:   "nord",
		Text:   ac("#eceff4", "#2e3440"),
		Dim:    ac("#7b88a1", "#6c7585"),
		Border: ac("#4c566a", "#aeb5be"),
		Muted:  ac("#5c6577", "#94a0b3"),
		Info:   ac("#88c0d0", "#3a7d92"),

		Cyan:        ac("#8fbcbb", "#3f8f8d"),
		Green:       ac("#a3be8c", "#577f3a"),
		Yellow:      ac("#ebcb8b", "#a8810b"),
		Red:         ac("#bf616a", "#9b3138"),
		Purple:      ac("#b48ead", "#76507b"),
		Blue:        ac("#88c0d0", "#3a7d92"),
		Pink:        ac("#b48ead", "#76507b"),
		BrightGreen: ac("#a3be8c", "#577f3a"),

		Ok:       ac("#a3be8c", "#577f3a"),
		Warn:     ac("#ebcb8b", "#a8810b"),
		Alert:    ac("#d08770", "#a14a30"),
		Critical: ac("#bf616a", "#9b3138"),

		BadgeBg:   ac("#2e3440", "#eceff4"),
		SurfaceBg: ac("#3b4252", "#e5e9f0"),
		OddBg:     ac("#353c4a", "#e5e9f0"),
		EvenBg:    ac("#39404f", "#ebedf2"),
		SelectBg:  ac("#434c5e", "#d3dae4"),
	},

	"gruvbox": {
		Name:   "gruvbox",
		Text:   ac("#ebdbb2", "#3c3836"),
		Dim:    ac("#928374", "#7c6f64"),
		Border: ac("#504945", "#bdae93"),
		Muted:  ac("#665c54", "#928374"),
		Info:   ac("#83a598", "#427b58"),

		Cyan:        ac("#83a598", "#427b58"),
		Green:       ac("#b8bb26", "#79740e"),
		Yellow:      ac("#fabd2f", "#b57614"),
		Red:         ac("#fb4934", "#9d0006"),
		Purple:      ac("#d3869b", "#8f3f71"),
		Blue:        ac("#458588", "#076678"),
		Pink:        ac("#d3869b", "#8f3f71"),
		BrightGreen: ac("#b8bb26", "#79740e"),

		Ok:       ac("#b8bb26", "#79740e"),
		Warn:     ac("#fabd2f", "#b57614"),
		Alert:    ac("#fe8019", "#af3a03"),
		Critical: ac("#fb4934", "#9d0006"),

		BadgeBg:   ac("#282828", "#fbf1c7"),
		SurfaceBg: ac("#3c3836", "#ebdbb2"),
		OddBg:     ac("#32302f", "#ebdbb2"),
		EvenBg:    ac("#363332", "#f0e2bd"),
		SelectBg:  ac("#504945", "#d5c4a1"),
	},

	"rose-pine": {
		Name:   "rose-pine",
		Text:   ac("#e0def4", "#575279"),
		Dim:    ac("#908caa", "#797593"),
		Border: ac("#26233a", "#9893a5"),
		Muted:  ac("#6e6a86", "#8f8ca3"),
		Info:   ac("#9ccfd8", "#56949f"),

		Cyan:        ac("#9ccfd8", "#56949f"),
		Green:       ac("#31748f", "#286983"),
		Yellow:      ac("#f6c177", "#ea9d34"),
		Red:         ac("#eb6f92", "#b4637a"),
		Purple:      ac("#c4a7e7", "#907aa9"),
		Blue:        ac("#c4a7e7", "#907aa9"),
		Pink:        ac("#ebbcba", "#d7827e"),
		BrightGreen: ac("#9ccfd8", "#56949f"),

		Ok:       ac("#31748f", "#286983"),
		Warn:     ac("#f6c177", "#ea9d34"),
		Alert:    ac("#ebbcba", "#d7827e"),
		Critical: ac("#eb6f92", "#b4637a"),

		BadgeBg:   ac("#191724", "#faf4ed"),
		SurfaceBg: ac("#1f1d2e", "#fffaf3"),
		OddBg:     ac("#1c1a2b", "#fffaf3"),
		EvenBg:    ac("#211f30", "#f6efe8"),
		SelectBg:  ac("#26233a", "#e8e0d8"),
	},

	"solarized": {
		Name:   "solarized",
		Text:   ac("#eee8d5", "#073642"),
		Dim:    ac("#586e75", "#93a1a1"),
		Border: ac("#586e75", "#93a1a1"),
		Muted:  ac("#657b83", "#839496"),
		Info:   ac("#268bd2", "#268bd2"),

		Cyan:        ac("#2aa198", "#2aa198"),
		Green:       ac("#859900", "#859900"),
		Yellow:      ac("#b58900", "#b58900"),
		Red:         ac("#dc322f", "#dc322f"),
		Purple:      ac("#6c71c4", "#6c71c4"),
		Blue:        ac("#268bd2", "#268bd2"),
		Pink:        ac("#d33682", "#d33682"),
		BrightGreen: ac("#859900", "#859900"),

		Ok:       ac("#859900", "#859900"),
		Warn:     ac("#b58900", "#b58900"),
		Alert:    ac("#cb4b16", "#cb4b16"),
		Critical: ac("#dc322f", "#dc322f"),

		BadgeBg:   ac("#002b36", "#fdf6e3"),
		SurfaceBg: ac("#073642", "#eee8d5"),
		OddBg:     ac("#06303c", "#eee8d5"),
		EvenBg:    ac("#053844", "#f2ebdc"),
		SelectBg:  ac("#586e75", "#93a1a1"),
	},

	"monochrome": {
		Name:   "monochrome",
		Text:   ac("#fafafa", "#171717"),
		Dim:    ac("#a3a3a3", "#525252"),
		Border: ac("#404040", "#a3a3a3"),
		Muted:  ac("#525252", "#a3a3a3"),
		Info:   ac("#d4d4d4", "#404040"),

		Cyan:        ac("#d4d4d4", "#404040"),
		Green:       ac("#bababa", "#404040"),
		Yellow:      ac("#ededed", "#262626"),
		Red:         ac("#fafafa", "#0a0a0a"),
		Purple:      ac("#d4d4d4", "#404040"),
		Blue:        ac("#fafafa", "#171717"),
		Pink:        ac("#a3a3a3", "#525252"),
		BrightGreen: ac("#bababa", "#404040"),

		Ok:       ac("#bababa", "#404040"),
		Warn:     ac("#ededed", "#262626"),
		Alert:    ac("#d4d4d4", "#404040"),
		Critical: ac("#fafafa", "#0a0a0a"),

		BadgeBg:   ac("#0a0a0a", "#fafafa"),
		SurfaceBg: ac("#171717", "#ededed"),
		OddBg:     ac("#141414", "#ededed"),
		EvenBg:    ac("#181818", "#e5e5e5"),
		SelectBg:  ac("#262626", "#d4d4d4"),
	},
}

// ThemeOrder is the cycle order for the T key, matching vmbench.
var ThemeOrder = []string{
	"dracula",
	"tokyonight",
	"catppuccin",
	"nord",
	"gruvbox",
	"rose-pine",
	"solarized",
	"monochrome",
}

// activeTheme is the current theme; default keeps vminfo's classic look.
var activeTheme = Themes["tokyonight"]

// ActiveTheme returns the current theme.
func ActiveTheme() Theme {
	return activeTheme
}

// SetTheme switches to the named theme, ignoring unknown names.
func SetTheme(name string) {
	if t, ok := Themes[strings.ToLower(strings.TrimSpace(name))]; ok {
		applyTheme(t)
	}
}

// CycleTheme advances to the next theme in ThemeOrder, wrapping.
func CycleTheme() {
	for i, n := range ThemeOrder {
		if n == activeTheme.Name {
			applyTheme(Themes[ThemeOrder[(i+1)%len(ThemeOrder)]])
			return
		}
	}
	applyTheme(Themes[ThemeOrder[0]])
}

// ThemeNames lists available theme names sorted alphabetically.
func ThemeNames() []string {
	out := make([]string, 0, len(Themes))
	for k := range Themes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// InitTheme picks the startup theme: the VMINFO_THEME env var wins over the
// persisted preference; when both are empty the current (default) theme stays.
func InitTheme(persisted string) {
	if v := os.Getenv("VMINFO_THEME"); v != "" {
		SetTheme(v)
		return
	}
	if persisted != "" {
		SetTheme(persisted)
	}
}

// ── Active Color Slots ───────────────────────────────────────────────
//
// The C* vars are the palette the views read at render time; applyTheme
// reassigns them, so every style picks up the new theme on the next frame.

var (
	// Base palette
	CText   lipgloss.AdaptiveColor // primary text
	CDim    lipgloss.AdaptiveColor // muted/dim
	CBorder lipgloss.AdaptiveColor // panel borders

	// Accent colors
	CCyan        lipgloss.AdaptiveColor
	CGreen       lipgloss.AdaptiveColor
	CYellow      lipgloss.AdaptiveColor
	CRed         lipgloss.AdaptiveColor
	CPurple      lipgloss.AdaptiveColor
	CBlue        lipgloss.AdaptiveColor
	CPink        lipgloss.AdaptiveColor // upload / Tx
	CBrightGreen lipgloss.AdaptiveColor // download / Rx

	// 4-tier threshold colors (bright, visually distinct)
	COk       lipgloss.AdaptiveColor // 0-50% green
	CWarn     lipgloss.AdaptiveColor // 50-75% yellow
	CAlert    lipgloss.AdaptiveColor // 75-90% orange
	CCritical lipgloss.AdaptiveColor // 90-100% red

	// UI element colors
	CMuted     lipgloss.AdaptiveColor // gray for idle/empty bar parts
	CInfo      lipgloss.AdaptiveColor // blue for key hints
	CBadgeBg   lipgloss.AdaptiveColor // header badge background
	CSurfaceBg lipgloss.AdaptiveColor // process column header background
	COddBg     lipgloss.AdaptiveColor // zebra row odd background
	CEvenBg    lipgloss.AdaptiveColor // zebra row even background
	CSelectBg  lipgloss.AdaptiveColor // selected row background
)

func init() {
	applyTheme(activeTheme)
}

// applyTheme makes t active: it refreshes the color slots and rebuilds the
// package-level styles that capture colors at construction time.
func applyTheme(t Theme) {
	activeTheme = t

	CText = t.Text
	CDim = t.Dim
	CBorder = t.Border
	CMuted = t.Muted
	CInfo = t.Info

	CCyan = t.Cyan
	CGreen = t.Green
	CYellow = t.Yellow
	CRed = t.Red
	CPurple = t.Purple
	CBlue = t.Blue
	CPink = t.Pink
	CBrightGreen = t.BrightGreen

	COk = t.Ok
	CWarn = t.Warn
	CAlert = t.Alert
	CCritical = t.Critical

	CBadgeBg = t.BadgeBg
	CSurfaceBg = t.SurfaceBg
	COddBg = t.OddBg
	CEvenBg = t.EvenBg
	CSelectBg = t.SelectBg

	rebuildStyles()
}

// ── Threshold Helpers ────────────────────────────────────────────────

// ThresholdColor returns a color based on 4-tier thresholds:
// 0-50% OK, 50-75% Warn, 75-90% Alert, 90-100% Critical.
func ThresholdColor(pct float64) lipgloss.TerminalColor {
	switch {
	case pct >= 90:
		return CCritical
	case pct >= 75:
		return CAlert
	case pct >= 50:
		return CWarn
	default:
		return COk
	}
}

// ── Panel Icons ──────────────────────────────────────────────────────

// PanelIcons maps panel names to their Unicode icon prefix.
var PanelIcons = map[string]string{
	"System":         "◈",
	"CPU":            "⚡",
	"Resources":      "◉",
	"Disk I/O":       "◆",
	"Network & Load": "◎",
	"Processes":      "☰",
}

// ── Panel Width Calculations ─────────────────────────────────────────

const panelGap = 1 // spacer width between panels in a row

// calcRow1Widths returns (sysW, diskW) for the two-column Row 1 layout
// based on 40%/60% split of available width.
func calcRow1Widths(totalW int) (sysW, diskW int) {
	available := max(totalW-4-panelGap, 40)
	sysW = max(available*40/100, 20)
	diskW = max(available-sysW, 20)
	return
}

// calcFullWidth returns the width for a full-width panel (accounting for outer padding).
func calcFullWidth(totalW int) int {
	return max(totalW-4, 30)
}

// sysInnerWidth computes the usable inner width of the System panel.
func sysInnerWidth(totalW int) int {
	sysW, _ := calcRow1Widths(totalW)
	return max(sysW-4, 16)
}

// resInnerWidth computes the usable inner width of the full-width Resources panel.
func resInnerWidth(totalW int) int {
	return max(calcFullWidth(totalW)-4, 30)
}
