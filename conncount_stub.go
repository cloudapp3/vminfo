//go:build !linux

package vminfo

import (
	"context"

	gnet "github.com/shirou/gopsutil/v3/net"
)

func countTCPConnections() uint32 {
	return countConnsGopsutil("tcp")
}

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
