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
