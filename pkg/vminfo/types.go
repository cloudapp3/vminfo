package vminfo

import "time"

const DefaultSampleInterval = time.Second

type Options struct {
	SampleInterval time.Duration
}

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

type RuntimeStats struct {
	CPU          float64   `json:"cpu"`
	CPUPerCore   []float64 `json:"cpu_per_core,omitempty"`
	CPUCount     int       `json:"cpu_count,omitempty"`
	MemUsed      uint64    `json:"mem_used,omitempty"`
	SwapUsed     uint64    `json:"swap_used,omitempty"`
	DiskUsed     uint64    `json:"disk_used,omitempty"`
	NetIn        uint64    `json:"net_in,omitempty"`
	NetOut       uint64    `json:"net_out,omitempty"`
	NetInSpeed   uint64    `json:"net_in_speed,omitempty"`
	NetOutSpeed  uint64    `json:"net_out_speed,omitempty"`
	Load1        float64   `json:"load1,omitempty"`
	Load5        float64   `json:"load5,omitempty"`
	Load15       float64   `json:"load15,omitempty"`
	TCPCount     uint32    `json:"tcp_count,omitempty"`
	UDPCount     uint32    `json:"udp_count,omitempty"`
	ProcessCount uint32    `json:"process_count,omitempty"`
	Uptime       uint64    `json:"uptime,omitempty"`
}

type Snapshot struct {
	Static StaticInfo   `json:"static"`
	Stats  RuntimeStats `json:"stats"`
}
