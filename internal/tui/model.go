package tui

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/VPSMarket/vminfo/pkg/vminfo"
)

const (
	refreshInterval   = 3 * time.Second
	sampleInterval    = 200 * time.Millisecond
	stateStaleAfter   = 2*refreshInterval + time.Second
	maxVisiblePerCore = 8
)

type viewMode string
type processSortKey string

const (
	viewOverview  viewMode = "overview"
	viewProcesses viewMode = "processes"

	sortCPU  processSortKey = "cpu"
	sortMem  processSortKey = "mem"
	sortPID  processSortKey = "pid"
	sortName processSortKey = "name"
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
	ctx         context.Context
	static      vminfo.StaticInfo
	stats       vminfo.RuntimeStats
	hasStats    bool
	statsErr    error
	lastUpdated time.Time
	paused      bool
	showHelp    bool
	view        viewMode
	processSort processSortKey
	processes   []vminfo.ProcessInfo
	processErr  error
	selected    int
	killConfirm bool
	killTarget  vminfo.ProcessInfo
	statusText  string
	width       int
	height      int
}

func Run(ctx context.Context, stdout io.Writer) error {
	staticInfo, err := vminfo.CollectStatic(ctx)
	if err != nil {
		return err
	}
	program := tea.NewProgram(
		newModel(ctx, staticInfo),
		tea.WithOutput(stdout),
		tea.WithInput(os.Stdin),
	)
	_, err = program.Run()
	return err
}

func newModel(ctx context.Context, staticInfo vminfo.StaticInfo) Model {
	return Model{
		ctx:         ctx,
		static:      staticInfo,
		view:        viewOverview,
		processSort: sortCPU,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchStatsCmd(m.ctx), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.paused {
			return m, tickCmd()
		}
		cmds := []tea.Cmd{fetchStatsCmd(m.ctx), tickCmd()}
		if m.view == viewProcesses {
			cmds = append(cmds, fetchProcessesCmd(m.ctx))
		}
		return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case statsMsg:
		m.statsErr = msg.err
		if msg.err == nil {
			m.stats = msg.stats
			m.hasStats = true
			m.lastUpdated = time.Now()
		}
		return m, nil
	case processMsg:
		m.processErr = msg.err
		if msg.err == nil {
			m.processes = msg.items
			m.selected = clampIndex(m.selected, len(m.processes))
		}
		return m, nil
	case killMsg:
		if msg.err != nil {
			m.statusText = "kill failed: " + msg.err.Error()
			return m, nil
		}
		m.statusText = "sent SIGTERM to PID " + strings.TrimSpace(renderPID(msg.pid))
		return m, tea.Batch(fetchProcessesCmd(m.ctx), fetchStatsCmd(m.ctx))
	case tea.KeyMsg:
		if m.killConfirm {
			switch strings.ToLower(strings.TrimSpace(msg.String())) {
			case "enter", "y":
				target := m.killTarget
				m.killConfirm = false
				m.statusText = "sending SIGTERM..."
				return m, terminateProcessCmd(m.ctx, target)
			case "esc", "n", "q":
				m.killConfirm = false
				m.statusText = "kill canceled"
				return m, nil
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
		case "p":
			m.paused = !m.paused
			if m.paused {
				m.statusText = "paused"
				return m, nil
			}
			m.statusText = "resumed"
			return m, fetchStatsCmd(m.ctx)
		case "r":
			m.statusText = "refreshing..."
			if m.view == viewProcesses {
				return m, tea.Batch(fetchStatsCmd(m.ctx), fetchProcessesCmd(m.ctx))
			}
			return m, fetchStatsCmd(m.ctx)
		case "tab":
			if m.view == viewOverview {
				m.view = viewProcesses
				m.statusText = "processes view"
				return m, tea.Batch(fetchStatsCmd(m.ctx), fetchProcessesCmd(m.ctx))
			}
			m.view = viewOverview
			m.statusText = "overview view"
			return m, nil
		case "up":
			if m.view == viewProcesses && m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down":
			if m.view == viewProcesses && m.selected < len(m.processes)-1 {
				m.selected++
			}
			return m, nil
		case "s":
			if m.view == viewProcesses {
				m.cycleProcessSort()
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
			m.statusText = "confirm kill?"
			return m, nil
		}
	}
	return m, nil
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
		m.statusText = "sort: mem"
	case sortMem:
		m.processSort = sortPID
		m.statusText = "sort: pid"
	case sortPID:
		m.processSort = sortName
		m.statusText = "sort: name"
	default:
		m.processSort = sortCPU
		m.statusText = "sort: cpu"
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

func (m Model) processWindow(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	visible := m.processVisibleCount()
	if total <= visible {
		return 0, total
	}
	selected := clampIndex(m.selected, total)
	start := selected - visible/2
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > total {
		end = total
		start = end - visible
	}
	if start < 0 {
		start = 0
	}
	return start, end
}

func (m Model) processVisibleCount() int {
	if m.height <= 0 {
		return 12
	}
	visible := m.height - 10
	if visible < 6 {
		return 6
	}
	if visible > 20 {
		return 20
	}
	return visible
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

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
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
