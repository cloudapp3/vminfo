package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestFilterProcessesMatchesCommandUserAndPID(t *testing.T) {
	items := []vminfo.ProcessInfo{
		{PID: 101, Name: "nginx", Command: "nginx: worker process", User: "www-data"},
		{PID: 202, Name: "postgres", Command: "postgres: checkpointer", User: "postgres"},
		{PID: 303, Name: "sleep", Command: "sleep 10", User: "root"},
	}

	tests := []struct {
		name   string
		filter string
		want   []int32
	}{
		{name: "command", filter: "worker", want: []int32{101}},
		{name: "user", filter: "postgres", want: []int32{202}},
		{name: "pid", filter: "303", want: []int32{303}},
		{name: "empty", filter: "", want: []int32{101, 202, 303}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProcesses(items, tt.filter)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d matches, got %d: %+v", len(tt.want), len(got), got)
			}
			for i, pid := range tt.want {
				if got[i].PID != pid {
					t.Fatalf("match %d PID = %d, want %d", i, got[i].PID, pid)
				}
			}
		})
	}
}

func TestProcessTreeRowsIncludesMatchedAncestors(t *testing.T) {
	items := []vminfo.ProcessInfo{
		{PID: 1, PPID: 0, Name: "init"},
		{PID: 10, PPID: 1, Name: "parent"},
		{PID: 11, PPID: 10, Name: "target", Command: "target --serve"},
		{PID: 20, PPID: 1, Name: "other"},
	}

	rows := processTreeRows(items, "target", 0)
	got := make([]int32, 0, len(rows))
	depths := make([]int, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.item.PID)
		depths = append(depths, row.depth)
	}
	want := []int32{1, 10, 11}
	wantDepths := []int{0, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("expected rows %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] || depths[i] != wantDepths[i] {
			t.Fatalf("row %d = pid %d depth %d, want pid %d depth %d", i, got[i], depths[i], want[i], wantDepths[i])
		}
	}
}

func TestProcessTreeRowsIncludesMatchedDescendants(t *testing.T) {
	// Only the master matches "nginx"; workers must appear via subtree
	// expansion, the unrelated branch must stay hidden, and the root ancestor
	// is still kept.
	items := []vminfo.ProcessInfo{
		{PID: 1, PPID: 0, Name: "init"},
		{PID: 10, PPID: 1, Name: "nginx"},
		{PID: 11, PPID: 10, Name: "worker"},
		{PID: 12, PPID: 10, Name: "worker"},
		{PID: 20, PPID: 1, Name: "other"},
	}

	rows := processTreeRows(items, "nginx", 0)
	got := make([]int32, 0, len(rows))
	depths := make([]int, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.item.PID)
		depths = append(depths, row.depth)
	}
	want := []int32{1, 10, 11, 12}
	wantDepths := []int{0, 1, 2, 2}
	if len(got) != len(want) {
		t.Fatalf("expected rows %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] || depths[i] != wantDepths[i] {
			t.Fatalf("row %d = pid %d depth %d, want pid %d depth %d", i, got[i], depths[i], want[i], wantDepths[i])
		}
	}
}

func TestProcessTreeRowsToleratesCyclicPPIDs(t *testing.T) {
	// Two processes whose PPIDs point at each other form a cycle with no
	// root, so the ancestor/descendant walks must terminate instead of
	// looping forever. Cyclic input is impossible from real /proc; this is a
	// purely defensive guard, so we only assert it returns promptly.
	items := []vminfo.ProcessInfo{
		{PID: 5, PPID: 6, Name: "alpha"},
		{PID: 6, PPID: 5, Name: "beta"},
	}

	done := make(chan struct{})
	go func() {
		_ = processTreeRows(items, "alpha", 0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("processTreeRows hung on cyclic PPIDs")
	}
}

func TestWriteProcessesShowsCommandAndAge(t *testing.T) {
	items := []vminfo.ProcessInfo{{
		PID:     42,
		PPID:    1,
		Name:    "sleep",
		Command: "sleep 10",
		User:    "root",
		State:   "S",
		Uptime:  65,
	}}

	var out bytes.Buffer
	if err := writeProcesses(&out, items, i18n.New("en")); err != nil {
		t.Fatalf("writeProcesses returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "COMMAND") || !strings.Contains(text, "AGE") {
		t.Fatalf("expected COMMAND and AGE headers, got:\n%s", text)
	}
	if !strings.Contains(text, "sleep 10") || !strings.Contains(text, "1m") {
		t.Fatalf("expected command and age in output, got:\n%s", text)
	}
}
