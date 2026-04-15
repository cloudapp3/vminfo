package vminfo

import "context"

// ProcessInfo describes one local process entry returned by ListProcesses.
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	PPID          int32   `json:"ppid,omitempty"`
	Name          string  `json:"name,omitempty"`
	User          string  `json:"user,omitempty"`
	State         string  `json:"state,omitempty"`
	CPUPercent    float64 `json:"cpu_percent,omitempty"`
	MemoryPercent float32 `json:"memory_percent,omitempty"`
	RSSBytes      uint64  `json:"rss_bytes,omitempty"`
	Threads       int32   `json:"threads,omitempty"`
	Nice          int32   `json:"nice,omitempty"`
	Uptime        uint64  `json:"uptime,omitempty"`
}

// ListProcesses returns local processes on Linux and an unsupported error on
// other platforms.
func ListProcesses(ctx context.Context) ([]ProcessInfo, error) {
	return listProcesses(ctx)
}

// TerminateProcess sends SIGTERM to the given process on Linux and returns an
// unsupported error on other platforms.
func TerminateProcess(ctx context.Context, pid int32) error {
	return terminateProcess(ctx, pid)
}
