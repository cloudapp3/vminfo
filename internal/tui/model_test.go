package tui

import (
	"context"
	"testing"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestSelectedProcessUsesFilteredList(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	m.showKernel = true
	m.processes = []vminfo.ProcessInfo{
		{PID: 1, Name: "alpha"},
		{PID: 2, Name: "beta"},
	}
	m.filterActive = true
	m.filterInput.SetValue("bet")
	m.refreshProcessListState()

	got, ok := m.selectedProcess()
	if !ok {
		t.Fatal("expected a selected process")
	}
	if got.PID != 2 {
		t.Fatalf("expected filtered process PID 2, got %d", got.PID)
	}
}

func TestRefreshProcessListStateClampsSelectionToVisibleRows(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	m.showKernel = true
	m.processes = []vminfo.ProcessInfo{
		{PID: 1, Name: "alpha"},
		{PID: 2, Name: "beta"},
		{PID: 3, Name: "gamma"},
	}
	m.selected = 2
	m.filterActive = true
	m.filterInput.SetValue("bet")

	m.refreshProcessListState()

	if m.selected != 0 {
		t.Fatalf("expected selected row to clamp to 0, got %d", m.selected)
	}

	items := m.filteredProcesses()
	if len(items) != 1 || items[0].PID != 2 {
		t.Fatalf("expected one filtered process with PID 2, got %+v", items)
	}
}

func TestSelectedProcessUsesVisibleTreeOrder(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	m.treeView = true
	m.showKernel = true
	m.processes = []vminfo.ProcessInfo{
		{PID: 2, PPID: 1, Name: "child"},
		{PID: 3, PPID: 0, Name: "other"},
		{PID: 1, PPID: 0, Name: "parent"},
	}
	m.refreshProcessListState()

	first, ok := m.selectedProcess()
	if !ok {
		t.Fatal("expected a selected process")
	}
	if first.PID != 1 {
		t.Fatalf("expected first visible tree process PID 1, got %d", first.PID)
	}

	m.selected = 1
	second, ok := m.selectedProcess()
	if !ok {
		t.Fatal("expected a selected process")
	}
	if second.PID != 2 {
		t.Fatalf("expected second visible tree process PID 2, got %d", second.PID)
	}
}

func TestKernelThreadsHiddenByDefault(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	m.processes = []vminfo.ProcessInfo{
		{PID: 1, PPID: 0, Name: "init", RSSBytes: 4096},
		{PID: 2, PPID: 0, Name: "kthreadd"},
		{PID: 100, PPID: 2, Name: "kworker", RSSBytes: 0},
		{PID: 200, PPID: 1, Name: "sshd", RSSBytes: 8192},
	}
	m.refreshProcessListState()

	items := m.filteredProcesses()
	for _, p := range items {
		if p.PID == 2 || p.PID == 100 {
			t.Fatalf("kernel thread leaked into list: %+v", p)
		}
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 user-space processes, got %d (%+v)", len(items), items)
	}

	m.showKernel = true
	m.refreshProcessListState()
	items = m.filteredProcesses()
	if len(items) != 4 {
		t.Fatalf("expected all 4 processes when showKernel=true, got %d", len(items))
	}
}

func TestPIDTwoUserProcessIsNotHiddenAsKernelThread(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	m.processes = []vminfo.ProcessInfo{
		{PID: 1, PPID: 0, Name: "bwrap", RSSBytes: 152 * 1024},
		{PID: 2, PPID: 1, Name: "zsh", RSSBytes: 3792 * 1024},
		{PID: 100, PPID: 2, Name: "go", RSSBytes: 8192 * 1024},
	}
	m.refreshProcessListState()

	items := m.filteredProcesses()
	if len(items) != 3 {
		t.Fatalf("expected PID namespace user processes to stay visible, got %d (%+v)", len(items), items)
	}
	for _, p := range items {
		if p.PID == 2 {
			return
		}
	}
	t.Fatalf("expected normal PID 2 process to stay visible, got %+v", items)
}
