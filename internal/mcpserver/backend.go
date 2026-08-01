package mcpserver

import (
	"context"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/collector"
)

const snapshotSampleInterval = 200 * time.Millisecond

// Backend isolates MCP tool handling from host and network collection so tool
// contracts can be tested without depending on the machine running the tests.
type Backend interface {
	SystemSnapshot(context.Context) (collector.Snapshot, error)
	ListProcesses(context.Context) ([]vminfo.ProcessInfo, error)
	ResolveDNS(context.Context, string, string) vminfo.DNSResult
	CheckPort(context.Context, string, int, time.Duration) vminfo.PortResult
	Ping(context.Context, string, vminfo.PingOptions) vminfo.PingResult
	LookupIP(context.Context, string) vminfo.IPInfo
	Metadata() vminfo.AppMetadata
}

type nativeBackend struct{}

func (nativeBackend) SystemSnapshot(ctx context.Context) (collector.Snapshot, error) {
	staticInfo, stats, err := vminfo.CollectAll(ctx, vminfo.Options{SampleInterval: snapshotSampleInterval})
	if err != nil {
		return collector.Snapshot{}, err
	}
	return collector.BuildSnapshot(staticInfo, stats, nil, nil), nil
}

func (nativeBackend) ListProcesses(ctx context.Context) ([]vminfo.ProcessInfo, error) {
	return vminfo.ListProcesses(ctx)
}

func (nativeBackend) ResolveDNS(ctx context.Context, domain, server string) vminfo.DNSResult {
	return vminfo.ResolveDNS(ctx, domain, server)
}

func (nativeBackend) CheckPort(ctx context.Context, host string, port int, timeout time.Duration) vminfo.PortResult {
	return vminfo.CheckPort(ctx, host, port, timeout)
}

func (nativeBackend) Ping(ctx context.Context, host string, opts vminfo.PingOptions) vminfo.PingResult {
	return vminfo.Ping(ctx, host, opts)
}

func (nativeBackend) LookupIP(ctx context.Context, ip string) vminfo.IPInfo {
	return vminfo.LookupIP(ctx, ip, vminfo.DefaultIPLookupServer)
}

func (nativeBackend) Metadata() vminfo.AppMetadata {
	return vminfo.Metadata()
}
