//go:build linux

package vminfo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type cachedProcess struct {
	createTime int64
	proc       *process.Process
}

var processCache = struct {
	mu    sync.Mutex
	items map[int32]cachedProcess
}{
	items: make(map[int32]cachedProcess),
}

const processTerminateTimeout = 3 * time.Second

func listProcesses(ctx context.Context) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	nextCache := make(map[int32]cachedProcess, len(procs))
	items := make([]ProcessInfo, 0, len(procs))
	for _, procItem := range procs {
		if procItem == nil || procItem.Pid <= 0 {
			continue
		}

		activeProc, createTime := resolveActiveProcess(ctx, procItem)
		if activeProc == nil || createTime <= 0 {
			continue
		}
		nextCache[activeProc.Pid] = cachedProcess{createTime: createTime, proc: activeProc}

		item := ProcessInfo{
			PID:        activeProc.Pid,
			Name:       readProcessName(ctx, activeProc),
			User:       readProcessUser(ctx, activeProc),
			State:      readProcessState(ctx, activeProc),
			CPUPercent: readProcessCPU(ctx, activeProc),
			RSSBytes:   readProcessRSS(ctx, activeProc),
		}
		if ppid, err := activeProc.PpidWithContext(ctx); err == nil {
			item.PPID = ppid
		}
		if memPercent, err := activeProc.MemoryPercentWithContext(ctx); err == nil {
			item.MemoryPercent = memPercent
		}
		items = append(items, item)
	}

	processCache.mu.Lock()
	processCache.items = nextCache
	processCache.mu.Unlock()
	return items, nil
}

func terminateProcess(ctx context.Context, pid int32) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}
	if pid == 1 {
		return fmt.Errorf("refuse to terminate pid 1")
	}
	if pid == int32(os.Getpid()) {
		return fmt.Errorf("refuse to terminate current process")
	}

	procItem, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return err
	}
	if err := procItem.TerminateWithContext(ctx); err != nil {
		return err
	}
	return waitProcessStopped(ctx, procItem, processTerminateTimeout)
}

func resolveActiveProcess(ctx context.Context, current *process.Process) (*process.Process, int64) {
	if current == nil {
		return nil, 0
	}
	createTime, err := current.CreateTimeWithContext(ctx)
	if err != nil || createTime <= 0 {
		return nil, 0
	}

	processCache.mu.Lock()
	defer processCache.mu.Unlock()
	if cached, ok := processCache.items[current.Pid]; ok {
		if cached.createTime == createTime && cached.proc != nil {
			return cached.proc, createTime
		}
	}
	return current, createTime
}

func readProcessName(ctx context.Context, procItem *process.Process) string {
	if procItem == nil {
		return ""
	}
	name, err := procItem.NameWithContext(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func readProcessUser(ctx context.Context, procItem *process.Process) string {
	if procItem == nil {
		return ""
	}
	user, err := procItem.UsernameWithContext(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(user)
}

func readProcessState(ctx context.Context, procItem *process.Process) string {
	if procItem == nil {
		return ""
	}
	states, err := procItem.StatusWithContext(ctx)
	if err != nil || len(states) == 0 {
		return ""
	}
	return strings.TrimSpace(states[0])
}

func readProcessCPU(ctx context.Context, procItem *process.Process) float64 {
	if procItem == nil {
		return 0
	}
	value, err := procItem.PercentWithContext(ctx, 0)
	if err != nil {
		return 0
	}
	return value
}

func readProcessRSS(ctx context.Context, procItem *process.Process) uint64 {
	if procItem == nil {
		return 0
	}
	memInfo, err := procItem.MemoryInfoWithContext(ctx)
	if err != nil || memInfo == nil {
		return 0
	}
	return memInfo.RSS
}

func waitProcessStopped(ctx context.Context, procItem *process.Process, timeout time.Duration) error {
	if procItem == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = processTerminateTimeout
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		running, err := procItem.IsRunningWithContext(waitCtx)
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("process %d did not exit after SIGTERM", procItem.Pid)
		case <-ticker.C:
		}
	}
}
