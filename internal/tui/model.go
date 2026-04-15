package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VPSMarket/vminfo"
	"github.com/VPSMarket/vminfo/internal/i18n"
)

const (
	defaultRefreshInterval = 3 * time.Second
	sampleInterval         = 200 * time.Millisecond
	maxCPUHistory          = 60
)

type viewMode string
type processSortKey string
type layoutMode int

const (
	viewOverview  viewMode = "overview"
	viewProcesses viewMode = "processes"

	sortCPU  processSortKey = "cpu"
	sortMem  processSortKey = "mem"
	sortPID  processSortKey = "pid"
	sortName processSortKey = "name"

	layoutCompact layoutMode = iota // < 80
	layoutNarrow                    // 80-99
	layoutMedium                    // 100-139
	layoutWide                      // >= 140
)

type tickMsg struct{}

type statsMsg struct {
	stats vminfo.RuntimeStats
	err   error
}

type processMsg struct {
	items []vminfo.ProcessInfo
	err   error
}

type killMsg struct {
	pid  int32
	name string
	err  error
}

type Model struct {
	ctx             context.Context
	static          vminfo.StaticInfo
	stats           vminfo.RuntimeStats
	hasStats        bool
	statsErr        error
	lastUpdated     time.Time
	paused          bool
	showHelp        bool
	view            viewMode
	processSort     processSortKey
	processes       []vminfo.ProcessInfo
	processErr      error
	selected        int
	killConfirm     bool
	killTarget      vminfo.ProcessInfo
	statusText      string
	width           int
	height          int
	cpuHistory      []float64
	treeView        bool
	filterActive    bool
	refreshInterval time.Duration
	tr              *i18n.Translator

	// Bubbles components
	spinner     spinner.Model
	filterInput textinput.Model
	viewport    viewport.Model
	ready       bool // viewport initialized
}

func Run(ctx context.Context, stdout io.Writer, tr *i18n.Translator) error {
	staticInfo, err := vminfo.CollectStatic(ctx)
	if err != nil {
		return err
	}
	program := tea.NewProgram(
		newModel(ctx, staticInfo, tr),
		tea.WithOutput(stdout),
		tea.WithInput(os.Stdin),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err = program.Run()
	return err
}

func newModel(ctx context.Context, staticInfo vminfo.StaticInfo, tr *i18n.Translator) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = subtleStyle

	ti := textinput.New()
	ti.CharLimit = 64
	ti.Placeholder = "filter process name..."

	return Model{
		ctx:             ctx,
		static:          staticInfo,
		view:            viewOverview,
		processSort:     sortCPU,
		refreshInterval: defaultRefreshInterval,
		tr:              tr,
		spinner:         s,
		filterInput:     ti,
	}
}

func (m Model) layoutMode() layoutMode {
	switch {
	case m.width >= 140:
		return layoutWide
	case m.width >= 100:
		return layoutMedium
	case m.width >= 80:
		return layoutNarrow
	default:
		return layoutCompact
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchStatsCmd(m.ctx),
		tickCmd(m.refreshInterval),
		m.spinner.Tick,
		setWindowTitleCmd(m),
	)
}

// setWindowTitleCmd sets the terminal window title.
func setWindowTitleCmd(m Model) tea.Cmd {
	state := "ONLINE"
	if m.paused {
		state = "PAUSED"
	}
	return tea.SetWindowTitle(fmt.Sprintf("vminfo - %s [%s]", m.static.Hostname, state))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.paused {
			return m, tickCmd(m.refreshInterval)
		}
		cmds := []tea.Cmd{fetchStatsCmd(m.ctx), tickCmd(m.refreshInterval)}
		if m.view == viewProcesses {
			cmds = append(cmds, fetchProcessesCmd(m.ctx))
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := max(m.height-12, 6)
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, vpHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = vpHeight
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statsMsg:
		m.statsErr = msg.err
		if msg.err == nil {
			m.stats = msg.stats
			m.hasStats = true
			m.lastUpdated = time.Now()
			m.cpuHistory = append(m.cpuHistory, msg.stats.CPU)
			if len(m.cpuHistory) > maxCPUHistory {
				m.cpuHistory = m.cpuHistory[len(m.cpuHistory)-maxCPUHistory:]
			}
		}
		return m, setWindowTitleCmd(m)

	case processMsg:
		m.processErr = msg.err
		if msg.err == nil {
			m.processes = msg.items
			m.selected = clampIndex(m.selected, len(m.processes))
		}
		return m, nil

	case killMsg:
		if msg.err != nil {
			m.statusText = m.tr.Tf("kill failed: %s", msg.err.Error())
			return m, nil
		}
		m.statusText = m.tr.Tf("sent SIGTERM to PID %s", strings.TrimSpace(renderPID(msg.pid)))
		return m, tea.Batch(fetchProcessesCmd(m.ctx), fetchStatsCmd(m.ctx))

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.view != viewProcesses || m.killConfirm || m.showHelp {
		return m, nil
	}
	items := m.filteredProcesses()
	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.selected > 0 {
				m.selected--
				m.syncViewport()
			}
		case tea.MouseButtonWheelDown:
			if m.selected < len(items)-1 {
				m.selected++
				m.syncViewport()
			}
		case tea.MouseButtonLeft:
			// Header lines above the process rows:
			// outer padding(1) + panel border-top(1) + title(1) + blank(1) +
			// optional filter(0/1) + column header(1) = ~5-6
			headerLines := 5
			if m.filterInput.Focused() || m.filterActive {
				headerLines++
			}
			row := msg.Y - headerLines
			if row >= 0 && row+m.viewport.YOffset < len(items) {
				m.selected = row + m.viewport.YOffset
				m.syncViewport()
			}
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.killConfirm {
		switch strings.ToLower(strings.TrimSpace(msg.String())) {
		case "enter", "y":
			target := m.killTarget
			m.killConfirm = false
			m.statusText = m.tr.T("sending SIGTERM...")
			return m, terminateProcessCmd(m.ctx, target)
		case "esc", "n", "q":
			m.killConfirm = false
			m.statusText = m.tr.T("kill canceled")
			return m, nil
		}
		return m, nil
	}

	if m.filterInput.Focused() {
		switch msg.String() {
		case "enter":
			m.filterInput.Blur()
			m.filterActive = m.filterInput.Value() != ""
			if m.filterActive {
				m.statusText = m.tr.Tf("filter: %s", m.filterInput.Value())
			} else {
				m.statusText = m.tr.T("filter cleared")
			}
			m.selected = 0
			return m, nil
		case "esc":
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.filterActive = false
			m.statusText = m.tr.T("filter canceled")
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			return m, cmd
		}
	}

	if m.showHelp {
		switch strings.ToLower(strings.TrimSpace(msg.String())) {
		case "?", "esc", "q", "enter":
			m.showHelp = false
		}
		return m, nil
	}

	switch strings.ToLower(strings.TrimSpace(msg.String())) {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	case "/":
		if m.view == viewProcesses {
			m.filterInput.Focus()
			m.filterInput.SetValue("")
			m.statusText = m.tr.T("filter: (type to search, enter to apply)")
		}
		return m, nil
	case "+", "=":
		if m.refreshInterval > time.Second {
			m.refreshInterval -= time.Second
			m.statusText = m.tr.Tf("refresh %s", m.refreshInterval)
		}
		return m, nil
	case "-":
		if m.refreshInterval < 10*time.Second {
			m.refreshInterval += time.Second
			m.statusText = fmt.Sprintf("refresh %s", m.refreshInterval)
		}
		return m, nil
	case "p":
		m.paused = !m.paused
		if m.paused {
			m.statusText = m.tr.T("paused")
			return m, setWindowTitleCmd(m)
		}
		m.statusText = m.tr.T("resumed")
		return m, tea.Batch(fetchStatsCmd(m.ctx), setWindowTitleCmd(m))
	case "r":
		m.statusText = m.tr.T("refreshing...")
		if m.view == viewProcesses {
			return m, tea.Batch(fetchStatsCmd(m.ctx), fetchProcessesCmd(m.ctx))
		}
		return m, fetchStatsCmd(m.ctx)
	case "tab":
		if m.view == viewOverview {
			m.view = viewProcesses
			m.statusText = m.tr.T("processes view")
			return m, tea.Batch(fetchStatsCmd(m.ctx), fetchProcessesCmd(m.ctx))
		}
		m.view = viewOverview
		m.statusText = m.tr.T("overview view")
		return m, nil
	case "up":
		if m.view == viewProcesses && m.selected > 0 {
			m.selected--
			m.syncViewport()
		}
		return m, nil
	case "down":
		if m.view == viewProcesses && m.selected < len(m.processes)-1 {
			m.selected++
			m.syncViewport()
		}
		return m, nil
	case "s":
		if m.view == viewProcesses {
			m.cycleProcessSort()
		}
		return m, nil
	case "t":
		if m.view == viewProcesses {
			m.treeView = !m.treeView
			if m.treeView {
				m.statusText = m.tr.T("tree view")
			} else {
				m.statusText = m.tr.T("flat view")
			}
		}
		return m, nil
	case "k":
		if m.view != viewProcesses {
			return m, nil
		}
		target, ok := m.selectedProcess()
		if !ok {
			return m, nil
		}
		m.killConfirm = true
		m.killTarget = target
		m.statusText = m.tr.T("confirm kill?")
		return m, nil
	}
	return m, nil
}

// syncViewport scrolls the viewport to keep the selected row visible.
func (m *Model) syncViewport() {
	if !m.ready {
		return
	}
	if m.selected < m.viewport.YOffset {
		m.viewport.SetYOffset(m.selected)
	} else if m.selected >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(m.selected - m.viewport.Height + 1)
	}
}

func (m Model) selectedProcess() (vminfo.ProcessInfo, bool) {
	items := m.sortedProcesses()
	if len(items) == 0 {
		return vminfo.ProcessInfo{}, false
	}
	index := clampIndex(m.selected, len(items))
	return items[index], true
}

func (m *Model) cycleProcessSort() {
	switch m.processSort {
	case sortCPU:
		m.processSort = sortMem
		m.statusText = m.tr.T("sort: mem")
	case sortMem:
		m.processSort = sortPID
		m.statusText = m.tr.T("sort: pid")
	case sortPID:
		m.processSort = sortName
		m.statusText = m.tr.T("sort: name")
	default:
		m.processSort = sortCPU
		m.statusText = m.tr.T("sort: cpu")
	}
}

func (m Model) sortedProcesses() []vminfo.ProcessInfo {
	items := append([]vminfo.ProcessInfo(nil), m.processes...)
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		switch m.processSort {
		case sortMem:
			if left.MemoryPercent != right.MemoryPercent {
				return left.MemoryPercent > right.MemoryPercent
			}
		case sortPID:
			if left.PID != right.PID {
				return left.PID < right.PID
			}
		case sortName:
			leftName := strings.ToLower(strings.TrimSpace(left.Name))
			rightName := strings.ToLower(strings.TrimSpace(right.Name))
			if leftName != rightName {
				return leftName < rightName
			}
		default:
			if left.CPUPercent != right.CPUPercent {
				return left.CPUPercent > right.CPUPercent
			}
		}
		return left.PID < right.PID
	})
	return items
}

func (m Model) filteredProcesses() []vminfo.ProcessInfo {
	items := m.sortedProcesses()
	filterText := m.filterInput.Value()
	if !m.filterActive && !m.filterInput.Focused() || filterText == "" {
		return items
	}
	query := strings.ToLower(filterText)
	filtered := make([]vminfo.ProcessInfo, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

type treeNode struct {
	proc     vminfo.ProcessInfo
	children []treeNode
	depth    int
}

func (m Model) buildProcessTree() []treeNode {
	pidSet := make(map[int32]bool, len(m.processes))
	childrenMap := make(map[int32][]int32)
	procMap := make(map[int32]vminfo.ProcessInfo)

	for _, p := range m.processes {
		pidSet[p.PID] = true
		procMap[p.PID] = p
		childrenMap[p.PPID] = append(childrenMap[p.PPID], p.PID)
	}

	var roots []int32
	for _, p := range m.processes {
		if p.PPID == 0 || !pidSet[p.PPID] {
			roots = append(roots, p.PID)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i] < roots[j] })

	result := make([]treeNode, 0, len(roots))
	for _, pid := range roots {
		result = append(result, buildSubTree(pid, procMap, childrenMap, 0))
	}
	return result
}

func buildSubTree(pid int32, procMap map[int32]vminfo.ProcessInfo, childrenMap map[int32][]int32, depth int) treeNode {
	node := treeNode{
		proc:  procMap[pid],
		depth: depth,
	}
	childPIDs := childrenMap[pid]
	sort.Slice(childPIDs, func(i, j int) bool { return childPIDs[i] < childPIDs[j] })
	for _, childPID := range childPIDs {
		node.children = append(node.children, buildSubTree(childPID, procMap, childrenMap, depth+1))
	}
	return node
}

func (m Model) flattenTree(nodes []treeNode) []treeNode {
	var result []treeNode
	for _, node := range nodes {
		result = append(result, node)
		result = append(result, m.flattenTree(node.children)...)
	}
	return result
}

func clampIndex(index, total int) int {
	if total <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= total {
		return total - 1
	}
	return index
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func fetchStatsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		stats, err := vminfo.CollectStats(ctx, vminfo.Options{SampleInterval: sampleInterval})
		return statsMsg{stats: stats, err: err}
	}
}

func fetchProcessesCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		items, err := vminfo.ListProcesses(ctx)
		return processMsg{items: items, err: err}
	}
}

func terminateProcessCmd(ctx context.Context, target vminfo.ProcessInfo) tea.Cmd {
	return func() tea.Msg {
		err := vminfo.TerminateProcess(ctx, target.PID)
		return killMsg{pid: target.PID, name: target.Name, err: err}
	}
}
