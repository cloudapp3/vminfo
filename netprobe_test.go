package vminfo

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestCheckPortOpenAndClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	open := CheckPort(context.Background(), "127.0.0.1", port, time.Second)
	if !open.Open || open.Err != "" {
		t.Fatalf("expected open port %d, got %+v", port, open)
	}

	// A neighbouring ephemeral port is almost certainly not listening in CI.
	closed := CheckPort(context.Background(), "127.0.0.1", port+1, 200*time.Millisecond)
	if closed.Open || closed.Err == "" {
		t.Fatalf("expected closed port, got %+v", closed)
	}
}

func TestResolveDNSLocalhost(t *testing.T) {
	res := ResolveDNS(context.Background(), "localhost", "")
	if res.Err != "" {
		t.Fatalf("resolve localhost failed: %s", res.Err)
	}
	if len(res.Addrs) == 0 {
		t.Fatal("expected at least one address for localhost")
	}
}

func TestPingTCPOpenAndClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	res := Ping(context.Background(), "127.0.0.1", PingOptions{Mode: "tcp", Port: port, Count: 3, Timeout: time.Second})
	if res.Err != "" || res.Lost != 0 || len(res.RTTs) != 3 {
		t.Fatalf("expected 3 ok probes, got %+v", res)
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	closed := Ping(context.Background(), "127.0.0.1", PingOptions{Mode: "tcp", Port: port, Count: 2, Timeout: 200 * time.Millisecond})
	if closed.Lost != 2 || len(closed.RTTs) != 0 {
		t.Fatalf("expected 2 lost probes, got %+v", closed)
	}
}

func TestNormalizePingOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    PingOptions
		want    PingOptions
		wantErr string
	}{
		{
			name: "defaults",
			want: PingOptions{
				Mode:    "tcp",
				Count:   defaultPingCount,
				Timeout: defaultPingTimeout,
				Port:    defaultPingPort,
			},
		},
		{
			name: "normalizes mode",
			opts: PingOptions{Mode: " ICMP ", Count: 1, Timeout: time.Second},
			want: PingOptions{Mode: "icmp", Count: 1, Timeout: time.Second},
		},
		{name: "negative count", opts: PingOptions{Count: -1}, wantErr: "count"},
		{name: "excessive count", opts: PingOptions{Count: maxPingCount + 1}, wantErr: "count"},
		{name: "negative timeout", opts: PingOptions{Timeout: -time.Second}, wantErr: "timeout"},
		{name: "excessive timeout", opts: PingOptions{Timeout: maxPingTimeout + time.Nanosecond}, wantErr: "timeout"},
		{name: "negative port", opts: PingOptions{Port: -1}, wantErr: "port"},
		{name: "excessive port", opts: PingOptions{Port: 65536}, wantErr: "port"},
		{name: "unsupported mode", opts: PingOptions{Mode: "udp"}, wantErr: "mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePingOptions(tt.opts)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("normalizePingOptions() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizePingOptions() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizePingOptions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPingRejectsUnboundedCount(t *testing.T) {
	res := Ping(context.Background(), "127.0.0.1", PingOptions{Count: 1_000_000_000})
	if res.Err == "" || !strings.Contains(res.Err, "count") {
		t.Fatalf("Ping() error = %q, want count validation error", res.Err)
	}
	if res.Sent != 0 || len(res.RTTs) != 0 {
		t.Fatalf("Ping() performed probes for invalid count: %+v", res)
	}
}

func TestNextProbeDeadlineUsesEarlierContextDeadline(t *testing.T) {
	now := time.Now()
	contextDeadline := now.Add(2 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), contextDeadline)
	defer cancel()

	if got := nextProbeDeadline(ctx, now, 5*time.Second); !got.Equal(contextDeadline) {
		t.Fatalf("nextProbeDeadline() = %v, want context deadline %v", got, contextDeadline)
	}
	if got := nextProbeDeadline(context.Background(), now, time.Second); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("nextProbeDeadline() = %v, want timeout deadline", got)
	}
}
