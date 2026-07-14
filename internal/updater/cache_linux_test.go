//go:build linux

package updater

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadCacheAtRejectsFIFO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, cacheFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("create cache FIFO: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_, _ = ReadCacheAt(dir)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ReadCacheAt blocked on a FIFO")
	}
}
