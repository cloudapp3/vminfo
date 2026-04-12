package vminfo

import "context"

type ProcessInfo struct {
	PID           int32   `json:"pid"`
	PPID          int32   `json:"ppid,omitempty"`
	Name          string  `json:"name,omitempty"`
	User          string  `json:"user,omitempty"`
	State         string  `json:"state,omitempty"`
	CPUPercent    float64 `json:"cpu_percent,omitempty"`
	MemoryPercent float32 `json:"memory_percent,omitempty"`
	RSSBytes      uint64  `json:"rss_bytes,omitempty"`
}

func ListProcesses(ctx context.Context) ([]ProcessInfo, error) {
	return listProcesses(ctx)
}

func TerminateProcess(ctx context.Context, pid int32) error {
	return terminateProcess(ctx, pid)
}
