package vminfo

import "time"

// DefaultSampleInterval is the fallback sampling interval used by runtime
// collection helpers when Options.SampleInterval is not set.
const DefaultSampleInterval = time.Second

// Options configures runtime collection behavior.
type Options struct {
	SampleInterval time.Duration
}

// StaticInfo contains host properties that change rarely across samples.
type StaticInfo struct {
	OS             string `json:"os"`
	Platform       string `json:"platform,omitempty"`
	OSVersion      string `json:"os_version,omitempty"`
	Kernel         string `json:"kernel,omitempty"`
	Arch           string `json:"arch,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	CPUModel       string `json:"cpu_model,omitempty"`
	CPUCores       uint32 `json:"cpu_cores,omitempty"`
	MemTotal       uint64 `json:"mem_total,omitempty"`
	SwapTotal      uint64 `json:"swap_total,omitempty"`
	DiskTotal      uint64 `json:"disk_total,omitempty"`
	Virtualization string `json:"virtualization,omitempty"`
}

// RuntimeStats contains sampled runtime metrics for the local host.
type RuntimeStats struct {
	CPU            float64           `json:"cpu"`
	CPUPerCore     []float64         `json:"cpu_per_core,omitempty"`
	CPUCount       int               `json:"cpu_count,omitempty"`
	CPUFreqMHz     float64           `json:"cpu_freq_mhz,omitempty"`
	MemUsed        uint64            `json:"mem_used,omitempty"`
	SwapUsed       uint64            `json:"swap_used,omitempty"`
	DiskUsed       uint64            `json:"disk_used,omitempty"`
	NetIn          uint64            `json:"net_in,omitempty"`
	NetOut         uint64            `json:"net_out,omitempty"`
	NetInSpeed     uint64            `json:"net_in_speed,omitempty"`
	NetOutSpeed    uint64            `json:"net_out_speed,omitempty"`
	Load1          float64           `json:"load1,omitempty"`
	Load5          float64           `json:"load5,omitempty"`
	Load15         float64           `json:"load15,omitempty"`
	TCPCount       uint32            `json:"tcp_count,omitempty"`
	TCPStates      map[string]uint32 `json:"tcp_states,omitempty"`
	UDPCount       uint32            `json:"udp_count,omitempty"`
	ConntrackCount uint32            `json:"conntrack_count,omitempty"`
	ConntrackMax   uint32            `json:"conntrack_max,omitempty"`
	ProcessCount   uint32            `json:"process_count,omitempty"`
	Uptime         uint64            `json:"uptime,omitempty"`
	DiskIO         []DiskIOStats     `json:"disk_io,omitempty"`
	Temps          []TempReading     `json:"temps,omitempty"`
	Interfaces     []InterfaceIO     `json:"interfaces,omitempty"`
}

// DiskIOStats holds per-device disk I/O statistics.
type DiskIOStats struct {
	Name       string `json:"name"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadSpeed  uint64 `json:"read_speed,omitempty"`
	WriteSpeed uint64 `json:"write_speed,omitempty"`
	ReadCount  uint64 `json:"read_count,omitempty"`
	WriteCount uint64 `json:"write_count,omitempty"`
	IOPS       uint64 `json:"iops,omitempty"`
}

// TempReading represents a single temperature sensor reading.
type TempReading struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature"`
	High        float64 `json:"high,omitempty"`
	Critical    float64 `json:"critical,omitempty"`
}

// InterfaceIO holds per-interface network I/O stats.
type InterfaceIO struct {
	Name     string `json:"name"`
	RxSpeed  uint64 `json:"rx_speed,omitempty"`
	TxSpeed  uint64 `json:"tx_speed,omitempty"`
	IPv4     string `json:"ipv4,omitempty"`
	RxBytes  uint64 `json:"rx_bytes,omitempty"`
	TxBytes  uint64 `json:"tx_bytes,omitempty"`
	RxErrors uint64 `json:"rx_errors,omitempty"`
	TxErrors uint64 `json:"tx_errors,omitempty"`
	RxDrops  uint64 `json:"rx_drops,omitempty"`
	TxDrops  uint64 `json:"tx_drops,omitempty"`
	// Per-second rates derived from consecutive samples; zero until the
	// second sample arrives. Used by health scoring so a long-lived
	// cumulative counter does not cause persistent false alarms.
	RxErrRate  float64 `json:"rx_err_rate,omitempty"`
	TxErrRate  float64 `json:"tx_err_rate,omitempty"`
	RxDropRate float64 `json:"rx_drop_rate,omitempty"`
	TxDropRate float64 `json:"tx_drop_rate,omitempty"`
}

// Snapshot combines static host metadata with sampled runtime metrics.
type Snapshot struct {
	Static StaticInfo   `json:"static"`
	Stats  RuntimeStats `json:"stats"`
}
