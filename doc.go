// Package vminfo collects local host metrics and runs lightweight network
// diagnostics for use inside other Go programs.
//
// It is the library behind the vminfo CLI: the same functions feed the terminal
// UI, the web dashboard, and the one-shot commands. Import it when you need host
// information or network probes in your own tool without shelling out to an
// external binary.
//
// Collection is split into two layers that match how the underlying values
// change:
//
//   - [CollectStatic] returns rarely-changing host properties: CPU model and
//     core count, total memory and swap, total disk, hostname, OS, kernel, and
//     architecture.
//   - [CollectStats] samples runtime metrics: overall and per-core CPU usage,
//     memory and swap in use, network and disk I/O with per-second rates, TCP
//     and UDP counts, conntrack saturation, TCP state distribution, load
//     averages, per-interface error/drop rates, temperatures, and uptime. Rates
//     are derived from consecutive samples, so the first call returns zero
//     rates; call it on a steady cadence of Options.SampleInterval (default
//     [DefaultSampleInterval]) for stable values.
//   - [CollectAll] returns both in a single call.
//
// Network diagnostics are independent of the collectors:
//
//   - [ResolveDNS] queries a resolver for a domain.
//   - [CheckPort] reports whether a TCP port is reachable.
//   - [Ping] measures TCP round-trip latency to a host.
//   - [LookupIP] returns network metadata for an IP address.
//
// Process listing ([ListProcesses]) and termination ([TerminateProcess]) are
// Linux-only; they return an unsupported error on other platforms.
//
// Example:
//
//	static, _ := vminfo.CollectStatic(ctx)
//	stats, _ := vminfo.CollectStats(ctx, vminfo.Options{SampleInterval: time.Second})
//	fmt.Println(static.Hostname, stats.CPU)
//
// The interactive terminal UI is a separate, importable package at
// [github.com/cloudapp3/vminfo/tui]. The web dashboard lives under internal/
// and is not importable.
package vminfo
