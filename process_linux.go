//go:build linux

package vminfo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/tklauser/go-sysconf"
)

const (
	processTerminateTimeout = 3 * time.Second
	procListWorkers         = 32
)

// procClockTicks resolves _SC_CLK_TCK once. Falls back to 100 (x86_64
// default) if sysconf fails. Some kernels use 250 or 1000 (CONFIG_HZ),
// so a hardcode breaks CPU% on ARM and embedded hosts.
var procClockTicks = func() int64 {
	v, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err != nil || v <= 0 {
		return 100
	}
	return v
}()

func listProcesses(ctx context.Context) ([]ProcessInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pids, err := readProcPids()
	if err != nil {
		return nil, err
	}

	systemUptime, _ := readProcUptime()
	memTotal, _ := readMemTotalBytes()
	users := readPasswdMap()

	type result struct {
		info ProcessInfo
		ok   bool
	}

	jobs := make(chan int32, len(pids))
	out := make(chan result, len(pids))

	var wg sync.WaitGroup
	workers := procListWorkers
	if workers > len(pids) {
		workers = len(pids)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pid := range jobs {
				if info, ok := readProcEntry(pid, systemUptime, memTotal, users); ok {
					out <- result{info: info, ok: true}
				}
			}
		}()
	}
	for _, pid := range pids {
		jobs <- pid
	}
	close(jobs)
	wg.Wait()
	close(out)

	items := make([]ProcessInfo, 0, len(pids))
	for r := range out {
		items = append(items, r.info)
	}
	return items, nil
}

func readProcEntry(pid int32, systemUptime float64, memTotal uint64, users map[uint32]string) (ProcessInfo, bool) {
	stat, ok := readProcStat(pid)
	if !ok {
		return ProcessInfo{}, false
	}

	rssBytes := readProcRSSBytes(pid)
	userName := ""
	if uid, ok := readProcUID(pid); ok {
		userName = lookupUser(uid, users)
	}

	clkTck := float64(procClockTicks)
	procUptimeSecs := systemUptime - float64(stat.starttime)/clkTck
	cpuPercent := 0.0
	if procUptimeSecs > 0 {
		totalSecs := float64(stat.utime+stat.stime) / clkTck
		cpuPercent = totalSecs / procUptimeSecs * 100
	}

	memPercent := float32(0)
	if memTotal > 0 {
		memPercent = float32(float64(rssBytes) / float64(memTotal) * 100)
	}

	uptime := uint64(0)
	if procUptimeSecs > 0 {
		uptime = uint64(procUptimeSecs)
	}

	return ProcessInfo{
		PID:           pid,
		PPID:          stat.ppid,
		Name:          stat.comm,
		User:          userName,
		State:         stat.state,
		CPUPercent:    cpuPercent,
		MemoryPercent: memPercent,
		RSSBytes:      rssBytes,
		Threads:       stat.numThreads,
		Nice:          stat.nice,
		Uptime:        uptime,
	}, true
}

func readProcPids() ([]int32, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	pids := make([]int32, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		v, err := strconv.ParseInt(name, 10, 32)
		if err != nil || v <= 0 {
			continue
		}
		pids = append(pids, int32(v))
	}
	return pids, nil
}

type procStat struct {
	comm       string
	state      string
	ppid       int32
	numThreads int32
	nice       int32
	utime      uint64
	stime      uint64
	starttime  uint64
}

// readProcStat parses /proc/<pid>/stat. Format:
// pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt
// majflt cmajflt utime(14) stime(15) cutime cstime priority nice(19)
// num_threads(20) itrealvalue starttime(22) ...
func readProcStat(pid int32) (procStat, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return procStat{}, false
	}
	rOpen := bytes.IndexByte(data, '(')
	rClose := bytes.LastIndexByte(data, ')')
	if rOpen < 0 || rClose < 0 || rClose <= rOpen {
		return procStat{}, false
	}
	comm := string(data[rOpen+1 : rClose])
	rest := data[rClose+2:] // skip ") "
	fields := bytes.Fields(rest)
	if len(fields) < 20 {
		return procStat{}, false
	}
	// fields[0] = state, [1] = ppid, [11] = utime, [12] = stime,
	// [16] = nice, [17] = num_threads, [19] = starttime
	s := procStat{comm: comm, state: string(fields[0])}
	if v, err := strconv.ParseInt(string(fields[1]), 10, 32); err == nil {
		s.ppid = int32(v)
	}
	if v, err := strconv.ParseUint(string(fields[11]), 10, 64); err == nil {
		s.utime = v
	}
	if v, err := strconv.ParseUint(string(fields[12]), 10, 64); err == nil {
		s.stime = v
	}
	if v, err := strconv.ParseInt(string(fields[16]), 10, 32); err == nil {
		s.nice = int32(v)
	}
	if v, err := strconv.ParseInt(string(fields[17]), 10, 32); err == nil {
		s.numThreads = int32(v)
	}
	if v, err := strconv.ParseUint(string(fields[19]), 10, 64); err == nil {
		s.starttime = v
	}
	return s, true
}

// readProcRSSBytes parses /proc/<pid>/statm — fields are pages.
// Format: size resident shared text lib data dt
func readProcRSSBytes(pid int32) uint64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0
	}
	fields := bytes.Fields(data)
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(string(fields[1]), 10, 64)
	if err != nil {
		return 0
	}
	return pages * uint64(os.Getpagesize())
}

// readProcUID parses /proc/<pid>/status for the real UID (Uid: line).
func readProcUID(pid int32) (uint32, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, false
	}
	idx := bytes.Index(data, []byte("\nUid:"))
	if idx < 0 {
		if !bytes.HasPrefix(data, []byte("Uid:")) {
			return 0, false
		}
		idx = -1
	}
	rest := data[idx+1:] // skip newline (or use full when prefix)
	end := bytes.IndexByte(rest, '\n')
	if end < 0 {
		end = len(rest)
	}
	line := rest[:end]
	fields := bytes.Fields(line)
	if len(fields) < 2 {
		return 0, false
	}
	v, err := strconv.ParseUint(string(fields[1]), 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

// readProcUptime returns system uptime in seconds.
func readProcUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := bytes.Fields(data)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty uptime")
	}
	return strconv.ParseFloat(string(fields[0]), 64)
}

// readMemTotalBytes returns MemTotal from /proc/meminfo.
func readMemTotalBytes() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("malformed MemTotal")
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("MemTotal not found")
}

// readPasswdMap returns uid → username from /etc/passwd. On error, empty map.
func readPasswdMap() map[uint32]string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return map[uint32]string{}
	}
	defer f.Close()
	m := make(map[uint32]string, 64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 3 {
			continue
		}
		uid, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil {
			continue
		}
		if _, exists := m[uint32(uid)]; !exists {
			m[uint32(uid)] = parts[0]
		}
	}
	return m
}

// lookupUser resolves uid → username, preferring the cached /etc/passwd
// map and falling back to os/user.LookupId for NSS-backed users (LDAP,
// SSSD, nss_systemd). Numeric UID is the last resort.
func lookupUser(uid uint32, cached map[uint32]string) string {
	if name, ok := cached[uid]; ok && name != "" {
		return name
	}
	nssUserCache.mu.RLock()
	v, ok := nssUserCache.m[uid]
	nssUserCache.mu.RUnlock()
	if ok {
		if v == "" {
			return strconv.FormatUint(uint64(uid), 10)
		}
		return v
	}
	resolved := ""
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil && u.Username != "" {
		resolved = u.Username
	}
	nssUserCache.mu.Lock()
	nssUserCache.m[uid] = resolved
	nssUserCache.mu.Unlock()
	if resolved != "" {
		return resolved
	}
	return strconv.FormatUint(uint64(uid), 10)
}

// nssUserCache memoizes os/user.LookupId results across listProcesses
// calls. NSS lookups can hit a remote directory (LDAP/SSSD), so caching
// avoids spending hundreds of cgo calls per refresh. Empty-string value
// means "looked up, not found" — still prevents repeat lookups.
var nssUserCache = struct {
	mu sync.RWMutex
	m  map[uint32]string
}{m: make(map[uint32]string, 16)}

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
