package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/collector"
)

type fakeBackend struct {
	snapshot    collector.Snapshot
	snapshotErr error
	processes   []vminfo.ProcessInfo
	processErr  error
	metadata    vminfo.AppMetadata

	dnsDomain string
	dnsServer string
	portHost  string
	port      int
	portWait  time.Duration
	pingHost  string
	pingOpts  vminfo.PingOptions
	lookupIP  string
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{metadata: vminfo.AppMetadata{Name: "vminfo", Version: "vtest"}}
}

func (f *fakeBackend) SystemSnapshot(context.Context) (collector.Snapshot, error) {
	return f.snapshot, f.snapshotErr
}

func (f *fakeBackend) ListProcesses(context.Context) ([]vminfo.ProcessInfo, error) {
	return f.processes, f.processErr
}

func (f *fakeBackend) ResolveDNS(_ context.Context, domain, server string) vminfo.DNSResult {
	f.dnsDomain, f.dnsServer = domain, server
	return vminfo.DNSResult{Domain: domain, Server: server, Addrs: []string{"192.0.2.1"}}
}

func (f *fakeBackend) CheckPort(_ context.Context, host string, port int, timeout time.Duration) vminfo.PortResult {
	f.portHost, f.port, f.portWait = host, port, timeout
	return vminfo.PortResult{Host: host, Port: port, Open: true}
}

func (f *fakeBackend) Ping(_ context.Context, host string, opts vminfo.PingOptions) vminfo.PingResult {
	f.pingHost, f.pingOpts = host, opts
	return vminfo.PingResult{Host: host, Mode: opts.Mode, Port: opts.Port, Sent: opts.Count}
}

func (f *fakeBackend) LookupIP(_ context.Context, ip string) vminfo.IPInfo {
	f.lookupIP = ip
	return vminfo.IPInfo{IP: ip, CountryCode: "ZZ"}
}

func (f *fakeBackend) Metadata() vminfo.AppMetadata { return f.metadata }

func TestToolHandlersGetSystemSnapshot(t *testing.T) {
	backend := newFakeBackend()
	backend.snapshot.System.Hostname = "test-host"
	handlers := &toolHandlers{backend: backend}

	_, got, err := handlers.getSystemSnapshot(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatalf("getSystemSnapshot returned error: %v", err)
	}
	if got.System.Hostname != "test-host" {
		t.Fatalf("hostname = %q, want test-host", got.System.Hostname)
	}

	backend.snapshotErr = errors.New("collector unavailable")
	if _, _, err := handlers.getSystemSnapshot(context.Background(), nil, emptyInput{}); err == nil || !strings.Contains(err.Error(), "collector unavailable") {
		t.Fatalf("getSystemSnapshot error = %v, want collector failure", err)
	}
}

func TestToolHandlersListProcessesFiltersSortsLimitsAndRedacts(t *testing.T) {
	backend := newFakeBackend()
	backend.processes = []vminfo.ProcessInfo{
		{PID: 11, Name: "api", User: "app", CPUPercent: 80, MemoryPercent: 10, Command: "/srv/api --token secret"},
		{PID: 22, Name: "db", User: "postgres", CPUPercent: 20, MemoryPercent: 70, Command: "/usr/bin/db"},
		{PID: 33, Name: "worker", User: "app", CPUPercent: 40, MemoryPercent: 30, Command: "/srv/worker"},
	}
	handlers := &toolHandlers{backend: backend}

	_, got, err := handlers.listProcesses(context.Background(), nil, listProcessesInput{
		SortBy: "mem",
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("listProcesses returned error: %v", err)
	}
	if got.Total != 3 || got.Matched != 3 || got.Returned != 1 {
		t.Fatalf("unexpected counts: %+v", got)
	}
	if got.Processes[0].PID != 22 {
		t.Fatalf("first PID = %d, want 22", got.Processes[0].PID)
	}
	if got.Processes[0].Command != "" || got.CommandIncluded {
		t.Fatalf("command was not redacted: %+v", got)
	}
	_, got, err = handlers.listProcesses(context.Background(), nil, listProcessesInput{Filter: "token secret"})
	if err != nil {
		t.Fatalf("listProcesses with redacted command filter returned error: %v", err)
	}
	if got.Matched != 0 {
		t.Fatalf("redacted command was searchable: %+v", got)
	}

	_, got, err = handlers.listProcesses(context.Background(), nil, listProcessesInput{
		Filter:         "token secret",
		IncludeCommand: true,
	})
	if err != nil {
		t.Fatalf("listProcesses with command returned error: %v", err)
	}
	if got.Matched != 1 || got.Processes[0].PID != 11 || got.Processes[0].Command == "" {
		t.Fatalf("unexpected command-inclusive result: %+v", got)
	}
}

func TestToolHandlersListProcessesRejectsInvalidOptions(t *testing.T) {
	handlers := &toolHandlers{backend: newFakeBackend()}
	tests := []struct {
		name  string
		input listProcessesInput
	}{
		{name: "sort", input: listProcessesInput{SortBy: "rss"}},
		{name: "negative limit", input: listProcessesInput{Limit: -1}},
		{name: "large limit", input: listProcessesInput{Limit: maxProcessLimit + 1}},
		{name: "large filter", input: listProcessesInput{Filter: strings.Repeat("x", maxProcessFilterLen+1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := handlers.listProcesses(context.Background(), nil, test.input); err == nil {
				t.Fatal("listProcesses returned nil error")
			}
		})
	}
}

func TestToolHandlersNetworkDefaults(t *testing.T) {
	backend := newFakeBackend()
	handlers := &toolHandlers{backend: backend}

	if _, _, err := handlers.resolveDNS(context.Background(), nil, resolveDNSInput{Domain: "example.com", Server: "1.1.1.1"}); err != nil {
		t.Fatalf("resolveDNS returned error: %v", err)
	}
	if backend.dnsDomain != "example.com" || backend.dnsServer != "1.1.1.1" {
		t.Fatalf("unexpected DNS call: domain=%q server=%q", backend.dnsDomain, backend.dnsServer)
	}

	if _, _, err := handlers.checkPort(context.Background(), nil, checkPortInput{Host: "example.com", Port: 443}); err != nil {
		t.Fatalf("checkPort returned error: %v", err)
	}
	if backend.portHost != "example.com" || backend.port != 443 || backend.portWait != 2*time.Second {
		t.Fatalf("unexpected port call: host=%q port=%d timeout=%s", backend.portHost, backend.port, backend.portWait)
	}

	if _, _, err := handlers.pingHost(context.Background(), nil, pingHostInput{Host: "example.com"}); err != nil {
		t.Fatalf("pingHost returned error: %v", err)
	}
	if backend.pingHost != "example.com" || backend.pingOpts.Mode != "tcp" || backend.pingOpts.Count != 4 || backend.pingOpts.Timeout != time.Second || backend.pingOpts.Port != 80 {
		t.Fatalf("unexpected ping call: host=%q opts=%+v", backend.pingHost, backend.pingOpts)
	}

	if _, _, err := handlers.lookupIP(context.Background(), nil, lookupIPInput{IP: "2001:db8::1"}); err != nil {
		t.Fatalf("lookupIP returned error: %v", err)
	}
	if backend.lookupIP != "2001:db8::1" {
		t.Fatalf("lookup IP = %q, want 2001:db8::1", backend.lookupIP)
	}
}

func TestToolHandlersRejectInvalidNetworkInput(t *testing.T) {
	handlers := &toolHandlers{backend: newFakeBackend()}
	tests := []struct {
		name string
		call func() error
	}{
		{name: "empty domain", call: func() error {
			_, _, err := handlers.resolveDNS(context.Background(), nil, resolveDNSInput{})
			return err
		}},
		{name: "invalid port", call: func() error {
			_, _, err := handlers.checkPort(context.Background(), nil, checkPortInput{Host: "example.com", Port: 0})
			return err
		}},
		{name: "large port timeout", call: func() error {
			_, _, err := handlers.checkPort(context.Background(), nil, checkPortInput{Host: "example.com", Port: 443, TimeoutMS: 10001})
			return err
		}},
		{name: "invalid ping mode", call: func() error {
			_, _, err := handlers.pingHost(context.Background(), nil, pingHostInput{Host: "example.com", Mode: "udp"})
			return err
		}},
		{name: "large ping count", call: func() error {
			_, _, err := handlers.pingHost(context.Background(), nil, pingHostInput{Host: "example.com", Count: maxPingCount + 1})
			return err
		}},
		{name: "invalid lookup IP", call: func() error {
			_, _, err := handlers.lookupIP(context.Background(), nil, lookupIPInput{IP: "example.com"})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil {
				t.Fatal("call returned nil error")
			}
		})
	}
}
