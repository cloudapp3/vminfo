//go:build linux

package vminfo

import (
	"bufio"
	"os"
)

func countTCPConnections() uint32 {
	return countConnsFromFile("/proc/net/tcp") + countConnsFromFile("/proc/net/tcp6")
}

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
