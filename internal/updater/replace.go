//go:build !windows

package updater

import (
	"fmt"
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
func AtomicReplace(newBinary, currentBinary string) error {
	dir := filepath.Dir(currentBinary)
	tmp := filepath.Join(dir, ".vminfo-update-tmp")

	// Copy new binary to temp file in same directory
	src, err := os.Open(newBinary)
	if err != nil {
		return fmt.Errorf("cannot open new binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer dst.Close()

	if _, err := copyFile(dst, src); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot copy binary: %w", err)
	}
	dst.Close()

	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot chmod temp file: %w", err)
	}

	if err := os.Rename(tmp, currentBinary); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot replace binary (try running with appropriate privileges): %w", err)
	}

	return nil
}

func copyFile(dst, src *os.File) (int64, error) {
	info, err := src.Stat()
	if err != nil {
		return 0, err
	}
	return copyFileN(dst, src, info.Size())
}

func copyFileN(dst, src *os.File, size int64) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for written < size {
		n, err := src.Read(buf)
		if n > 0 {
			_, werr := dst.Write(buf[:n])
			if werr != nil {
				return written, werr
			}
			written += int64(n)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return written, err
		}
	}
	return written, nil
}
