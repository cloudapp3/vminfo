package collector

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/cloudapp3/vminfo"
)

// Snapshot represents a complete system state at a point in time.
type Snapshot struct {
	Timestamp time.Time   `json:"timestamp"`
	System    SystemInfo  `json:"system"`
	CPU       CPUInfo     `json:"cpu"`
	Memory    MemoryInfo  `json:"memory"`
	Disk      DiskInfo    `json:"disk"`
	Network   NetworkInfo `json:"network"`
	Load      LoadInfo    `json:"load"`
	Processes ProcessInfo `json:"processes"`
	Health    HealthInfo  `json:"health"`
}

type SystemInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Kernel    string `json:"kernel"`
	Arch      string `json:"arch"`
	CPUModel  string `json:"cpu_model"`
	Cores     int    `json:"cores"`
	UptimeSec uint64 `json:"uptime_seconds"`
	UptimeStr string `json:"uptime_human"`
}

type CPUInfo struct {
	TotalPercent float64   `json:"total_percent"`
	AvgPercent   float64   `json:"avg_percent"`
	MaxPercent   float64   `json:"max_percent"`
	FreqMHz      float64   `json:"frequency_mhz"`
	PerCore      []float64 `json:"per_core"`
	History      []float64 `json:"history"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	Percent     float64 `json:"percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapPercent float64 `json:"swap_percent"`
}

type DiskInfo struct {
	Filesystems []Filesystem `json:"filesystems"`
	IO          []DiskIO     `json:"io"`
}

type Filesystem struct {
	Mount   string  `json:"mount"`
	Device  string  `json:"device"`
	FSType  string  `json:"fstype"`
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Percent float64 `json:"percent"`
}

type DiskIO struct {
	Device       string `json:"device"`
	ReadByteSec  uint64 `json:"read_bytes_sec"`
	WriteByteSec uint64 `json:"write_bytes_sec"`
	IOPS         uint64 `json:"iops"`
}

type NetworkInfo struct {
	TotalDownloadSec uint64         `json:"total_download_sec"`
	TotalUploadSec   uint64         `json:"total_upload_sec"`
	TCPConns         uint32         `json:"tcp_connections"`
	UDPConns         uint32         `json:"udp_connections"`
	Interfaces       []NetInterface `json:"interfaces"`
}

type NetInterface struct {
	Name        string `json:"name"`
	DownloadSec uint64 `json:"download_sec"`
	UploadSec   uint64 `json:"upload_sec"`
	IPv4        string `json:"ipv4,omitempty"`
	RxBytes     uint64 `json:"rx_bytes,omitempty"`
	TxBytes     uint64 `json:"tx_bytes,omitempty"`
	RxErrors    uint64 `json:"rx_errors,omitempty"`
	TxErrors    uint64 `json:"tx_errors,omitempty"`
	RxDrops     uint64 `json:"rx_drops,omitempty"`
	TxDrops     uint64 `json:"tx_drops,omitempty"`
}

type LoadInfo struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type ProcessInfo struct {
	Total int            `json:"total"`
	List  []ProcessEntry `json:"list"`
}

type ProcessEntry struct {
	PID           int32   `json:"pid"`
	PPID          int32   `json:"ppid,omitempty"`
	Name          string  `json:"name"`
	User          string  `json:"user"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemPercent    float32 `json:"mem_percent"`
	RSS           uint64  `json:"rss"`
	Status        string  `json:"status"`
	Command       string  `json:"command"`
	Threads       int32   `json:"threads,omitempty"`
	Nice          int32   `json:"nice,omitempty"`
	Uptime        uint64  `json:"uptime,omitempty"`
	StartedAtUnix int64   `json:"started_at_unix,omitempty"`
}

type HealthInfo struct {
	Score    int             `json:"score"`
	Warnings []HealthWarning `json:"warnings"`
}

type HealthWarning struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BuildSnapshot creates a Snapshot from existing vminfo package data.
func BuildSnapshot(
	static vminfo.StaticInfo,
	stats vminfo.RuntimeStats,
	procs []vminfo.ProcessInfo,
	cpuHistory []float64,
) Snapshot {
	memPercent := float64(0)
	avail := uint64(0)
	if static.MemTotal > 0 {
		memPercent = percent(stats.MemUsed, static.MemTotal)
		avail = static.MemTotal - stats.MemUsed
	}

	swapPercent := float64(0)
	if static.SwapTotal > 0 {
		swapPercent = percent(stats.SwapUsed, static.SwapTotal)
	}

	// CPU per-core stats
	var avgCore, maxCore float64
	if len(stats.CPUPerCore) > 0 {
		var sum float64
		for _, v := range stats.CPUPerCore {
			sum += v
			if v > maxCore {
				maxCore = v
			}
		}
		avgCore = sum / float64(len(stats.CPUPerCore))
	}

	history := make([]float64, len(cpuHistory))
	copy(history, cpuHistory)

	// Disk I/O entries
	ioEntries := make([]DiskIO, 0, len(stats.DiskIO))
	for _, d := range stats.DiskIO {
		ioEntries = append(ioEntries, DiskIO{
			Device:       d.Name,
			ReadByteSec:  d.ReadSpeed,
			WriteByteSec: d.WriteSpeed,
			IOPS:         d.IOPS,
		})
	}

	// Network interfaces
	ifaces := make([]NetInterface, 0, len(stats.Interfaces))
	for _, iface := range stats.Interfaces {
		ifaces = append(ifaces, NetInterface{
			Name:        iface.Name,
			DownloadSec: iface.RxSpeed,
			UploadSec:   iface.TxSpeed,
			IPv4:        iface.IPv4,
			RxBytes:     iface.RxBytes,
			TxBytes:     iface.TxBytes,
			RxErrors:    iface.RxErrors,
			TxErrors:    iface.TxErrors,
			RxDrops:     iface.RxDrops,
			TxDrops:     iface.TxDrops,
		})
	}

	procList := buildProcessEntries(procs)
	health := buildHealth(static, stats, procList)

	return Snapshot{
		Timestamp: time.Now(),
		System: SystemInfo{
			Hostname:  static.Hostname,
			OS:        formatOS(static),
			Kernel:    static.Kernel,
			Arch:      static.Arch,
			CPUModel:  static.CPUModel,
			Cores:     int(static.CPUCores),
			UptimeSec: stats.Uptime,
			UptimeStr: formatUptime(stats.Uptime),
		},
		CPU: CPUInfo{
			TotalPercent: stats.CPU,
			AvgPercent:   avgCore,
			MaxPercent:   maxCore,
			FreqMHz:      stats.CPUFreqMHz,
			PerCore:      stats.CPUPerCore,
			History:      history,
		},
		Memory: MemoryInfo{
			Total:       static.MemTotal,
			Used:        stats.MemUsed,
			Available:   avail,
			Percent:     memPercent,
			SwapTotal:   static.SwapTotal,
			SwapUsed:    stats.SwapUsed,
			SwapPercent: swapPercent,
		},
		Disk: DiskInfo{
			Filesystems: buildFilesystems(static, stats),
			IO:          ioEntries,
		},
		Network: NetworkInfo{
			TotalDownloadSec: stats.NetInSpeed,
			TotalUploadSec:   stats.NetOutSpeed,
			TCPConns:         stats.TCPCount,
			UDPConns:         stats.UDPCount,
			Interfaces:       ifaces,
		},
		Load: LoadInfo{
			Load1:  stats.Load1,
			Load5:  stats.Load5,
			Load15: stats.Load15,
		},
		Processes: ProcessInfo{
			Total: int(stats.ProcessCount),
			List:  procList,
		},
		Health: health,
	}
}

func buildProcessEntries(procs []vminfo.ProcessInfo) []ProcessEntry {
	items := make([]vminfo.ProcessInfo, len(procs))
	copy(items, procs)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CPUPercent > items[j].CPUPercent
	})
	entries := make([]ProcessEntry, len(items))
	for i, p := range items {
		entries[i] = ProcessEntry{
			PID:           p.PID,
			PPID:          p.PPID,
			Name:          p.Name,
			User:          p.User,
			CPUPercent:    math.Round(p.CPUPercent*10) / 10,
			MemPercent:    p.MemoryPercent,
			RSS:           p.RSSBytes,
			Status:        p.State,
			Command:       firstNonEmpty(p.Command, p.Name),
			Threads:       p.Threads,
			Nice:          p.Nice,
			Uptime:        p.Uptime,
			StartedAtUnix: p.StartedAtUnix,
		}
	}
	return entries
}

func buildHealth(static vminfo.StaticInfo, stats vminfo.RuntimeStats, procs []ProcessEntry) HealthInfo {
	warnings := make([]HealthWarning, 0, 4)
	add := func(level, code, message string) {
		warnings = append(warnings, HealthWarning{
			Level:   level,
			Code:    code,
			Message: message,
		})
	}

	if stats.CPU >= 90 {
		add("critical", "cpu_high", fmt.Sprintf("CPU usage is %.1f%%", stats.CPU))
	} else if stats.CPU >= 75 {
		add("warning", "cpu_high", fmt.Sprintf("CPU usage is %.1f%%", stats.CPU))
	}

	cores := float64(static.CPUCores)
	if cores <= 0 {
		cores = float64(stats.CPUCount)
	}
	if cores > 0 {
		loadRatio := stats.Load1 / cores
		if loadRatio >= 1.5 {
			add("critical", "load_high", fmt.Sprintf("1m load %.2f is %.1fx CPU cores", stats.Load1, loadRatio))
		} else if loadRatio >= 1.0 {
			add("warning", "load_high", fmt.Sprintf("1m load %.2f is %.1fx CPU cores", stats.Load1, loadRatio))
		}
	}

	memPercent := percent(stats.MemUsed, static.MemTotal)
	if memPercent >= 90 {
		add("critical", "memory_high", fmt.Sprintf("memory usage is %.1f%%", memPercent))
	} else if memPercent >= 80 {
		add("warning", "memory_high", fmt.Sprintf("memory usage is %.1f%%", memPercent))
	}

	swapPercent := percent(stats.SwapUsed, static.SwapTotal)
	if swapPercent >= 50 {
		add("warning", "swap_high", fmt.Sprintf("swap usage is %.1f%%", swapPercent))
	}

	diskPercent := percent(stats.DiskUsed, static.DiskTotal)
	if diskPercent >= 95 {
		add("critical", "disk_high", fmt.Sprintf("disk usage is %.1f%%", diskPercent))
	} else if diskPercent >= 85 {
		add("warning", "disk_high", fmt.Sprintf("disk usage is %.1f%%", diskPercent))
	}

	for _, iface := range stats.Interfaces {
		totalWarn := iface.RxErrors + iface.TxErrors + iface.RxDrops + iface.TxDrops
		if totalWarn > 0 {
			add("warning", "network_errors", fmt.Sprintf("%s has %d errors/drops", iface.Name, totalWarn))
			break
		}
	}

	for _, proc := range procs {
		if proc.CPUPercent >= 90 {
			add("warning", "process_cpu_high", fmt.Sprintf("process %s (%d) uses %.1f%% CPU", firstNonEmpty(proc.Name, proc.Command, "-"), proc.PID, proc.CPUPercent))
			break
		}
	}

	score := 100
	for _, warning := range warnings {
		if warning.Level == "critical" {
			score -= 20
			continue
		}
		score -= 10
	}
	if score < 0 {
		score = 0
	}
	return HealthInfo{
		Score:    score,
		Warnings: warnings,
	}
}

func buildHealthFromSnapshot(snap Snapshot) HealthInfo {
	static := vminfo.StaticInfo{
		CPUCores:  uint32(snap.System.Cores),
		MemTotal:  snap.Memory.Total,
		DiskTotal: firstFilesystemTotal(snap.Disk.Filesystems),
		SwapTotal: snap.Memory.SwapTotal,
	}
	stats := vminfo.RuntimeStats{
		CPU:        snap.CPU.TotalPercent,
		CPUCount:   snap.System.Cores,
		MemUsed:    snap.Memory.Used,
		SwapUsed:   snap.Memory.SwapUsed,
		DiskUsed:   firstFilesystemUsed(snap.Disk.Filesystems),
		Load1:      snap.Load.Load1,
		Interfaces: snapshotInterfaces(snap.Network.Interfaces),
	}
	return buildHealth(static, stats, snap.Processes.List)
}

func firstFilesystemTotal(filesystems []Filesystem) uint64 {
	if len(filesystems) == 0 {
		return 0
	}
	return filesystems[0].Total
}

func firstFilesystemUsed(filesystems []Filesystem) uint64 {
	if len(filesystems) == 0 {
		return 0
	}
	return filesystems[0].Used
}

func snapshotInterfaces(items []NetInterface) []vminfo.InterfaceIO {
	out := make([]vminfo.InterfaceIO, 0, len(items))
	for _, item := range items {
		out = append(out, vminfo.InterfaceIO{
			Name:     item.Name,
			RxErrors: item.RxErrors,
			TxErrors: item.TxErrors,
			RxDrops:  item.RxDrops,
			TxDrops:  item.TxDrops,
		})
	}
	return out
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(used)/float64(total)*1000) / 10
}

func buildFilesystems(static vminfo.StaticInfo, stats vminfo.RuntimeStats) []Filesystem {
	if static.DiskTotal == 0 {
		return nil
	}
	pct := float64(stats.DiskUsed) / float64(static.DiskTotal) * 100
	return []Filesystem{{
		Mount:   "/",
		Device:  "total",
		FSType:  "aggregate",
		Total:   static.DiskTotal,
		Used:    stats.DiskUsed,
		Percent: math.Round(pct*10) / 10,
	}}
}

func formatOS(s vminfo.StaticInfo) string {
	os := s.Platform
	if s.OSVersion != "" {
		os += " " + s.OSVersion
	}
	return os
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		return "0s"
	}
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
