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

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

const (
	defaultRefreshInterval = 3 * time.Second
	sampleInterval         = 200 * time.Millisecond
	firstSampleInterval    = 50 * time.Millisecond
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
	showKernel      bool
	refreshInterval time.Duration
	tr              *i18n.Translator

	// Bubbles components
	spinner     spinner.Model
	filterInput textinput.Model
	viewport    viewport.Model
	ready       bool // viewport initialized

	// Render cache
	sortedCache   []vminfo.ProcessInfo
	filteredCache []vminfo.ProcessInfo
	cacheDirty    bool
}

type RunOptions struct {
	Stdout io.Writer
	Stdin  io.Reader
	TR     *i18n.Translator
}

func Run(ctx context.Context, stdout io.Writer, tr *i18n.Translator) error {
	return RunWithOptions(ctx, RunOptions{
		Stdout: stdout,
		Stdin:  os.Stdin,
		TR:     tr,
	})
}

func RunWithOptions(ctx context.Context, opts RunOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.TR == nil {
		opts.TR = i18n.New(i18n.Detect())
	}

	staticInfo, err := vminfo.CollectStatic(ctx)
	if err != nil {
		return err
	}
	program := tea.NewProgram(
		newModel(ctx, staticInfo, opts.TR),
		tea.WithOutput(opts.Stdout),
		tea.WithInput(opts.Stdin),
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
	case m.width >= 70:
		return layoutNarrow
	default:
		return layoutCompact
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchStatsCmdInterval(m.ctx, firstSampleInterval),
		fetchProcessesCmd(m.ctx),
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
			m.refreshProcessListState()
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
	items := m.selectableProcesses()
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
			headerLines := m.processHeaderLines()
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
			m.refreshProcessListState()
			return m, nil
		case "esc":
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.filterActive = false
			m.statusText = m.tr.T("filter canceled")
			m.refreshProcessListState()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.refreshProcessListState()
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

	rawKey := strings.TrimSpace(msg.String())
	if rawKey == "K" && m.view == viewProcesses {
		m.showKernel = !m.showKernel
		if m.showKernel {
			m.statusText = m.tr.T("kernel threads: shown")
		} else {
			m.statusText = m.tr.T("kernel threads: hidden")
		}
		m.selected = 0
		m.refreshProcessListState()
		return m, nil
	}

	switch strings.ToLower(rawKey) {
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
			m.refreshProcessListState()
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
		if m.view == viewProcesses && m.selected < len(m.selectableProcesses())-1 {
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
			m.refreshProcessListState()
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
	items := m.selectableProcesses()
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
	m.refreshProcessListState()
}

func (m *Model) refreshProcessListState() {
	m.rebuildProcessCache()
	m.selected = clampIndex(m.selected, len(m.selectableProcesses()))
	m.syncViewport()
}

func (m Model) processHeaderLines() int {
	headerLines := 5
	if !m.treeView && (m.filterInput.Focused() || m.filterActive) {
		headerLines++
	}
	return headerLines
}

func (m Model) selectableProcesses() []vminfo.ProcessInfo {
	if m.treeView {
		nodes := m.flattenTree(m.buildProcessTree())
		items := make([]vminfo.ProcessInfo, 0, len(nodes))
		for _, node := range nodes {
			items = append(items, node.proc)
		}
		return items
	}
	return m.filteredProcesses()
}

// rebuildProcessCache recomputes the sorted and filtered process caches.
// Must be called from pointer-receiver contexts (Update handlers).
func (m *Model) rebuildProcessCache() {
	m.cacheDirty = false
	kernelRootPID := kernelThreadRootPID(m.processes)
	items := make([]vminfo.ProcessInfo, 0, len(m.processes))
	for _, p := range m.processes {
		if !m.showKernel && isKernelThread(p, kernelRootPID) {
			continue
		}
		items = append(items, p)
	}
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
	m.sortedCache = items

	filterText := m.filterInput.Value()
	if m.filterActive || (m.filterInput.Focused() && filterText != "") {
		filtered := make([]vminfo.ProcessInfo, 0, len(items))
		for _, item := range items {
			if processMatchesFilter(item, filterText) {
				filtered = append(filtered, item)
			}
		}
		m.filteredCache = filtered
	} else {
		m.filteredCache = items
	}
}

func (m Model) sortedProcesses() []vminfo.ProcessInfo {
	if !m.cacheDirty && m.sortedCache != nil {
		return m.sortedCache
	}
	kernelRootPID := kernelThreadRootPID(m.processes)
	items := make([]vminfo.ProcessInfo, 0, len(m.processes))
	for _, p := range m.processes {
		if !m.showKernel && isKernelThread(p, kernelRootPID) {
			continue
		}
		items = append(items, p)
	}
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
	if !m.cacheDirty && m.filteredCache != nil {
		return m.filteredCache
	}
	items := m.sortedProcesses()
	filterText := m.filterInput.Value()
	if !m.filterActive && !m.filterInput.Focused() || filterText == "" {
		return items
	}
	filtered := make([]vminfo.ProcessInfo, 0, len(items))
	for _, item := range items {
		if processMatchesFilter(item, filterText) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func processMatchesFilter(item vminfo.ProcessInfo, filter string) bool {
	query := strings.ToLower(strings.TrimSpace(filter))
	if query == "" {
		return true
	}
	fields := []string{
		renderPID(item.PID),
		renderPID(item.PPID),
		item.Name,
		item.Command,
		item.User,
		item.State,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
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
	kernelRootPID := kernelThreadRootPID(m.processes)

	for _, p := range m.processes {
		if !m.showKernel && isKernelThread(p, kernelRootPID) {
			continue
		}
		pidSet[p.PID] = true
		procMap[p.PID] = p
		childrenMap[p.PPID] = append(childrenMap[p.PPID], p.PID)
	}

	var roots []int32
	for _, p := range m.processes {
		if !m.showKernel && isKernelThread(p, kernelRootPID) {
			continue
		}
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

// kernelThreadRootPID returns the Linux kthreadd PID for a host process list.
// PID 2 is only treated as a kernel root when its name is actually kthreadd;
// inside PID namespaces PID 2 can be a normal user-space process.
func kernelThreadRootPID(items []vminfo.ProcessInfo) int32 {
	for _, p := range items {
		if p.PID == 2 && strings.TrimSpace(p.Name) == "kthreadd" {
			return p.PID
		}
	}
	return 0
}

// isKernelThread reports whether p is a kthreadd-owned kernel thread.
// The caller must pass a verified kthreadd PID to avoid hiding normal
// PID-namespace processes that happen to use PID/PPID 2.
func isKernelThread(p vminfo.ProcessInfo, kernelRootPID int32) bool {
	if kernelRootPID <= 0 {
		return false
	}
	if p.PID == kernelRootPID {
		return true
	}
	return p.PPID == kernelRootPID && p.RSSBytes == 0
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func fetchStatsCmd(ctx context.Context) tea.Cmd {
	return fetchStatsCmdInterval(ctx, sampleInterval)
}

func fetchStatsCmdInterval(ctx context.Context, interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		stats, err := vminfo.CollectStats(ctx, vminfo.Options{SampleInterval: interval})
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
