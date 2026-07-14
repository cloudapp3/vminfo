//go:build !windows

package updater

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SelfPath returns the absolute path of the currently running executable,
// resolving any symlinks.
func SelfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

// AtomicReplace replaces the binary at currentBinary with the new binary at
// newBinary. It writes to a temp file in the same directory and renames,
// which is atomic on Linux and macOS when on the same filesystem.
func AtomicReplace(newBinary, currentBinary string) (retErr error) {
	dir := filepath.Dir(currentBinary)

	src, err := os.Open(newBinary)
	if err != nil {
		return fmt.Errorf("cannot open new binary: %w", err)
	}
	srcOpen := true
	defer func() {
		if !srcOpen {
			return
		}
		if err := src.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("cannot close new binary: %w", err))
		}
	}()

	dst, err := os.CreateTemp(dir, ".vminfo-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmp := dst.Name()
	dstOpen := true
	keepTemp := true
	defer func() {
		if dstOpen {
			if err := dst.Close(); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("cannot close temp file: %w", err))
			}
		}
		if keepTemp {
			if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
				retErr = errors.Join(retErr, fmt.Errorf("cannot remove temp file: %w", err))
			}
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("cannot copy binary: %w", err)
	}
	if err := src.Close(); err != nil {
		srcOpen = false
		return fmt.Errorf("cannot close new binary: %w", err)
	}
	srcOpen = false

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("cannot sync temp file: %w", err)
	}
	if err := dst.Close(); err != nil {
		dstOpen = false
		return fmt.Errorf("cannot close temp file: %w", err)
	}
	dstOpen = false

	if err := os.Chmod(tmp, 0o755); err != nil {
		return fmt.Errorf("cannot chmod temp file: %w", err)
	}

	if err := os.Rename(tmp, currentBinary); err != nil {
		return fmt.Errorf("cannot replace binary (try running with appropriate privileges): %w", err)
	}
	keepTemp = false

	return nil
}
