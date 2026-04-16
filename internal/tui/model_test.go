package tui

import (
	"context"
	"testing"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestSelectedProcessUsesFilteredList(t *testing.T) {
	m := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
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
