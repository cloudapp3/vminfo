//go:build linux

package vminfo

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func countUDPConnections() uint32 {
	return countConnsFromFile("/proc/net/udp") + countConnsFromFile("/proc/net/udp6")
}

func countConnsFromFile(path string) uint32 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var count uint32
	scanner.Scan() // skip header line
	for scanner.Scan() {
		count++
	}
	return count
}

// readTCPStates scans /proc/net/tcp and /proc/net/tcp6 in a single pass,
// returning the total TCP socket count and a per-state distribution keyed by
// state name (ESTABLISHED, TIME_WAIT, ...). countTCPConnections reuses this so
// the kernel table files are read once per sample, not twice.
func readTCPStates() (uint32, map[string]uint32) {
	var count uint32
	states := make(map[string]uint32)
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip header line
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				continue
			}
			count++
			if name := decodeTCPState(fields[3]); name != "" {
				states[name]++
			}
		}
		f.Close()
	}
	return count, states
}

// decodeTCPState maps a /proc/net/tcp hex state code (4th field) to its name.
func decodeTCPState(hexCode string) string {
	switch strings.ToLower(strings.TrimSpace(hexCode)) {
	case "01":
		return "ESTABLISHED"
	case "02":
		return "SYN_SENT"
	case "03":
		return "SYN_RECV"
	case "04":
		return "FIN_WAIT1"
	case "05":
		return "FIN_WAIT2"
	case "06":
		return "TIME_WAIT"
	case "07":
		return "CLOSE"
	case "08":
		return "CLOSE_WAIT"
	case "09":
		return "LAST_ACK"
	case "0a":
		return "LISTEN"
	case "0b":
		return "CLOSING"
	}
	return ""
}

// conntrackUsage returns the current nf_conntrack entry count and the
// configured maximum. Returns (0, 0) when conntrack is unavailable (e.g.
// inside containers without the module loaded).
func conntrackUsage() (count, max uint32) {
	return readConntrackFile("/proc/sys/net/netfilter/nf_conntrack_count"),
		readConntrackFile("/proc/sys/net/netfilter/nf_conntrack_max")
}

func readConntrackFile(path string) uint32 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 32)
	return uint32(n)
}
