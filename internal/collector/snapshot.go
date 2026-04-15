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
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	User       string  `json:"user"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float32 `json:"mem_percent"`
	RSS        uint64  `json:"rss"`
	Status     string  `json:"status"`
	Command    string  `json:"command"`
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
		memPercent = float64(stats.MemUsed) / float64(static.MemTotal) * 100
		avail = static.MemTotal - stats.MemUsed
	}

	swapPercent := float64(0)
	if static.SwapTotal > 0 {
		swapPercent = float64(stats.SwapUsed) / float64(static.SwapTotal) * 100
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

	// Full process list sorted by CPU
	procList := buildProcessEntries(procs)

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
			Percent:     math.Round(memPercent*10) / 10,
			SwapTotal:   static.SwapTotal,
			SwapUsed:    stats.SwapUsed,
			SwapPercent: math.Round(swapPercent*10) / 10,
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
			PID:        p.PID,
			Name:       p.Name,
			User:       p.User,
			CPUPercent: math.Round(p.CPUPercent*10) / 10,
			MemPercent: p.MemoryPercent,
			RSS:        p.RSSBytes,
			Status:     p.State,
			Command:    p.Name,
		}
	}
	return entries
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
