package vminfo

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func TestCollectAllRefreshesDynamicUptimeWithStaticCache(t *testing.T) {
	original := defaultStaticCache
	defaultStaticCache = &staticCache{ttl: time.Minute}
	t.Cleanup(func() {
		defaultStaticCache = original
	})

	ctx := context.Background()
	_, first, err := CollectAll(ctx, Options{SampleInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(400 * time.Millisecond)
		_, next, err := CollectAll(ctx, Options{SampleInterval: 5 * time.Millisecond})
		if err != nil {
			t.Fatalf("CollectAll returned error: %v", err)
		}
		if next.Uptime > first.Uptime {
			return
		}
	}

	t.Fatalf("expected uptime to increase with cache hit; first=%d", first.Uptime)
}

func TestCalcIfaceSpeedsRates(t *testing.T) {
	start := map[string]netIfaceSample{"eth0": {in: 1000, out: 2000, rxErrors: 10, txErrors: 5, rxDrops: 2, txDrops: 1}}
	end := map[string]netIfaceSample{"eth0": {in: 3000, out: 4000, rxErrors: 25, txErrors: 35, rxDrops: 12, txDrops: 31}}
	addrs := map[string]string{"eth0": "10.0.0.1"}

	got := calcIfaceSpeeds(start, end, addrs, time.Second)
	if len(got) != 1 {
		t.Fatalf("expected 1 interface, got %d (%+v)", len(got), got)
	}
	iface := got[0]
	if iface.RxSpeed != 2000 || iface.TxSpeed != 2000 {
		t.Fatalf("byte speed = (%d, %d), want (2000, 2000)", iface.RxSpeed, iface.TxSpeed)
	}
	if iface.RxErrRate != 15 {
		t.Fatalf("rx error rate = %v, want 15", iface.RxErrRate)
	}
	if iface.TxErrRate != 30 {
		t.Fatalf("tx error rate = %v, want 30", iface.TxErrRate)
	}
	if iface.RxDropRate != 10 {
		t.Fatalf("rx drop rate = %v, want 10", iface.RxDropRate)
	}
	if iface.TxDropRate != 30 {
		t.Fatalf("tx drop rate = %v, want 30", iface.TxDropRate)
	}
}

func TestParseCPUSampleDoesNotDoubleCountGuestTime(t *testing.T) {
	stat := cpu.TimesStat{
		User:      100,
		System:    20,
		Idle:      50,
		Nice:      10,
		Iowait:    5,
		Irq:       2,
		Softirq:   3,
		Steal:     4,
		Guest:     30,
		GuestNice: 5,
	}

	got := parseCPUSample(stat)
	wantTotal := 229.0
	if runtime.GOOS == "linux" {
		wantTotal = 194
	}
	const wantIdle = 55
	if got.total != wantTotal || got.idle != wantIdle {
		t.Fatalf("parseCPUSample() = %+v, want total=%v idle=%v", got, wantTotal, wantIdle)
	}
}

func TestParseCPUSampleCarriesIowait(t *testing.T) {
	stat := cpu.TimesStat{User: 100, System: 20, Idle: 50, Iowait: 7}
	got := parseCPUSample(stat)
	if got.iowait != 7 {
		t.Fatalf("parseCPUSample().iowait = %v, want 7", got.iowait)
	}
}

func TestCalcIOWaitPercent(t *testing.T) {
	cases := []struct {
		name  string
		start cpuSample
		end   cpuSample
		want  float64
	}{
		{"normal window", cpuSample{total: 1000, iowait: 10}, cpuSample{total: 2000, iowait: 110}, 10},
		{"zero delta total", cpuSample{total: 1000, iowait: 10}, cpuSample{total: 1000, iowait: 20}, 0},
		{"negative total", cpuSample{total: 2000, iowait: 10}, cpuSample{total: 1000, iowait: 20}, 0},
		{"negative iowait", cpuSample{total: 1000, iowait: 50}, cpuSample{total: 2000, iowait: 10}, 0},
		{"clamped above 100", cpuSample{total: 1000, iowait: 0}, cpuSample{total: 1100, iowait: 1100}, 100},
		{"zero iowait", cpuSample{total: 1000, iowait: 0}, cpuSample{total: 2000, iowait: 0}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := calcIOWaitPercent(tc.start, tc.end); got != tc.want {
				t.Fatalf("calcIOWaitPercent(%+v, %+v) = %v, want %v", tc.start, tc.end, got, tc.want)
			}
		})
	}
}
