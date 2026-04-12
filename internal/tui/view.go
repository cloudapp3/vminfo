package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorTitle = lipgloss.Color("12")
	colorMuted = lipgloss.Color("8")
	colorInfo  = lipgloss.Color("14")
	colorWarn  = lipgloss.Color("11")
	colorError = lipgloss.Color("9")
	colorOK    = lipgloss.Color("10")

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	infoStyle   = lipgloss.NewStyle().Foreground(colorInfo)
	warnStyle   = lipgloss.NewStyle().Foreground(colorWarn)
	errorStyle  = lipgloss.NewStyle().Foreground(colorError)
	okStyle     = lipgloss.NewStyle().Foreground(colorOK)
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	headerStyle = lipgloss.NewStyle().PaddingBottom(1)
)

func (m Model) View() string {
	if m.killConfirm {
		return m.renderKillConfirm()
	}
	if m.showHelp {
		return m.renderHelp()
	}
	return m.renderMain()
}

func (m Model) renderMain() string {
	parts := []string{
		headerStyle.Render(
			titleStyle.Render("vminfo "+firstNonEmpty(m.static.Hostname, "-")) +
				"  " + m.renderBadge(m.stateLabel(), m.stateColor()) +
				"  " + m.renderBadge(m.pageLabel(), colorInfo),
		),
	}

	if m.view == viewOverview {
		parts = append(parts, m.renderOverview())
	} else {
		parts = append(parts, m.renderProcesses())
	}

	parts = append(parts, "", mutedStyle.Render(m.statusLine()), mutedStyle.Render(m.footerHints()))
	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(parts, "\n"))
}

func (m Model) renderOverview() string {
	staticLines := []string{
		fmt.Sprintf("OS           %s", strings.TrimSpace(strings.Join([]string{firstNonEmpty(m.static.Platform, m.static.OS, "-"), strings.TrimSpace(m.static.OSVersion)}, " "))),
		fmt.Sprintf("Kernel       %s", firstNonEmpty(m.static.Kernel, "-")),
		fmt.Sprintf("Arch         %s", firstNonEmpty(m.static.Arch, "-")),
		fmt.Sprintf("CPU          %s (%d cores)", firstNonEmpty(m.static.CPUModel, "-"), m.static.CPUCores),
		fmt.Sprintf("Memory       %s", formatBytes(m.static.MemTotal)),
		fmt.Sprintf("Swap         %s", formatBytes(m.static.SwapTotal)),
		fmt.Sprintf("Disk         %s", formatBytes(m.static.DiskTotal)),
		fmt.Sprintf("Virt         %s", firstNonEmpty(m.static.Virtualization, "-")),
	}
	left := panelStyle.Render(strings.Join(staticLines, "\n"))

	dynamicLines := []string{"Loading runtime stats..."}
	if m.hasStats {
		dynamicLines = []string{
			fmt.Sprintf("CPU          %s", formatPercent(m.stats.CPU)),
			fmt.Sprintf("Load         %.2f %.2f %.2f", m.stats.Load1, m.stats.Load5, m.stats.Load15),
			fmt.Sprintf("Memory       %s / %s", formatBytes(m.stats.MemUsed), formatBytes(m.static.MemTotal)),
			fmt.Sprintf("Swap         %s / %s", formatBytes(m.stats.SwapUsed), formatBytes(m.static.SwapTotal)),
			fmt.Sprintf("Disk         %s / %s", formatBytes(m.stats.DiskUsed), formatBytes(m.static.DiskTotal)),
			fmt.Sprintf("Network      ↓ %s/s ↑ %s/s", formatBytes(m.stats.NetInSpeed), formatBytes(m.stats.NetOutSpeed)),
			fmt.Sprintf("Conn         tcp %d / udp %d / proc %d", m.stats.TCPCount, m.stats.UDPCount, m.stats.ProcessCount),
			fmt.Sprintf("Uptime       %s", formatUptime(m.stats.Uptime)),
		}
		if len(m.stats.CPUPerCore) > 0 {
			coreText := make([]string, 0, minInt(len(m.stats.CPUPerCore), maxVisiblePerCore))
			for i, value := range m.stats.CPUPerCore {
				if i >= maxVisiblePerCore {
					break
				}
				coreText = append(coreText, fmt.Sprintf("c%d=%s", i, formatPercent(value)))
			}
			dynamicLines = append(dynamicLines, "Cores        "+strings.Join(coreText, "  "))
		}
	}
	if m.statsErr != nil {
		dynamicLines = append(dynamicLines, errorStyle.Render("Error        "+m.statsErr.Error()))
	}
	right := panelStyle.Render(strings.Join(dynamicLines, "\n"))

	if m.width > 120 {
		return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	}
	return lipgloss.JoinVertical(lipgloss.Left, left, "", right)
}

func (m Model) renderProcesses() string {
	if m.processErr != nil {
		return panelStyle.Render(errorStyle.Render(m.processErr.Error()))
	}
	items := m.sortedProcesses()
	if len(items) == 0 {
		return panelStyle.Render("Loading process list...")
	}

	start, end := m.processWindow(len(items))
	lines := []string{
		"SEL  PID    CPU%   MEM%   RSS        USER           NAME",
	}
	for idx := start; idx < end; idx++ {
		item := items[idx]
		marker := " "
		if idx == clampIndex(m.selected, len(items)) {
			marker = ">"
		}
		lines = append(lines, fmt.Sprintf(
			"%s    %-6d %-6.1f %-6.1f %-10s %-14s %s",
			marker,
			item.PID,
			item.CPUPercent,
			item.MemoryPercent,
			formatBytes(item.RSSBytes),
			truncate(firstNonEmpty(item.User, "-"), 14),
			truncate(firstNonEmpty(item.Name, "-"), 36),
		))
	}

	selected, ok := m.selectedProcess()
	if ok {
		lines = append(lines, "", mutedStyle.Render(
			fmt.Sprintf("selected pid=%d name=%s state=%s", selected.PID, firstNonEmpty(selected.Name, "-"), firstNonEmpty(selected.State, "-")),
		))
	}
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderHelp() string {
	lines := []string{
		titleStyle.Render("vminfo help"),
		"",
		"q / ctrl+c     quit",
		"?              toggle help",
		"p              pause/resume refresh",
		"r              refresh now",
		"tab            switch overview / processes",
		"up / down      move process selection",
		"s              cycle process sort in processes view",
		"k              kill selected process (with confirm)",
		"enter / y      confirm kill",
		"esc / n        cancel kill/help",
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(panelStyle.Render(strings.Join(lines, "\n")))
}

func (m Model) renderKillConfirm() string {
	target, ok := m.selectedProcess()
	if !ok {
		target = m.killTarget
	}
	lines := []string{
		titleStyle.Render("confirm kill"),
		"",
		fmt.Sprintf("Send SIGTERM to PID %d (%s)?", target.PID, firstNonEmpty(target.Name, "-")),
		"",
		warnStyle.Render("Enter / y to confirm"),
		mutedStyle.Render("Esc / n to cancel"),
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(panelStyle.Render(strings.Join(lines, "\n")))
}

func (m Model) statusLine() string {
	parts := []string{m.stateLabel(), m.pageLabel()}
	if m.view == viewProcesses {
		parts = append(parts, "sort="+string(m.processSort))
	}
	if !m.lastUpdated.IsZero() {
		parts = append(parts, "updated="+m.lastUpdated.Format("15:04:05"))
	}
	if strings.TrimSpace(m.statusText) != "" {
		parts = append(parts, m.statusText)
	}
	parts = append(parts, "refresh="+refreshInterval.String())
	return strings.Join(parts, " • ")
}

func (m Model) footerHints() string {
	if m.view == viewProcesses {
		return "tab overview • ↑↓ select • s sort • k kill • r refresh • p pause • ? help • q quit"
	}
	return "tab processes • r refresh • p pause • ? help • q quit"
}

func (m Model) pageLabel() string {
	if m.view == viewProcesses {
		return "PROCESSES"
	}
	return "OVERVIEW"
}

func (m Model) stateLabel() string {
	switch {
	case m.paused:
		return "PAUSED"
	case !m.hasStats:
		return "LOADING"
	case m.statsErr != nil:
		return "ERROR"
	case !m.lastUpdated.IsZero() && time.Since(m.lastUpdated) > stateStaleAfter:
		return "STALE"
	default:
		return "LIVE"
	}
}

func (m Model) stateColor() lipgloss.Color {
	switch m.stateLabel() {
	case "LIVE":
		return colorOK
	case "PAUSED", "STALE":
		return colorWarn
	case "ERROR":
		return colorError
	default:
		return colorInfo
	}
}

func (m Model) renderBadge(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Render(label)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func formatPercent(value float64) string {
	if value < 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", value)
}

func formatBytes(bytes uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	value := float64(bytes)
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	if index == 0 {
		return fmt.Sprintf("%d %s", bytes, units[index])
	}
	return fmt.Sprintf("%.1f %s", value, units[index])
}

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		return "0s"
	}
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func truncate(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return string(runes[:1])
	}
	return string(runes[:limit-1]) + "…"
}

func renderPID(pid int32) string {
	return fmt.Sprintf("%d", pid)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
