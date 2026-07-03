package collector

import (
	"testing"

	"github.com/cloudapp3/vminfo"
)

func TestBuildHealthReportsResourcePressure(t *testing.T) {
	staticInfo := vminfo.StaticInfo{
		CPUCores:  2,
		MemTotal:  100,
		SwapTotal: 100,
		DiskTotal: 100,
	}
	stats := vminfo.RuntimeStats{
		CPU:      91,
		MemUsed:  90,
		SwapUsed: 60,
		DiskUsed: 96,
		Load1:    3.5,
		Interfaces: []vminfo.InterfaceIO{{
			Name:     "eth0",
			RxErrors: 1,
		}},
	}
	procs := []ProcessEntry{{PID: 42, Name: "busy", CPUPercent: 95}}

	health := buildHealth(staticInfo, stats, procs)
	if health.Score >= 100 {
		t.Fatalf("expected pressure to reduce health score, got %+v", health)
	}
	codes := map[string]bool{}
	for _, warning := range health.Warnings {
		codes[warning.Code] = true
	}
	for _, code := range []string{"cpu_high", "load_high", "memory_high", "swap_high", "disk_high", "process_cpu_high"} {
		if !codes[code] {
			t.Fatalf("expected warning code %s in %+v", code, health.Warnings)
		}
	}
	if codes["network_errors"] {
		t.Fatalf("did not expect cumulative network errors to affect health: %+v", health.Warnings)
	}
}

func TestBuildHealthFromSnapshotUsesHydratedProcesses(t *testing.T) {
	snap := Snapshot{
		System: SystemInfo{Cores: 2},
		CPU:    CPUInfo{TotalPercent: 10},
		Memory: MemoryInfo{Total: 100, Used: 10, SwapTotal: 100, SwapUsed: 0},
		Disk:   DiskInfo{Filesystems: []Filesystem{{Total: 100, Used: 10}}},
		Load:   LoadInfo{Load1: 0.1},
		Processes: ProcessInfo{List: []ProcessEntry{{
			PID:        99,
			Name:       "hot",
			CPUPercent: 95,
		}}},
	}

	health := buildHealthFromSnapshot(snap)
	if len(health.Warnings) != 1 || health.Warnings[0].Code != "process_cpu_high" {
		t.Fatalf("expected process CPU warning, got %+v", health)
	}
}

func warningLevels(h HealthInfo) map[string]string {
	levels := make(map[string]string, len(h.Warnings))
	for _, w := range h.Warnings {
		levels[w.Code] = w.Level
	}
	return levels
}

func TestBuildHealthNetworkWarnings(t *testing.T) {
	static := vminfo.StaticInfo{CPUCores: 2}

	tests := []struct {
		name      string
		stats     vminfo.RuntimeStats
		wantCode  string
		wantLevel string
	}{
		{"error rate warning", vminfo.RuntimeStats{Interfaces: []vminfo.InterfaceIO{{Name: "eth0", RxErrRate: 12}}}, "network_errors", "warning"},
		{"error rate critical", vminfo.RuntimeStats{Interfaces: []vminfo.InterfaceIO{{Name: "eth0", RxErrRate: 60, TxErrRate: 60}}}, "network_errors", "critical"},
		{"drop rate warning", vminfo.RuntimeStats{Interfaces: []vminfo.InterfaceIO{{Name: "eth0", TxDropRate: 15}}}, "network_drops", "warning"},
		{"drop rate critical", vminfo.RuntimeStats{Interfaces: []vminfo.InterfaceIO{{Name: "eth0", RxDropRate: 150}}}, "network_drops", "critical"},
		{"tcp count warning", vminfo.RuntimeStats{TCPCount: 6000}, "tcpconn_high", "warning"},
		{"tcp count critical", vminfo.RuntimeStats{TCPCount: 25000}, "tcpconn_high", "critical"},
		{"conntrack warning", vminfo.RuntimeStats{ConntrackCount: 860, ConntrackMax: 1000}, "conntrack_high", "warning"},
		{"conntrack critical", vminfo.RuntimeStats{ConntrackCount: 960, ConntrackMax: 1000}, "conntrack_high", "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levels := warningLevels(buildHealth(static, tt.stats, nil))
			level, ok := levels[tt.wantCode]
			if !ok {
				t.Fatalf("expected warning %q, got %+v", tt.wantCode, levels)
			}
			if level != tt.wantLevel {
				t.Fatalf("warning %q level = %q, want %q", tt.wantCode, level, tt.wantLevel)
			}
		})
	}
}

func TestBuildHealthNetworkNoFalseAlarm(t *testing.T) {
	static := vminfo.StaticInfo{CPUCores: 2}

	// Cumulative errors with no sustained rate must not alarm (d2caa02 intent).
	for _, w := range buildHealth(static, vminfo.RuntimeStats{
		Interfaces: []vminfo.InterfaceIO{{Name: "eth0", RxErrors: 1}},
	}, nil).Warnings {
		if w.Code == "network_errors" {
			t.Fatalf("cumulative errors without rate should not alarm: %+v", w)
		}
	}

	// Low rates and low TCP count must not alarm.
	for _, w := range buildHealth(static, vminfo.RuntimeStats{
		TCPCount:   100,
		Interfaces: []vminfo.InterfaceIO{{Name: "eth0", RxErrRate: 1, RxDropRate: 1}},
	}, nil).Warnings {
		switch w.Code {
		case "network_errors", "network_drops", "tcpconn_high":
			t.Fatalf("low network activity should not alarm: %+v", w)
		}
	}

	// Conntrack unavailable (max==0, e.g. non-Linux or module not loaded) must not alarm.
	for _, w := range buildHealth(static, vminfo.RuntimeStats{ConntrackCount: 100}, nil).Warnings {
		if w.Code == "conntrack_high" {
			t.Fatalf("conntrack with max==0 should not alarm: %+v", w)
		}
	}
}

func TestBuildHealthFromSnapshotConntrack(t *testing.T) {
	snap := Snapshot{
		System:  SystemInfo{Cores: 2},
		Network: NetworkInfo{ConntrackCount: 960, ConntrackMax: 1000},
	}
	health := buildHealthFromSnapshot(snap)
	if level, ok := warningLevels(health)["conntrack_high"]; !ok || level != "critical" {
		t.Fatalf("expected critical conntrack_high from snapshot, got %+v", health.Warnings)
	}
}
