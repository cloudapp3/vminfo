//go:build !linux

package vminfo

import (
	"context"

	gnet "github.com/shirou/gopsutil/v3/net"
)

func countUDPConnections() uint32 {
	return countConnsGopsutil("udp")
}

func countConnsGopsutil(kind string) uint32 {
	conns, err := gnet.ConnectionsWithContext(context.Background(), kind)
	if err != nil {
		return 0
	}
	return uint32(len(conns))
}

// readTCPStates buckets TCP connections by their decoded status via gopsutil.
// Non-Linux platforms lack /proc/net/tcp, so gopsutil is the cross-platform
// source for both the total count and the state distribution.
func readTCPStates() (uint32, map[string]uint32) {
	conns, err := gnet.ConnectionsWithContext(context.Background(), "tcp")
	if err != nil {
		return 0, nil
	}
	states := make(map[string]uint32, len(conns))
	for _, c := range conns {
		if c.Status != "" {
			states[c.Status]++
		}
	}
	return uint32(len(conns)), states
}

// conntrackUsage is Linux-specific; on other platforms conntracking is not
// exposed via a comparable interface, so report (0, 0) and let callers hide it.
func conntrackUsage() (count, max uint32) {
	return 0, 0
}
