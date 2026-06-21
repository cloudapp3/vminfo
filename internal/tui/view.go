package tui

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/cloudapp3/vminfo"
)

// ── Styles & Constants ────────────────────────────────────────────────

var (
	outerStyle = lipgloss.NewStyle().Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			BorderForeground(CBorder)

	subtleStyle = lipgloss.NewStyle().Foreground(CDim)
	warnStyle   = lipgloss.NewStyle().Foreground(CYellow)
	errorStyle  = lipgloss.NewStyle().Foreground(CRed)
	valueStyle  = lipgloss.NewStyle().Foreground(CText).Bold(true)

	labelW = 8

	// Progress bar chars
	blockFull  = '█'
	blockEmpty = '·'

	// Sparkline — uses block elements that render well in all terminals
	sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
)

// ── Main View ────────────────────────────────────────────────────────

func (m Model) View() string {
	var content string
	if m.killConfirm {
		// kill confirm uses lipgloss.Place which already fills the screen
		return m.renderKillConfirm()
	} else if m.showHelp {
		content = m.renderHelp()
	} else {
		content = m.renderMain()
	}

	// Pad output to fill terminal height so previous frame content is cleared
	if m.height > 0 {
		lines := strings.Count(content, "\n") + 1
		if lines < m.height {
			content += strings.Repeat("\n", m.height-lines)
		}
	}
	return content
}

func (m Model) renderMain() string {
	// Header bar
	host := lipgloss.NewStyle().Bold(true).Foreground(CText).Render(
		" vminfo " + firstNonEmpty(m.static.Hostname, "-"),
	)
	stateBadge := m.renderBadge(m.stateLabel(), m.stateColor())
	pageBadge := m.renderBadge(m.pageLabel(), CBlue)
	header := lipgloss.NewStyle().Foreground(CBorder).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		PaddingBottom(0).
		Render(host + "  " + stateBadge + "  " + pageBadge)

	var body string
	if m.view == viewOverview {
		body = m.renderOverview()
	} else {
		body = m.renderProcesses()
	}

	status := subtleStyle.Render(m.statusLine())
	separator := lipgloss.NewStyle().Foreground(CBorder).Render(
		strings.Repeat("─", max(m.width-4, 10)))
	footer := subtleStyle.Render(m.hintsForMode())

	return outerStyle.Render(strings.Join([]string{header, body, "", status, separator, footer}, "\n"))
}

// ── Overview ─────────────────────────────────────────────────────────

func (m Model) renderOverview() string {
	totalW := m.width
	if totalW <= 0 {
		totalW = 120
	}

	gap := lipgloss.NewStyle().Width(panelGap).Render(" ")
	makePanel := func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			BorderForeground(CBorder).
			Width(w)
	}

	fullW := calcFullWidth(totalW)

	switch m.layoutMode() {
	case layoutWide:
		// >= 140: 2-col Row 1 (System|DiskIO) + full Resources + full Network
		sysW, diskW := calcRow1Widths(totalW)
		sysContent, diskContent := m.renderSystemContent(), m.renderDiskIOContent()
		sysContent, diskContent = equalizeContent2(sysContent, diskContent)
		sysCard := makePanel(sysW).Render(sysContent)
		diskCard := makePanel(diskW).Render(diskContent)
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, sysCard, gap, diskCard)
		resCard := makePanel(fullW).Render(m.renderResourceContent())
		netCard := makePanel(fullW).Render(m.renderNetworkContent())
		return lipgloss.JoinVertical(lipgloss.Left, topRow, resCard, netCard)

	case layoutMedium:
		// 100-139: 2-col Row 1 (System|DiskIO) + full Resources + full Network
		sysW, diskW := calcRow1Widths(totalW)
		sysContent, diskContent := m.renderSystemContent(), m.renderDiskIOContent()
		sysContent, diskContent = equalizeContent2(sysContent, diskContent)
		sysCard := makePanel(sysW).Render(sysContent)
		diskCard := makePanel(diskW).Render(diskContent)
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, sysCard, gap, diskCard)
		resCard := makePanel(fullW).Render(m.renderResourceContent())
		netCard := makePanel(fullW).Render(m.renderNetworkContent())
		return lipgloss.JoinVertical(lipgloss.Left, topRow, resCard, netCard)

	case layoutNarrow:
		// 80-99: single column, all panels full width
		sysCard := makePanel(fullW).Render(m.renderSystemContent())
		diskCard := makePanel(fullW).Render(m.renderDiskIOContent())
		resCard := makePanel(fullW).Render(m.renderResourceContent())
		netCard := makePanel(fullW).Render(m.renderNetworkContent())
		return lipgloss.JoinVertical(lipgloss.Left, sysCard, diskCard, resCard, netCard)

	default:
		// < 70: compact — tiny System summary line + Resources + Network
		sysLine := subtleStyle.Render(m.renderSystemOneLine(fullW))
		resCard := makePanel(fullW).Render(m.renderResourceContent())
		netCard := makePanel(fullW).Render(m.renderNetworkContent())
		compactHint := subtleStyle.Render(m.tr.T("Compact view. Resize to 70+ cols for System panel."))
		return lipgloss.JoinVertical(lipgloss.Left, sysLine, resCard, netCard, compactHint)
	}
}

// ── Panel content renderers (no border, border applied by caller) ────

// renderSystemOneLine produces a single-line system summary for tight layouts.
func (m Model) renderSystemOneLine(width int) string {
	parts := []string{}
	if h := strings.TrimSpace(m.static.Hostname); h != "" {
		parts = append(parts, valueStyle.Render(h))
	}
	if v := strings.TrimSpace(firstNonEmpty(m.static.Platform, m.static.OS, "")); v != "" {
		ver := strings.TrimSpace(m.static.OSVersion)
		if ver != "" {
			v += " " + ver
		}
		parts = append(parts, subtleStyle.Render(v))
	}
	if m.hasStats {
		parts = append(parts, subtleStyle.Render(m.tr.T("Uptime")+" "+formatUptime(m.stats.Uptime)))
		parts = append(parts, subtleStyle.Render(m.tr.T("Procs")+" "+fmt.Sprintf("%d", m.stats.ProcessCount)))
	}
	out := " " + strings.Join(parts, subtleStyle.Render(" │ "))
	if width > 0 {
		return truncate(out, width)
	}
	return out
}

func (m Model) renderSystemContent() string {
	sysW := sysInnerWidth(m.width)
	valW := max(sysW-labelW-2, 10)

	lines := []string{
		m.panelTitle("System"),
		"",
		m.kv("OS", truncate(firstNonEmpty(m.static.Platform, m.static.OS, "-")+" "+strings.TrimSpace(m.static.OSVersion), valW)),
		m.kv("Kernel", truncate(firstNonEmpty(m.static.Kernel, "-"), valW)),
		m.kv("Arch", firstNonEmpty(m.static.Arch, "-")),
		m.kv("Host", firstNonEmpty(m.static.Hostname, "-")),
		m.kv("CPU", truncate(fmt.Sprintf("%s ("+m.tr.T("%d cores")+")", firstNonEmpty(m.static.CPUModel, "-"), m.static.CPUCores), valW)),
	}
	if v := firstNonEmpty(m.static.Virtualization, ""); v != "" && v != "-" {
		lines = append(lines, m.kv("Virt", v))
	}
	if m.hasStats {
		lines = append(lines,
			m.label("Uptime")+lipgloss.NewStyle().Foreground(CGreen).Render(formatUptime(m.stats.Uptime)),
			m.kv("Procs", fmt.Sprintf("%d", m.stats.ProcessCount)),
		)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderCPUContent() string {
	lines := []string{
		m.panelTitle("CPU"),
		"",
	}

	if len(m.cpuHistory) > 1 {
		sparkW := max(m.width/3, 30)
		spark := renderSparkline(m.cpuHistory, sparkW)
		lines = append(lines, spark)

		cur := m.cpuHistory[len(m.cpuHistory)-1]
		statsLine := subtleStyle.Render("  cur ") + colorizePercent(cur) +
			subtleStyle.Render("  "+m.tr.T("avg")+" ") + colorizePercent(avgFloat64(m.cpuHistory)) +
			subtleStyle.Render("  "+m.tr.T("max")+" ") + colorizePercent(maxFloat64(m.cpuHistory))
		lines = append(lines, statsLine)
	} else {
		lines = append(lines, subtleStyle.Render(m.tr.T("Collecting...")))
	}

	if m.hasStats {
		var extras []string
		if len(m.stats.Temps) > 0 {
			t := m.stats.Temps[0]
			tc := colorForTempEnhanced(t.Temperature)
			extras = append(extras, lipgloss.NewStyle().Foreground(tc).Bold(true).Render(
				fmt.Sprintf("%.0f°C", t.Temperature)))
		}
		if len(extras) > 0 {
			lines = append(lines, subtleStyle.Render("  ")+strings.Join(extras, subtleStyle.Render("  ")))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderResourceContent() string {
	title := m.panelTitle("Resources")

	if !m.hasStats {
		return title + "\n\n" + m.spinner.View() + " " + subtleStyle.Render(m.tr.T("Loading..."))
	}
	if m.statsErr != nil {
		return title + "\n\n" + errorStyle.Render(m.tr.Tf("Error: %s", m.statsErr.Error()))
	}

	innerW := resInnerWidth(m.width)
	// Split: left ~45% for bars, right ~55% for sparkline/cores
	leftW := max(innerW*45/100, 30)
	rightW := max(innerW-leftW-1, 20) // 1 for separator

	// ─── Left: progress bars ───
	cpuPct := m.stats.CPU
	memPct := safePercent(m.stats.MemUsed, m.static.MemTotal)
	swapPct := safePercent(m.stats.SwapUsed, m.static.SwapTotal)
	diskPct := safePercent(m.stats.DiskUsed, m.static.DiskTotal)

	leftLines := []string{
		m.renderResourceBar("CPU", cpuPct, 0, 0),
		m.renderResourceBar("Mem", memPct, m.stats.MemUsed, m.static.MemTotal),
		m.renderResourceBar("Swap", swapPct, m.stats.SwapUsed, m.static.SwapTotal),
		m.renderResourceBar("Disk", diskPct, m.stats.DiskUsed, m.static.DiskTotal),
	}

	// ─── Right: sparkline + stats + cores ───
	rightLines := []string{}
	if len(m.cpuHistory) > 1 {
		spark := renderSparkline(m.cpuHistory, rightW)
		rightLines = append(rightLines, spark)

		cur := m.cpuHistory[len(m.cpuHistory)-1]
		statsLine := subtleStyle.Render(m.tr.T("cur")+" ") + colorizePercent(cur) +
			subtleStyle.Render("  "+m.tr.T("avg")+" ") + colorizePercent(avgFloat64(m.cpuHistory)) +
			subtleStyle.Render("  "+m.tr.T("max")+" ") + colorizePercent(maxFloat64(m.cpuHistory))
		var extras []string
		if len(m.stats.Temps) > 0 {
			t := m.stats.Temps[0]
			tc := colorForTempEnhanced(t.Temperature)
			extras = append(extras, lipgloss.NewStyle().Foreground(tc).Bold(true).Render(fmt.Sprintf("%.0f\u00b0C", t.Temperature)))
		}
		if len(extras) > 0 {
			statsLine += subtleStyle.Render("  ") + strings.Join(extras, subtleStyle.Render("  "))
		}
		rightLines = append(rightLines, statsLine)
	} else {
		rightLines = append(rightLines, subtleStyle.Render(m.tr.T("Collecting...")))
	}

	// Cores bar
	if len(m.stats.CPUPerCore) > 0 {
		limit := min(len(m.stats.CPUPerCore), 16)
		isCompact := len(m.stats.CPUPerCore) > 8
		chars := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			chars = append(chars, miniBar(m.stats.CPUPerCore[i]))
		}
		sep := " "
		if isCompact {
			sep = ""
		}
		coreLine := strings.Join(chars, sep)
		if len(m.stats.CPUPerCore) > 16 {
			coreLine += subtleStyle.Render(fmt.Sprintf("+%d", len(m.stats.CPUPerCore)-16))
		}
		// Average
		var sum float64
		for _, v := range m.stats.CPUPerCore[:limit] {
			sum += v
		}
		coreAvg := sum / float64(limit)
		coreLine += subtleStyle.Render("  "+m.tr.T("avg")+" ") + lipgloss.NewStyle().Foreground(ThresholdColor(coreAvg)).Render(fmt.Sprintf("%.1f%%", coreAvg))
		rightLines = append(rightLines, "", subtleStyle.Render(m.tr.T("Cores")+" ")+coreLine)
		if !isCompact {
			nums := make([]string, limit)
			for i := range nums {
				nums[i] = fmt.Sprintf("%d", i)
			}
			coreNums := strings.Join(nums, sep)
			rightLines = append(rightLines, subtleStyle.Render("      ")+coreNums)
		}
	}

	// ─── Equalize line count ───
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// ─── Build body lines ───
	bodyLines := []string{""}
	for i := 0; i < maxLines; i++ {
		left := lipgloss.NewStyle().Width(leftW).Render(leftLines[i])
		sepChar := lipgloss.NewStyle().Foreground(CBorder).Render("\u2502")
		right := rightLines[i]
		bodyLines = append(bodyLines, left+" "+sepChar+" "+right)
	}

	return title + "\n" + strings.Join(bodyLines, "\n")
}

func (m Model) renderDiskIOContent() string {
	lines := []string{
		m.panelTitle("Disk I/O"),
		"",
	}

	if m.hasStats && len(m.stats.DiskIO) > 0 {
		// Sort: active devices first, then idle
		disks := make([]vminfo.DiskIOStats, len(m.stats.DiskIO))
		copy(disks, m.stats.DiskIO)
		sort.SliceStable(disks, func(i, j int) bool {
			iActive := disks[i].ReadSpeed > 0 || disks[i].WriteSpeed > 0 || disks[i].IOPS > 0
			jActive := disks[j].ReadSpeed > 0 || disks[j].WriteSpeed > 0 || disks[j].IOPS > 0
			return iActive && !jActive
		})
		for _, d := range disks {
			isIdle := d.ReadSpeed == 0 && d.WriteSpeed == 0 && d.IOPS == 0
			if isIdle {
				line := lipgloss.NewStyle().Foreground(CMuted).Render(fmt.Sprintf("%-8s ↓%6s/s  ↑%6s/s  iops %d",
					truncate(d.Name, 8), formatBytes(d.ReadSpeed), formatBytes(d.WriteSpeed), d.IOPS))
				lines = append(lines, line)
			} else {
				line := lipgloss.NewStyle().Foreground(CText).Bold(true).Render(
					fmt.Sprintf("%-8s", truncate(d.Name, 8))) +
					" ↓" + lipgloss.NewStyle().Foreground(CBrightGreen).Render(fmt.Sprintf("%6s/s", formatBytes(d.ReadSpeed))) +
					"  ↑" + lipgloss.NewStyle().Foreground(CPink).Render(fmt.Sprintf("%6s/s", formatBytes(d.WriteSpeed))) +
					subtleStyle.Render(fmt.Sprintf("  iops %d", d.IOPS))
				lines = append(lines, line)
			}
		}
	} else if m.hasStats {
		lines = append(lines, subtleStyle.Render(m.tr.T("No data")))
	} else {
		lines = append(lines, m.spinner.View()+" "+subtleStyle.Render(m.tr.T("Loading...")))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderNetworkContent() string {
	lines := []string{
		m.panelTitle("Network & Load"),
		"",
	}

	if !m.hasStats {
		lines = append(lines, m.spinner.View()+" "+subtleStyle.Render(m.tr.T("Loading...")))
		return strings.Join(lines, "\n")
	}

	summary := m.renderNetworkSummary()
	if summary != "" {
		lines = append(lines, summary)
	}

	traffic := m.renderNetworkTrafficSection()
	if traffic != "" {
		lines = append(lines, "", traffic)
	}

	ifaces := m.renderNetworkInterfaces()
	if ifaces != "" {
		lines = append(lines, "", ifaces)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderNetworkSummary() string {
	cores := m.static.CPUCores
	if cores == 0 {
		cores = 1
	}
	wide := m.width >= 120
	medium := m.width >= 80

	loadValue := func(load float64) string {
		return lipgloss.NewStyle().Foreground(loadColor(load, cores)).Render(fmt.Sprintf("%.2f", load))
	}
	loadBar := func(load float64) string {
		return lipgloss.NewStyle().Foreground(loadColor(load, cores)).Render(strings.Repeat(string(loadMiniBar(load, cores)), 3))
	}
	tcp := valueStyle.Render(fmt.Sprintf("%d", m.stats.TCPCount))
	udp := valueStyle.Render(fmt.Sprintf("%d", m.stats.UDPCount))

	switch {
	case wide:
		line1 := strings.Join([]string{
			"  " + subtleStyle.Render(m.tr.T("Load")),
			loadValue(m.stats.Load1),
			loadValue(m.stats.Load5),
			loadValue(m.stats.Load15),
			"    " + subtleStyle.Render("TCP"),
			tcp,
			" " + subtleStyle.Render("UDP"),
			udp,
		}, " ")
		line2 := "  " + subtleStyle.Render(strings.Repeat(" ", len(m.tr.T("Load"))+1)) + loadBar(m.stats.Load1) + "   " + loadBar(m.stats.Load5) + "   " + loadBar(m.stats.Load15)
		line3 := "  " + subtleStyle.Render(strings.Repeat(" ", len(m.tr.T("Load"))+1)) + subtleStyle.Render("1m") + "    " + subtleStyle.Render("5m") + "    " + subtleStyle.Render("15m")
		return strings.Join([]string{line1, line2, line3}, "\n")
	case medium:
		return strings.Join([]string{
			"  " + subtleStyle.Render(m.tr.T("Load")) + " " + loadValue(m.stats.Load1) + " " + loadValue(m.stats.Load5) + " " + loadValue(m.stats.Load15),
			"  " + subtleStyle.Render("TCP") + " " + tcp + "  " + subtleStyle.Render("UDP") + " " + udp,
		}, "\n")
	default:
		rx := lipgloss.NewStyle().Foreground(CBrightGreen).Render("↓ " + formatBytes(m.stats.NetInSpeed) + "/s")
		tx := lipgloss.NewStyle().Foreground(CPink).Render("↑ " + formatBytes(m.stats.NetOutSpeed) + "/s")
		return strings.Join([]string{
			"  " + subtleStyle.Render(m.tr.T("Load")) + " " + loadValue(m.stats.Load1) + " " + loadValue(m.stats.Load5) + " " + loadValue(m.stats.Load15) + "  " + rx + " " + tx,
			"  " + subtleStyle.Render("TCP") + " " + tcp + "  " + subtleStyle.Render("UDP") + " " + udp,
		}, "\n")
	}
}

func (m Model) renderNetworkTrafficSection() string {
	if m.width < 80 {
		return ""
	}

	rx := lipgloss.NewStyle().Foreground(CBrightGreen).Render("↓ " + formatBytes(m.stats.NetInSpeed) + "/s")
	tx := lipgloss.NewStyle().Foreground(CPink).Render("↑ " + formatBytes(m.stats.NetOutSpeed) + "/s")
	dashCount := max(calcFullWidth(m.width)-18, 12)

	lines := []string{
		"  " + subtleStyle.Render("Traffic "+strings.Repeat("─", dashCount)),
		"  " + subtleStyle.Render("Total") + "  " + rx + "  " + tx,
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderNetworkInterfaces() string {
	ifaces := m.stats.Interfaces
	if len(ifaces) == 0 {
		return ""
	}

	sorted := sortInterfaces(ifaces)
	showTotal := m.width >= 120
	compact := m.width < 80
	showHeader := !compact
	maxIdleVisible := 3
	if compact {
		maxIdleVisible = 1
	}
	if len(sorted) <= maxIdleVisible+2 {
		maxIdleVisible = len(sorted)
	}

	greenDot := lipgloss.NewStyle().Foreground(COk).Render("●")
	grayDot := lipgloss.NewStyle().Foreground(CMuted).Render("○")
	activeStyle := lipgloss.NewStyle().Foreground(CText)
	idleStyle := lipgloss.NewStyle().Foreground(CMuted)
	ifaceW := 12
	ipW := 16
	rxW := 13
	txW := 13
	totalW := 12
	if showTotal {
		ifaceW = 14
		ipW = 17
		rxW = 14
		txW = 14
	}
	if compact {
		ifaceW = 10
		ipW = 15
	}

	var lines []string
	if showHeader {
		headers := []string{
			"  ",
			padRight("IFACE", ifaceW+2, false),
			padRight("IP", ipW, false),
			padLeft("RX/s", rxW, false),
			padLeft("TX/s", txW, false),
		}
		if showTotal {
			headers = append(headers, padLeft("TOTAL RX", totalW, false), padLeft("TOTAL TX", totalW, false))
		}
		lines = append(lines, subtleStyle.Render(strings.Join(headers, "")))
	}

	idleTotal := 0
	foldedCount := 0
	var foldedRx, foldedTx uint64
	for _, iface := range sorted {
		isActive := iface.RxSpeed > 0 || iface.TxSpeed > 0
		if !isActive {
			idleTotal++
			if idleTotal > maxIdleVisible && len(sorted) > maxIdleVisible+1 {
				foldedCount++
				foldedRx += iface.RxBytes
				foldedTx += iface.TxBytes
				continue
			}
		}

		rowStyle := idleStyle
		dot := grayDot
		if isActive {
			rowStyle = activeStyle
			dot = greenDot
		}

		name := truncateIfaceName(iface.Name, ifaceW-2)
		ip := iface.IPv4
		if ip == "" {
			ip = "—"
		}
		ipText := padRight(ip, ipW, false)
		ipStyle := lipgloss.NewStyle().Foreground(CMuted)
		if ip != "—" && !isPrivateIP(ip) {
			ipStyle = lipgloss.NewStyle().Foreground(CInfo).Bold(true)
		}

		rxText := lipgloss.NewStyle().Foreground(CBrightGreen).Render("↓ " + padLeft(formatBytes(iface.RxSpeed)+"/s", rxW-2, false))
		txText := lipgloss.NewStyle().Foreground(CPink).Render("↑ " + padLeft(formatBytes(iface.TxSpeed)+"/s", txW-2, false))

		if compact {
			line := "  " + dot + " " + rowStyle.Render(padRight(name, ifaceW, false)) + ipStyle.Render(ipText) + " " + rxText + " " + txText
			lines = append(lines, line)
			continue
		}

		parts := []string{
			"  " + dot + " ",
			rowStyle.Render(padRight(name, ifaceW, false)),
			ipStyle.Render(ipText),
			rxText,
			" ",
			txText,
		}
		if showTotal {
			parts = append(parts,
				" "+rowStyle.Render(padLeft(formatBytes(iface.RxBytes), totalW, false)),
				" "+rowStyle.Render(padLeft(formatBytes(iface.TxBytes), totalW, false)),
			)
		}
		line := strings.Join(parts, "")
		lines = append(lines, line)
	}

	if foldedCount > 0 {
		label := fmt.Sprintf("  %s %d %s", grayDot, foldedCount, m.tr.T("idle interfaces"))
		if compact {
			lines = append(lines, idleStyle.Render(label))
		} else if showTotal {
			lines = append(lines, idleStyle.Render(label+padLeft("", max(0, ifaceW+ipW+rxW+txW-15), false)+padLeft(formatBytes(foldedRx), totalW+1, false)+padLeft(formatBytes(foldedTx), totalW+1, false)))
		} else {
			lines = append(lines, idleStyle.Render(label))
		}
	}

	return strings.Join(lines, "\n")
}

func sortInterfaces(ifaces []vminfo.InterfaceIO) []vminfo.InterfaceIO {
	sorted := make([]vminfo.InterfaceIO, len(ifaces))
	copy(sorted, ifaces)
	sort.SliceStable(sorted, func(i, j int) bool {
		aActive := sorted[i].RxSpeed > 0 || sorted[i].TxSpeed > 0
		bActive := sorted[j].RxSpeed > 0 || sorted[j].TxSpeed > 0
		if aActive != bActive {
			return aActive
		}
		aPri := ifacePriority(sorted[i].Name)
		bPri := ifacePriority(sorted[j].Name)
		if aPri != bPri {
			return aPri < bPri
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func loadColor(load float64, cores uint32) lipgloss.Color {
	if cores == 0 {
		cores = 1
	}
	ratio := load / float64(cores)
	switch {
	case ratio >= 1.0:
		return CCritical
	case ratio >= 0.8:
		return CAlert
	case ratio >= 0.5:
		return CWarn
	default:
		return COk
	}
}

func loadMiniBar(load float64, cores uint32) rune {
	if cores == 0 {
		cores = 1
	}
	ratio := load / float64(cores)
	switch {
	case ratio >= 1.0:
		return '█'
	case ratio >= 0.8:
		return '▆'
	case ratio >= 0.5:
		return '▄'
	default:
		return '▁'
	}
}

func padRight(value string, width int, styled bool) string {
	if width <= 0 {
		return ""
	}
	if styled {
		return lipgloss.NewStyle().Width(width).Render(value)
	}
	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return value + strings.Repeat(" ", width-len(runes))
}

func padLeft(value string, width int, styled bool) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[len(runes)-width:])
	}
	return strings.Repeat(" ", width-len(runes)) + value
}

// ── Progress Bar ─────────────────────────────────────────────────────

const maxBarWidth = 30

// renderResourceBar renders a single-line resource bar: label + bar + pct + abs
func (m Model) renderResourceBar(lbl string, pct float64, used, total uint64) string {
	barW := maxBarWidth
	color := ThresholdColor(pct)

	ratio := max(pct/100, 0)
	ratio = min(ratio, 1)
	filled := int(ratio * float64(barW))
	filled = min(filled, barW)
	empty := barW - filled

	barFilled := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat(string(blockFull), filled))
	barEmpty := lipgloss.NewStyle().Foreground(CMuted).Render(strings.Repeat(string(blockEmpty), empty))

	pctStr := lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%5.1f%%", pct))

	absStr := ""
	if total > 0 {
		absStr = lipgloss.NewStyle().Foreground(CDim).Render(fmt.Sprintf("  %s/%s", formatBytes(used), formatBytes(total)))
	}

	return lipgloss.NewStyle().Foreground(CDim).Width(6).Render(m.tr.T(lbl)) + barFilled + barEmpty + " " + pctStr + absStr
}

// Mini inline bar for per-core display
func miniBar(pct float64) string {
	idx := int((pct / 100.0) * float64(len(sparklineChars)-1))
	idx = max(idx, 0)
	idx = min(idx, len(sparklineChars)-1)
	return lipgloss.NewStyle().Foreground(colorForPercent(pct)).Render(string(sparklineChars[idx]))
}

// ── Sparkline ────────────────────────────────────────────────────────

func renderSparkline(data []float64, width int) string {
	if len(data) == 0 || width <= 0 {
		return ""
	}
	sampled := sampleData(data, width)
	var sb strings.Builder

	// Pad left with ▁ when insufficient data
	padCount := width - len(sampled)
	if padCount > 0 {
		padChar := lipgloss.NewStyle().Foreground(lipgloss.Color("#4a5568")).Render(string(sparklineChars[0]))
		for range padCount {
			sb.WriteString(padChar)
		}
	}

	// Render actual data points
	for _, value := range sampled {
		idx := int((value / 100.0) * float64(len(sparklineChars)-1))
		idx = max(idx, 0)
		idx = min(idx, len(sparklineChars)-1)
		sb.WriteString(lipgloss.NewStyle().Foreground(colorForPercent(value)).Render(string(sparklineChars[idx])))
	}
	return sb.String()
}

func sampleData(data []float64, targetLen int) []float64 {
	if len(data) <= targetLen {
		return data
	}
	return data[len(data)-targetLen:]
}

// ── Color Helpers ────────────────────────────────────────────────────

func colorForPercent(pct float64) lipgloss.Color {
	return ThresholdColor(pct)
}

func colorizePercent(pct float64) string {
	return lipgloss.NewStyle().Foreground(colorForPercent(pct)).Bold(true).Render(fmt.Sprintf("%.1f%%", pct))
}

// colorForTempEnhanced uses 4-tier thresholds for CPU temperature display.
func colorForTempEnhanced(temp float64) lipgloss.Color {
	switch {
	case temp >= 90:
		return CCritical
	case temp >= 80:
		return CAlert
	case temp >= 60:
		return CWarn
	default:
		return COk
	}
}

// equalizeContent pads multiple content strings to the same line count,
// so that makePanel().Render() produces panels of equal height.
func equalizeContent(contents ...string) []string {
	maxLines := 0
	split := make([][]string, len(contents))
	for i, c := range contents {
		lines := strings.Split(c, "\n")
		split[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	result := make([]string, len(contents))
	for i, lines := range split {
		for len(lines) < maxLines {
			lines = append(lines, "")
		}
		result[i] = strings.Join(lines, "\n")
	}
	return result
}

// equalizeContent2 pads two content strings to the same line count.
func equalizeContent2(a, b string) (string, string) {
	r := equalizeContent(a, b)
	return r[0], r[1]
}

// ── Layout Helpers ───────────────────────────────────────────────────

func (m Model) panelTitle(text string) string {
	icon := ""
	if v, ok := PanelIcons[text]; ok {
		icon = lipgloss.NewStyle().Foreground(CBlue).Render(v + " ")
	}
	return icon + lipgloss.NewStyle().Bold(true).Foreground(CCyan).Render(m.tr.T(text)) +
		subtleStyle.Render(" "+strings.Repeat("─", 20))
}

func (m Model) label(key string) string {
	return lipgloss.NewStyle().Foreground(CDim).Width(labelW).Render(m.tr.T(key))
}

func (m Model) kv(key, value string) string {
	return m.label(key) + valueStyle.Render(value)
}

// ── Processes View ───────────────────────────────────────────────────

// depthColor returns a dimmer text color based on tree depth.
func depthColor(depth int) lipgloss.Color {
	switch {
	case depth == 0:
		return CText
	case depth == 1:
		return lipgloss.Color("#a0a8c0")
	case depth == 2:
		return lipgloss.Color("#8088a0")
	default:
		return CDim
	}
}

func (m Model) renderProcessTree() string {
	if m.processErr != nil {
		return panelStyle.Render(errorStyle.Render(m.processErr.Error()))
	}
	roots := m.buildProcessTree()
	flatNodes := m.flattenTree(roots)
	if len(flatNodes) == 0 {
		return panelStyle.Render(m.spinner.View() + " " + subtleStyle.Render(m.tr.T("Loading...")))
	}

	w := m.width
	if w <= 0 {
		w = 120
	}
	panelW := max(w-4, 40)
	innerW := max(panelW-4, 20)

	hdr := m.panelTitle("Processes (tree)")
	headerLines := []string{hdr, ""}
	headerLines = append(headerLines, subtleStyle.Render(fmt.Sprintf("%-6s %5s %5s  %s", m.tr.T("PID"), m.tr.T("CPU%"), m.tr.T("MEM%"), m.tr.T("NAME"))))
	connectorColor := lipgloss.Color("#444444")
	selectedBg := lipgloss.Color("#2D4F67")
	selectedIndex := clampIndex(m.selected, len(flatNodes))
	rowIndex := 0
	rowLines := make([]string, 0, len(flatNodes))
	var renderNodes func(nodes []treeNode, prefix string)
	renderNodes = func(nodes []treeNode, prefix string) {
		for i, node := range nodes {
			connector := "├─ "
			if i == len(nodes)-1 {
				connector = "└─ "
			}
			cpuColor := colorForPercent(node.proc.CPUPercent)
			nameColor := depthColor(node.depth)
			line := lipgloss.NewStyle().Foreground(CText).Render(
				fmt.Sprintf("%-6d ", node.proc.PID)) +
				lipgloss.NewStyle().Foreground(cpuColor).Render(
					fmt.Sprintf("%5.1f ", node.proc.CPUPercent)) +
				lipgloss.NewStyle().Foreground(CText).Render(
					fmt.Sprintf("%5.1f  ", node.proc.MemoryPercent)) +
				subtleStyle.Render(prefix) +
				lipgloss.NewStyle().Foreground(connectorColor).Render(connector) +
				lipgloss.NewStyle().Foreground(nameColor).Render(firstNonEmpty(node.proc.Name, "-"))
			if rowIndex == selectedIndex {
				line = lipgloss.NewStyle().Background(selectedBg).Width(innerW).Render(line)
			} else {
				line = lipgloss.NewStyle().Width(innerW).Render(line)
			}
			rowIndex++
			rowLines = append(rowLines, line)
			childPrefix := prefix + "│ "
			if i == len(nodes)-1 {
				childPrefix = prefix + "  "
			}
			renderNodes(node.children, childPrefix)
		}
	}
	renderNodes(roots, "")

	infoLine := ""
	selected, ok := m.selectedProcess()
	if ok {
		stateColor := CMuted
		switch strings.ToLower(firstNonEmpty(selected.State, "")) {
		case "running", "run":
			stateColor = COk
		case "sleep", "sleeping":
			stateColor = CInfo
		case "stop", "stopped":
			stateColor = CWarn
		case "zombie":
			stateColor = CCritical
		}
		infoLine = subtleStyle.Render("  "+m.tr.T("PID:")) + lipgloss.NewStyle().Foreground(CInfo).Render(fmt.Sprintf(" %d", selected.PID)) +
			subtleStyle.Render("  "+m.tr.T("Name:")) + lipgloss.NewStyle().Foreground(CText).Bold(true).Render(" "+firstNonEmpty(selected.Name, "-")) +
			subtleStyle.Render("  "+m.tr.T("State:")) + lipgloss.NewStyle().Foreground(stateColor).Render(" "+firstNonEmpty(selected.State, "-")) +
			subtleStyle.Render("  "+m.tr.T("CPU:")) + lipgloss.NewStyle().Foreground(ThresholdColor(selected.CPUPercent)).Render(fmt.Sprintf(" %.1f%%", selected.CPUPercent)) +
			subtleStyle.Render("  "+m.tr.T("Mem:")) + lipgloss.NewStyle().Foreground(ThresholdColor(float64(selected.MemoryPercent))).Render(fmt.Sprintf(" %.1f%%", selected.MemoryPercent)) +
			subtleStyle.Render("  "+m.tr.T("RSS:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+formatBytes(selected.RSSBytes)) +
			subtleStyle.Render("  "+m.tr.T("Age:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+formatUptime(selected.Uptime)) +
			subtleStyle.Render("  "+m.tr.T("Cmd:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+truncate(firstNonEmpty(selected.Command, selected.Name, "-"), 80))
	}

	allLines := append(headerLines, rowLines...)
	m.viewport.SetContent(strings.Join(allLines, "\n"))

	parts := []string{m.viewport.View()}
	if infoLine != "" {
		parts = append(parts, "", infoLine)
	}

	fullPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(CBorder).
		Width(panelW)
	return fullPanel.Render(strings.Join(parts, "\n"))
}

func (m Model) renderProcesses() string {
	if m.treeView {
		return m.renderProcessTree()
	}
	if m.processErr != nil {
		return panelStyle.Render(errorStyle.Render(m.processErr.Error()))
	}
	items := m.filteredProcesses()
	if len(items) == 0 {
		if len(m.processes) > 0 {
			return panelStyle.Render(subtleStyle.Render(m.tr.T("No matching processes")))
		}
		return panelStyle.Render(m.spinner.View() + " " + subtleStyle.Render(m.tr.T("Loading...")))
	}

	// Full-width panel for processes view
	w := m.width
	if w <= 0 {
		w = 120
	}
	panelW := max(w-4, 40)
	innerW := max(panelW-4, 20)

	hdr := m.panelTitle("Processes")
	headerLines := []string{hdr, ""}
	if m.filterInput.Focused() {
		headerLines = append(headerLines, m.filterInput.View())
	} else if m.filterActive {
		headerLines = append(headerLines, subtleStyle.Render(m.tr.Tf("filter: %s  (%d matches)", m.filterInput.Value(), len(items))))
	}

	// Dynamic column widths — USER auto-fits content, NAME absorbs remaining.
	colPID := 8
	colCPU := 7
	colMEM := 6
	colRSS := 9
	colGap := 2
	colUser := 4 // minimum for "USER" header
	for _, item := range items {
		if u := firstNonEmpty(item.User, "-"); len(u) > colUser {
			colUser = len(u)
		}
	}
	if colUser > 16 {
		colUser = 16
	}
	fixedW := 4 + colPID + colCPU + colMEM + colRSS + colUser + colGap*6 // sel marker + gaps
	colName := max(innerW-fixedW, 12)

	// Column headers with sort indicator
	sortArrow := "▼"
	cpuHeader := m.tr.T("CPU%")
	memHeader := m.tr.T("MEM%")
	pidHeader := m.tr.T("PID")
	switch m.processSort {
	case sortCPU:
		cpuHeader += sortArrow
	case sortMem:
		memHeader += sortArrow
	case sortPID:
		pidHeader += sortArrow
	}
	hdrStyle := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(CDim).Background(lipgloss.Color("#2A2A3E"))
	headerLine := lipgloss.NewStyle().Width(colPID).Render(pidHeader) +
		strings.Repeat(" ", colGap) +
		lipgloss.NewStyle().Width(colCPU).Render(cpuHeader) +
		strings.Repeat(" ", colGap) +
		lipgloss.NewStyle().Width(colMEM).Render(memHeader) +
		strings.Repeat(" ", colGap) +
		lipgloss.NewStyle().Width(colRSS).Render(m.tr.T("RSS")) +
		strings.Repeat(" ", colGap) +
		lipgloss.NewStyle().Width(colUser).Render(m.tr.T("USER")) +
		strings.Repeat(" ", colGap) +
		lipgloss.NewStyle().Width(colName).Render(m.tr.T("NAME"))
	headerLines = append(headerLines, hdrStyle.Width(innerW).Render("  "+headerLine))

	selectedBg := lipgloss.Color("#2D4F67")
	oddBg := lipgloss.Color("#1A1A2E")
	evenBg := lipgloss.Color("#16213E")

	// Render ALL process rows (viewport handles scrolling)
	rowLines := make([]string, 0, len(items))
	for idx := range items {
		item := items[idx]
		isSelected := idx == clampIndex(m.selected, len(items))
		isIdle := item.CPUPercent == 0 && item.MemoryPercent == 0 && item.RSSBytes == 0

		// Marker
		marker := " "
		if isSelected {
			marker = lipgloss.NewStyle().Foreground(CCyan).Bold(true).Render("▶")
		}

		// Column rendering with per-column colors
		pidS := lipgloss.NewStyle().Foreground(CBlue).Width(colPID).Render(fmt.Sprintf("%d", item.PID))

		cpuColor := ThresholdColor(item.CPUPercent)
		cpuS := lipgloss.NewStyle().Foreground(cpuColor).Width(colCPU).Render(fmt.Sprintf("%.1f%%", item.CPUPercent))

		memColor := ThresholdColor(float64(item.MemoryPercent))
		memS := lipgloss.NewStyle().Foreground(memColor).Width(colMEM).Render(fmt.Sprintf("%.1f%%", item.MemoryPercent))

		rssS := lipgloss.NewStyle().Foreground(CDim).Width(colRSS).Render(formatBytes(item.RSSBytes))

		userS := lipgloss.NewStyle().Foreground(CDim).Width(colUser).Render(truncate(firstNonEmpty(item.User, "-"), colUser))

		nameS := lipgloss.NewStyle().Foreground(CText).Width(colName).Render(truncate(firstNonEmpty(item.Name, "-"), colName))

		gap := strings.Repeat(" ", colGap)
		row := " " + marker + " " + pidS + gap + cpuS + gap + memS + gap + rssS + gap + userS + gap + nameS

		// Apply row-level styling
		if isSelected {
			row = lipgloss.NewStyle().Background(selectedBg).Width(innerW).Render(row)
		} else if isIdle {
			row = lipgloss.NewStyle().Foreground(CMuted).Background(
				lipgloss.Color(map[int]lipgloss.Color{0: evenBg, 1: oddBg}[idx%2])).Width(innerW).Render(row)
		} else {
			bg := evenBg
			if idx%2 == 1 {
				bg = oddBg
			}
			row = lipgloss.NewStyle().Background(bg).Width(innerW).Render(row)
		}

		rowLines = append(rowLines, row)
	}

	// Selected process info — structured format (below viewport)
	infoLine := ""
	selected, ok := m.selectedProcess()
	if ok {
		stateColor := CMuted
		switch strings.ToLower(firstNonEmpty(selected.State, "")) {
		case "running", "run":
			stateColor = COk
		case "sleep", "sleeping":
			stateColor = CInfo
		case "stop", "stopped":
			stateColor = CWarn
		case "zombie":
			stateColor = CCritical
		}
		infoLine = subtleStyle.Render("  "+m.tr.T("PID:")) + lipgloss.NewStyle().Foreground(CInfo).Render(fmt.Sprintf(" %d", selected.PID)) +
			subtleStyle.Render("  "+m.tr.T("Name:")) + lipgloss.NewStyle().Foreground(CText).Bold(true).Render(" "+firstNonEmpty(selected.Name, "-")) +
			subtleStyle.Render("  "+m.tr.T("State:")) + lipgloss.NewStyle().Foreground(stateColor).Render(" "+firstNonEmpty(selected.State, "-")) +
			subtleStyle.Render("  "+m.tr.T("CPU:")) + lipgloss.NewStyle().Foreground(ThresholdColor(selected.CPUPercent)).Render(fmt.Sprintf(" %.1f%%", selected.CPUPercent)) +
			subtleStyle.Render("  "+m.tr.T("Mem:")) + lipgloss.NewStyle().Foreground(ThresholdColor(float64(selected.MemoryPercent))).Render(fmt.Sprintf(" %.1f%%", selected.MemoryPercent)) +
			subtleStyle.Render("  "+m.tr.T("RSS:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+formatBytes(selected.RSSBytes)) +
			subtleStyle.Render("  "+m.tr.T("Age:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+formatUptime(selected.Uptime)) +
			subtleStyle.Render("  "+m.tr.T("Threads:")) + lipgloss.NewStyle().Foreground(CDim).Render(fmt.Sprintf(" %d", selected.Threads)) +
			subtleStyle.Render("  "+m.tr.T("Nice:")) + lipgloss.NewStyle().Foreground(CDim).Render(fmt.Sprintf(" %d", selected.Nice)) +
			subtleStyle.Render("  "+m.tr.T("Cmd:")) + lipgloss.NewStyle().Foreground(CDim).Render(" "+truncate(firstNonEmpty(selected.Command, selected.Name, "-"), 100))
	}

	// Build viewport content: header + all rows
	allLines := append(headerLines, rowLines...)
	m.viewport.SetContent(strings.Join(allLines, "\n"))

	// Wrap with panel border and info line below
	viewportRender := m.viewport.View()
	parts := []string{viewportRender}
	if infoLine != "" {
		parts = append(parts, "", infoLine)
	}

	fullPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(CBorder).
		Width(panelW)
	return fullPanel.Render(strings.Join(parts, "\n"))
}

// ── Help View ────────────────────────────────────────────────────────

func (m Model) renderHelp() string {
	hdr := m.panelTitle("Help")
	keys := []struct{ k, d string }{
		{"q / ctrl+c", m.tr.T("quit")},
		{"?", m.tr.T("toggle help")},
		{"p", m.tr.T("pause/resume refresh")},
		{"+ / -", m.tr.T("adjust refresh interval")},
		{"r", m.tr.T("refresh now")},
		{"tab", m.tr.T("switch overview / processes")},
		{"up / down", m.tr.T("move process selection")},
		{"s", m.tr.T("cycle process sort")},
		{"t", m.tr.T("toggle tree view")},
		{"/", m.tr.T("filter processes by name")},
		{"k", m.tr.T("kill selected process")},
		{"K", m.tr.T("toggle kernel threads")},
		{"enter / y", m.tr.T("confirm kill")},
		{"esc / n", m.tr.T("cancel")},
	}
	lines := []string{hdr, ""}
	for _, k := range keys {
		lines = append(lines, subtleStyle.Render("  ")+
			lipgloss.NewStyle().Foreground(CPurple).Width(14).Render(k.k)+
			subtleStyle.Render(k.d))
	}
	return outerStyle.Render(panelStyle.Render(strings.Join(lines, "\n")))
}

// ── Kill Confirm ─────────────────────────────────────────────────────

func (m Model) renderKillConfirm() string {
	target, ok := m.selectedProcess()
	if !ok {
		target = m.killTarget
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(CRed).Render("⚠ " + m.tr.T("Confirm Kill"))
	pidLabel := subtleStyle.Render(m.tr.T("PID:"))
	pidVal := lipgloss.NewStyle().Foreground(CInfo).Bold(true).Render(fmt.Sprintf("%d", target.PID))
	nameLabel := subtleStyle.Render(m.tr.T("Name:"))
	nameVal := valueStyle.Render(firstNonEmpty(target.Name, "-"))
	userLabel := subtleStyle.Render(m.tr.T("User:"))
	userVal := lipgloss.NewStyle().Foreground(CDim).Render(firstNonEmpty(target.User, "-"))

	body := []string{
		title,
		"",
		pidLabel + " " + pidVal + "    " + nameLabel + " " + nameVal,
		userLabel + " " + userVal,
		"",
		warnStyle.Render(m.tr.T("Send SIGTERM to this process?")),
		"",
		lipgloss.NewStyle().Foreground(COk).Bold(true).Render("[ Y ]") + " " + subtleStyle.Render(m.tr.T("yes, kill")) +
			"    " + lipgloss.NewStyle().Foreground(CRed).Bold(true).Render("[ N ]") + " " + subtleStyle.Render(m.tr.T("cancel")),
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CRed).
		Padding(1, 3).
		Render(strings.Join(body, "\n"))

	if m.width <= 0 || m.height <= 0 {
		return outerStyle.Render(box)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// ── Status & Footer ─────────────────────────────────────────────────

func (m Model) statusLine() string {
	// State badge with color dot
	state := m.stateLabel()
	stateColor := m.stateColor()
	dots := map[string]string{"ONLINE": "🟢", "PAUSED": "🟡", "ERROR": "🔴", "STALE": "🟠", "LOADING": "🔵"}
	dot := dots[state]
	if dot == "" {
		dot = "⚪"
	}
	badge := lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render(dot + " " + state)

	page := "OVERVIEW"
	if m.view == viewProcesses {
		page = "PROCS"
	}
	pageStr := lipgloss.NewStyle().Foreground(CText).Bold(true).Render(page)

	interval := subtleStyle.Render(m.tr.T("Interval:") + " " + m.refreshInterval.String())

	// Sort info for processes view
	sortStr := ""
	if m.view == viewProcesses {
		sortStr = subtleStyle.Render(" │ "+m.tr.T("Sort:")) + " " + lipgloss.NewStyle().Foreground(CInfo).Render(string(m.processSort)+" ▼")
	}

	// Left part
	left := badge + subtleStyle.Render(" │ ") + pageStr + subtleStyle.Render(" │ ") + interval + sortStr

	// Right part — timestamp
	right := ""
	if !m.lastUpdated.IsZero() {
		right = lipgloss.NewStyle().Foreground(CDim).Render(m.tr.T("Updated:") + " " + m.lastUpdated.Format("15:04:05"))
	}

	// Status text
	if strings.TrimSpace(m.statusText) != "" {
		left += subtleStyle.Render(" │ ") + subtleStyle.Render(m.statusText)
	}

	// Join left and right
	w := max(m.width-4, 30)
	leftRendered := lipgloss.NewStyle().Width(w / 2).Render(left)
	rightRendered := lipgloss.NewStyle().Width(w / 2).Align(lipgloss.Right).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, rightRendered)
}

func (m Model) hintsForMode() string {
	keyStyle := lipgloss.NewStyle().Foreground(CInfo)
	dimStyle := lipgloss.NewStyle().Foreground(CMuted)

	hint := func(key, desc string) string {
		return keyStyle.Render("["+key+"]") + dimStyle.Render(desc+" ")
	}

	switch {
	case m.killConfirm:
		return hint("enter/y", m.tr.T("confirm")) + hint("esc/n", m.tr.T("cancel"))
	case m.filterInput.Focused():
		return dimStyle.Render(m.tr.T("type to filter")+" ") + hint("enter", m.tr.T("confirm")) + hint("esc", m.tr.T("cancel"))
	case m.view == viewProcesses:
		base := hint("tab", m.tr.T("view")) + hint("↑↓", m.tr.T("select")) + hint("s", m.tr.T("sort")) + hint("t", m.tr.T("tree")) + hint("/", m.tr.T("filter")) + hint("k", "kill") + hint("K", m.tr.T("kthreads")) + hint("p", m.tr.T("pause")) + hint("r", m.tr.T("refresh")) + hint("?", m.tr.T("help")) + hint("q", m.tr.T("exit"))
		return base
	default:
		return hint("tab", m.tr.T("view")) + hint("+/-", m.tr.T("interval")) + hint("p", m.tr.T("pause")) + hint("r", m.tr.T("refresh")) + hint("?", m.tr.T("help")) + hint("q", m.tr.T("exit"))
	}
}

func (m Model) pageLabel() string {
	if m.view == viewProcesses {
		return m.tr.T("PROCS")
	}
	return m.tr.T("MAIN")
}

func (m Model) stateLabel() string {
	switch {
	case m.paused:
		return m.tr.T("PAUSED")
	case !m.hasStats:
		return m.tr.T("LOADING")
	case m.statsErr != nil:
		return m.tr.T("ERROR")
	case !m.lastUpdated.IsZero() && time.Since(m.lastUpdated) > 2*m.refreshInterval+time.Second:
		return m.tr.T("STALE")
	default:
		return m.tr.T("ONLINE")
	}
}

func (m Model) stateColor() lipgloss.Color {
	switch m.stateLabel() {
	case "ONLINE":
		return CGreen
	case "PAUSED", "STALE":
		return CYellow
	case "ERROR":
		return CRed
	default:
		return CBlue
	}
}

func (m Model) renderBadge(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Background(lipgloss.Color("#1a1b26")).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// ── General Helpers ──────────────────────────────────────────────────

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func formatBytes(bytes uint64) string {
	units := []string{"B", "K", "M", "G", "T", "P"}
	value := float64(bytes)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%dB", bytes)
	}
	if value >= 10 {
		return fmt.Sprintf("%.0f%s", value, units[i])
	}
	return fmt.Sprintf("%.1f%s", value, units[i])
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

// ── Math Helpers ─────────────────────────────────────────────────────

func safePercent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return (float64(used) / float64(total)) * 100
}

func avgFloat64(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func maxFloat64(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// ── Network Helpers ──────────────────────────────────────────────────

func ifacePriority(name string) int {
	switch {
	case strings.HasPrefix(name, "eth"), strings.HasPrefix(name, "en"):
		return 0
	case strings.HasPrefix(name, "wl"):
		return 1
	case strings.HasPrefix(name, "br"):
		return 2
	case strings.HasPrefix(name, "docker"):
		return 3
	case strings.HasPrefix(name, "veth"):
		return 4
	default:
		return 5
	}
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	privateNets := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"}
	for _, cidr := range privateNets {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func truncateIfaceName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	if maxLen >= 10 {
		return name[:maxLen-5] + "\u2026" + name[len(name)-4:]
	}
	return name[:maxLen-1] + "\u2026"
}
