package tui

import "github.com/charmbracelet/lipgloss"

// ── Color Constants ──────────────────────────────────────────────────

var (
	// Base palette
	CText   = lipgloss.Color("#c0caf5") // primary text
	CDim    = lipgloss.Color("#565f89") // muted/dim
	CBorder = lipgloss.Color("#3b4261") // panel borders

	// Accent colors
	CCyan   = lipgloss.Color("#7dcfff")
	CGreen  = lipgloss.Color("#9ece6a")
	CYellow = lipgloss.Color("#e0af68")
	CRed    = lipgloss.Color("#f7768e")
	CPurple = lipgloss.Color("#bb9af7")
	CBlue   = lipgloss.Color("#7aa2f7")
	COrange = lipgloss.Color("#ff9e64")
	CTeal   = lipgloss.Color("#73daca")

	// New accent colors
	CPink        = lipgloss.Color("#ff79c6") // upload / Tx
	CBrightGreen = lipgloss.Color("#00ff87") // download / Rx

	// 4-tier threshold colors (bright, visually distinct)
	COk       = lipgloss.Color("#00ff87") // 0-50% green
	CWarn     = lipgloss.Color("#ffd700") // 50-75% yellow
	CAlert    = lipgloss.Color("#ffaf5f") // 75-90% orange
	CCritical = lipgloss.Color("#ff5555") // 90-100% red

	// UI element colors
	CMuted = lipgloss.Color("#6c6c6c") // gray for idle/empty bar parts
	CInfo  = lipgloss.Color("#5fafff") // blue for key hints
)

// ── Threshold Helpers ────────────────────────────────────────────────

// ThresholdColor returns a color based on 4-tier thresholds:
// 0-50% OK, 50-75% Warn, 75-90% Alert, 90-100% Critical.
func ThresholdColor(pct float64) lipgloss.Color {
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
