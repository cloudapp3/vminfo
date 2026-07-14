package collector

import (
	"context"
	"testing"
	"time"
)

func TestCollectorStopBeforeStartIsPersistentAndIdempotent(t *testing.T) {
	collector := New(time.Hour)
	collector.Stop()
	collector.Stop()

	done := make(chan struct{})
	go func() {
		collector.Start(context.Background())
		collector.Start(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not honor a prior Stop")
	}
	if got := collector.Latest(); got != nil {
		t.Fatalf("pre-stopped collector unexpectedly collected a snapshot: %+v", got)
	}
}

func TestCollectorLatestReturnsDeepCopy(t *testing.T) {
	collector := New(time.Second)
	collector.snapshot = &Snapshot{
		CPU: CPUInfo{PerCore: []float64{1}, History: []float64{2}},
		Disk: DiskInfo{
			Filesystems: []Filesystem{{Mount: "/"}},
			IO:          []DiskIO{{Device: "sda"}},
		},
		Network: NetworkInfo{
			TCPStates:  map[string]uint32{"ESTABLISHED": 1},
			Interfaces: []NetInterface{{Name: "eth0"}},
		},
		Processes: ProcessInfo{Total: 1, List: []ProcessEntry{{Name: "init"}}},
		Health:    HealthInfo{Warnings: []HealthWarning{{Code: "test"}}},
	}

	first := collector.Latest()
	first.CPU.PerCore[0] = 10
	first.CPU.History[0] = 20
	first.Disk.Filesystems[0].Mount = "/mutated"
	first.Disk.IO[0].Device = "mutated"
	first.Network.TCPStates["ESTABLISHED"] = 10
	first.Network.Interfaces[0].Name = "mutated"
	first.Processes.List[0].Name = "mutated"
	first.Health.Warnings[0].Code = "mutated"

	second := collector.Latest()
	if second.CPU.PerCore[0] != 1 || second.CPU.History[0] != 2 {
		t.Fatalf("CPU slices were mutated through Latest: %+v", second.CPU)
	}
	if second.Disk.Filesystems[0].Mount != "/" || second.Disk.IO[0].Device != "sda" {
		t.Fatalf("disk slices were mutated through Latest: %+v", second.Disk)
	}
	if second.Network.TCPStates["ESTABLISHED"] != 1 || second.Network.Interfaces[0].Name != "eth0" {
		t.Fatalf("network data was mutated through Latest: %+v", second.Network)
	}
	if second.Processes.List[0].Name != "init" || second.Health.Warnings[0].Code != "test" {
		t.Fatalf("process or health slices were mutated through Latest: %+v %+v", second.Processes, second.Health)
	}
}

func TestCollectorLatestJSONReturnsCopy(t *testing.T) {
	collector := New(time.Second)
	collector.cachedJSON = []byte(`{"status":"ok"}`)

	first := collector.LatestJSON()
	first[0] = 'x'
	if got := string(collector.LatestJSON()); got != `{"status":"ok"}` {
		t.Fatalf("LatestJSON cache mutated through caller slice: %q", got)
	}
}
