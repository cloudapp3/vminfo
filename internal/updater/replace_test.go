//go:build !windows

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	newBinary := filepath.Join(dir, "new-vminfo")
	currentBinary := filepath.Join(dir, "vminfo")
	if err := os.WriteFile(newBinary, []byte("new binary"), 0o600); err != nil {
		t.Fatalf("write new binary: %v", err)
	}
	if err := os.WriteFile(currentBinary, []byte("old binary"), 0o700); err != nil {
		t.Fatalf("write current binary: %v", err)
	}

	if err := AtomicReplace(newBinary, currentBinary); err != nil {
		t.Fatalf("AtomicReplace returned error: %v", err)
	}

	data, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("read replaced binary: %v", err)
	}
	if got := string(data); got != "new binary" {
		t.Fatalf("replaced binary = %q, want %q", got, "new binary")
	}
	info, err := os.Stat(currentBinary)
	if err != nil {
		t.Fatalf("stat replaced binary: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("replaced binary mode = %o, want 755", got)
	}
	assertNoUpdateTemps(t, dir)
}

func TestAtomicReplaceConcurrent(t *testing.T) {
	const replacements = 16

	dir := t.TempDir()
	currentBinary := filepath.Join(dir, "vminfo")
	if err := os.WriteFile(currentBinary, []byte("old binary"), 0o700); err != nil {
		t.Fatalf("write current binary: %v", err)
	}

	wantContents := make(map[string]struct{}, replacements)
	newBinaries := make([]string, replacements)
	for i := range replacements {
		content := fmt.Sprintf("new binary %d", i)
		wantContents[content] = struct{}{}
		newBinaries[i] = filepath.Join(dir, fmt.Sprintf("new-vminfo-%d", i))
		if err := os.WriteFile(newBinaries[i], []byte(content), 0o600); err != nil {
			t.Fatalf("write new binary %d: %v", i, err)
		}
	}

	start := make(chan struct{})
	errs := make(chan error, replacements)
	var wg sync.WaitGroup
	for _, newBinary := range newBinaries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- AtomicReplace(newBinary, currentBinary)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("AtomicReplace returned error: %v", err)
		}
	}
	if t.Failed() {
		return
	}

	data, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("read replaced binary: %v", err)
	}
	if _, ok := wantContents[string(data)]; !ok {
		t.Fatalf("replaced binary has unexpected contents %q", string(data))
	}
	assertNoUpdateTemps(t, dir)
}

func TestAtomicReplaceDoesNotFollowLegacyTempSymlink(t *testing.T) {
	dir := t.TempDir()
	newBinary := filepath.Join(dir, "new-vminfo")
	currentBinary := filepath.Join(dir, "vminfo")
	victim := filepath.Join(dir, "victim")
	legacyTemp := filepath.Join(dir, ".vminfo-update-tmp")

	if err := os.WriteFile(newBinary, []byte("new binary"), 0o600); err != nil {
		t.Fatalf("write new binary: %v", err)
	}
	if err := os.WriteFile(currentBinary, []byte("old binary"), 0o700); err != nil {
		t.Fatalf("write current binary: %v", err)
	}
	if err := os.WriteFile(victim, []byte("do not modify"), 0o600); err != nil {
		t.Fatalf("write victim: %v", err)
	}
	if err := os.Symlink(victim, legacyTemp); err != nil {
		t.Fatalf("create legacy temp symlink: %v", err)
	}

	if err := AtomicReplace(newBinary, currentBinary); err != nil {
		t.Fatalf("AtomicReplace returned error: %v", err)
	}

	data, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read victim: %v", err)
	}
	if got := string(data); got != "do not modify" {
		t.Fatalf("victim contents = %q, want %q", got, "do not modify")
	}
	info, err := os.Lstat(legacyTemp)
	if err != nil {
		t.Fatalf("lstat legacy temp symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("legacy temp path mode = %v, want symlink", info.Mode())
	}
}

func TestAtomicReplaceCleansUpAfterRenameFailure(t *testing.T) {
	dir := t.TempDir()
	newBinary := filepath.Join(dir, "new-vminfo")
	currentBinary := filepath.Join(dir, "vminfo")
	if err := os.WriteFile(newBinary, []byte("new binary"), 0o600); err != nil {
		t.Fatalf("write new binary: %v", err)
	}
	if err := os.Mkdir(currentBinary, 0o700); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(currentBinary, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("write target directory entry: %v", err)
	}

	if err := AtomicReplace(newBinary, currentBinary); err == nil {
		t.Fatal("AtomicReplace returned nil error, want rename failure")
	}
	assertNoUpdateTemps(t, dir)
}

func assertNoUpdateTemps(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".vminfo-update-*"))
	if err != nil {
		t.Fatalf("glob update temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("update temp files were not cleaned up: %v", matches)
	}
}
