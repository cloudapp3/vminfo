package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cloudapp3/vminfo/internal/collector"
)

func TestHandleProcessesRejectsNonGET(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/processes", nil)
	rr := httptest.NewRecorder()

	srv.handleProcesses(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestParseProcessQuery(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    processQueryOptions
		wantErr bool
	}{
		{
			name:   "defaults",
			values: url.Values{},
			want: processQueryOptions{
				sortKey: "cpu",
			},
		},
		{
			name: "filter sort and limit",
			values: url.Values{
				"filter": []string{"postgres"},
				"sort":   []string{"mem"},
				"limit":  []string{"10"},
			},
			want: processQueryOptions{
				filter:  "postgres",
				sortKey: "mem",
				limit:   10,
			},
		},
		{
			name: "q alias",
			values: url.Values{
				"q": []string{"ssh"},
			},
			want: processQueryOptions{
				filter:  "ssh",
				sortKey: "cpu",
			},
		},
		{
			name: "negative limit",
			values: url.Values{
				"limit": []string{"-1"},
			},
			wantErr: true,
		},
		{
			name: "non numeric limit",
			values: url.Values{
				"limit": []string{"many"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProcessQuery(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseProcessQuery returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("options = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestApplyProcessQueryFiltersSortsAndLimits(t *testing.T) {
	info := collector.ProcessInfo{
		Total: 4,
		List: []collector.ProcessEntry{
			{PID: 101, PPID: 1, Name: "nginx", Command: "nginx: worker", User: "www-data", CPUPercent: 12.5, MemPercent: 1.2, Status: "S"},
			{PID: 202, PPID: 1, Name: "postgres", Command: "postgres: writer", User: "postgres", CPUPercent: 3.1, MemPercent: 9.8, Status: "S"},
			{PID: 303, PPID: 202, Name: "postgres", Command: "postgres: checkpointer", User: "postgres", CPUPercent: 7.4, MemPercent: 5.6, Status: "R"},
			{PID: 404, PPID: 1, Name: "sleep", Command: "sleep 10", User: "root", CPUPercent: 0.0, MemPercent: 0.1, Status: "S"},
		},
	}

	got := applyProcessQuery(info, processQueryOptions{
		filter:  "postgres",
		sortKey: "mem",
		limit:   1,
	})

	if got.Total != info.Total {
		t.Fatalf("total = %d, want %d", got.Total, info.Total)
	}
	if len(got.List) != 1 {
		t.Fatalf("expected 1 process, got %d", len(got.List))
	}
	if got.List[0].PID != 202 {
		t.Fatalf("PID = %d, want 202", got.List[0].PID)
	}
}

func TestProcessEntryMatchesPIDPPIDCommandUserAndStatus(t *testing.T) {
	item := collector.ProcessEntry{
		PID:     303,
		PPID:    202,
		Name:    "postgres",
		Command: "postgres: checkpointer",
		User:    "postgres",
		Status:  "D",
	}

	for _, filter := range []string{"303", "202", "checkpointer", "postgres", "d"} {
		t.Run(filter, func(t *testing.T) {
			if !processEntryMatches(item, filter) {
				t.Fatalf("expected %q to match %+v", filter, item)
			}
		})
	}
}

func TestHandleNetDiagRejectsNonPOST(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/net/diag", nil)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleNetDiagRequiresTarget(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", strings.NewReader(`{"action":"dns"}`))
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing target, got %d", rr.Code)
	}
}

func TestHandleNetDiagUnknownAction(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", strings.NewReader(`{"action":"frob","target":"x"}`))
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 unknown action, got %d", rr.Code)
	}
}

func TestHandleNetDiagDNS(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", strings.NewReader(`{"action":"dns","target":"localhost"}`))
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleNetDiagPing(t *testing.T) {
	srv := &Server{}
	body := strings.NewReader(`{"action":"ping","target":"127.0.0.1","mode":"tcp","port":1,"count":1,"timeout_ms":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", body)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}
