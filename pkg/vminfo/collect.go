package vminfo

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type hostBase struct {
	hostInfo  *host.InfoStat
	cpuInfos  []cpu.InfoStat
	memInfo   *mem.VirtualMemoryStat
	swapInfo  *mem.SwapMemoryStat
	diskTotal uint64
	diskUsed  uint64
}

type cpuSample struct {
	total float64
	idle  float64
}

type netSample struct {
	in  uint64
	out uint64
}

func CollectStatic(ctx context.Context) (StaticInfo, error) {
	base, err := readHostBase(ctx)
	if err != nil {
		return StaticInfo{}, err
	}
	return buildStaticInfo(ctx, base), nil
}

func CollectStats(ctx context.Context, opts Options) (RuntimeStats, error) {
	base, err := readHostBase(ctx)
	if err != nil {
		return RuntimeStats{}, err
	}
	return buildRuntimeStats(ctx, withDefaults(opts), base)
}

func CollectAll(ctx context.Context, opts Options) (StaticInfo, RuntimeStats, error) {
	base, err := readHostBase(ctx)
	if err != nil {
		return StaticInfo{}, RuntimeStats{}, err
	}
	staticInfo := buildStaticInfo(ctx, base)
	stats, err := buildRuntimeStats(ctx, withDefaults(opts), base)
	if err != nil {
		return staticInfo, RuntimeStats{}, err
	}
	return staticInfo, stats, nil
}

func withDefaults(opts Options) Options {
	if opts.SampleInterval <= 0 {
		opts.SampleInterval = DefaultSampleInterval
	}
	return opts
}

func readHostBase(ctx context.Context) (hostBase, error) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return hostBase{}, err
	}

	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return hostBase{}, err
	}

	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return hostBase{}, err
	}

	swapInfo, _ := mem.SwapMemoryWithContext(ctx)
	if swapInfo == nil {
		swapInfo = &mem.SwapMemoryStat{}
	}

	diskTotal, diskUsed := readDiskTotals(ctx)
	return hostBase{
		hostInfo:  hostInfo,
		cpuInfos:  cpuInfos,
		memInfo:   memInfo,
		swapInfo:  swapInfo,
		diskTotal: diskTotal,
		diskUsed:  diskUsed,
	}, nil
}

func buildStaticInfo(ctx context.Context, base hostBase) StaticInfo {
	return StaticInfo{
		OS:             base.hostInfo.OS,
		Platform:       base.hostInfo.Platform,
		OSVersion:      base.hostInfo.PlatformVersion,
		Kernel:         base.hostInfo.KernelVersion,
		Arch:           base.hostInfo.KernelArch,
		Hostname:       base.hostInfo.Hostname,
		CPUModel:       readCPUModel(base.cpuInfos),
		CPUCores:       readCPUCores(ctx, base.cpuInfos),
		MemTotal:       base.memInfo.Total,
		SwapTotal:      base.swapInfo.Total,
		DiskTotal:      base.diskTotal,
		Virtualization: readVirtualizationType(),
	}
}

func buildRuntimeStats(ctx context.Context, opts Options, base hostBase) (RuntimeStats, error) {
	stats := RuntimeStats{
		MemUsed:  base.memInfo.Used,
		SwapUsed: base.swapInfo.Used,
		DiskUsed: base.diskUsed,
		Uptime:   base.hostInfo.Uptime,
	}

	startCPU, cpuStartErr := readCPUSample(ctx)
	startCores, coreStartErr := readCPUCoreSamples(ctx)
	startNet, netStartErr := readNetSample(ctx)
	startTime := time.Now()
	if err := sleepWithContext(ctx, opts.SampleInterval); err != nil {
		return stats, err
	}
	elapsed := time.Since(startTime)
	endCPU, cpuEndErr := readCPUSample(ctx)
	endCores, coreEndErr := readCPUCoreSamples(ctx)
	endNet, netEndErr := readNetSample(ctx)

	if cpuStartErr == nil && cpuEndErr == nil {
		stats.CPU = calcCPUUsage(startCPU, endCPU)
	}

	if coreStartErr == nil && coreEndErr == nil && len(startCores) == len(endCores) {
		stats.CPUPerCore = make([]float64, len(startCores))
		for i := range startCores {
			stats.CPUPerCore[i] = calcCPUUsage(startCores[i], endCores[i])
		}
		stats.CPUCount = len(stats.CPUPerCore)
	}

	if netEndErr == nil {
		stats.NetIn = endNet.in
		stats.NetOut = endNet.out
		if netStartErr == nil {
			stats.NetInSpeed, stats.NetOutSpeed = calcNetSpeed(startNet, endNet, elapsed)
		}
	} else if netStartErr == nil {
		stats.NetIn = startNet.in
		stats.NetOut = startNet.out
	}

	if avg, err := load.AvgWithContext(ctx); err == nil {
		stats.Load1 = avg.Load1
		stats.Load5 = avg.Load5
		stats.Load15 = avg.Load15
	}

	stats.TCPCount = countConnections(ctx, "tcp")
	stats.UDPCount = countConnections(ctx, "udp")
	stats.ProcessCount = countProcesses(ctx)
	return stats, nil
}

func readCPUModel(infos []cpu.InfoStat) string {
	if len(infos) == 0 {
		return ""
	}
	return infos[0].ModelName
}

func readCPUCores(ctx context.Context, infos []cpu.InfoStat) uint32 {
	cores, err := cpu.CountsWithContext(ctx, false)
	if err != nil || cores <= 0 {
		cores, _ = cpu.CountsWithContext(ctx, true)
	}
	if cores <= 0 {
		cores = len(infos)
	}
	if cores < 0 {
		return 0
	}
	return uint32(cores)
}

func readCPUSample(ctx context.Context) (cpuSample, error) {
	times, err := cpu.TimesWithContext(ctx, false)
	if err != nil || len(times) == 0 {
		return cpuSample{}, err
	}
	return parseCPUSample(times[0]), nil
}

func readCPUCoreSamples(ctx context.Context) ([]cpuSample, error) {
	times, err := cpu.TimesWithContext(ctx, true)
	if err != nil || len(times) == 0 {
		return nil, err
	}
	samples := make([]cpuSample, len(times))
	for i, stat := range times {
		samples[i] = parseCPUSample(stat)
	}
	return samples, nil
}

func parseCPUSample(stat cpu.TimesStat) cpuSample {
	idle := stat.Idle + stat.Iowait
	total := stat.User + stat.System + stat.Idle + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal + stat.Guest + stat.GuestNice
	return cpuSample{total: total, idle: idle}
}

func calcCPUUsage(start, end cpuSample) float64 {
	total := end.total - start.total
	idle := end.idle - start.idle
	if total <= 0 {
		return 0
	}
	used := total - idle
	if used < 0 {
		return 0
	}
	return (used / total) * 100
}

func readNetSample(ctx context.Context) (netSample, error) {
	stats, err := gnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return netSample{}, err
	}
	var in uint64
	var out uint64
	for _, stat := range stats {
		in += stat.BytesRecv
		out += stat.BytesSent
	}
	return netSample{in: in, out: out}, nil
}

func calcNetSpeed(start, end netSample, elapsed time.Duration) (uint64, uint64) {
	if elapsed <= 0 {
		return 0, 0
	}
	deltaIn := diffUint64(end.in, start.in)
	deltaOut := diffUint64(end.out, start.out)
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		return 0, 0
	}
	return uint64(float64(deltaIn) / seconds), uint64(float64(deltaOut) / seconds)
}

func diffUint64(end, start uint64) uint64 {
	if end < start {
		return 0
	}
	return end - start
}

func readDiskTotals(ctx context.Context) (uint64, uint64) {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return 0, 0
	}
	seen := make(map[string]struct{}, len(parts))
	var total uint64
	var used uint64
	for _, part := range parts {
		key := part.Device + "|" + part.Mountpoint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		usage, err := disk.UsageWithContext(ctx, part.Mountpoint)
		if err != nil {
			continue
		}
		total += usage.Total
		used += usage.Used
	}
	return total, used
}

func countConnections(ctx context.Context, kind string) uint32 {
	conns, err := gnet.ConnectionsWithContext(ctx, kind)
	if err != nil {
		return 0
	}
	return uint32(len(conns))
}

func countProcesses(ctx context.Context) uint32 {
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return 0
	}
	return uint32(len(pids))
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readVirtualizationType() string {
	system, _, err := host.Virtualization()
	if err != nil || system == "" {
		return "-"
	}
	return system
}
