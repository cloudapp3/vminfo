package vminfo

import (
	"context"
	"net"
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

	closed := Ping(context.Background(), "127.0.0.1", PingOptions{Mode: "tcp", Port: port + 1, Count: 2, Timeout: 200 * time.Millisecond})
	if closed.Lost != 2 || len(closed.RTTs) != 0 {
		t.Fatalf("expected 2 lost probes, got %+v", closed)
	}
}
