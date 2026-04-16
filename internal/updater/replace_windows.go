//go:build windows

package updater

import "errors"

// SelfPath returns an error on Windows.
func SelfPath() (string, error) {
	return "", errors.New("self-update is not supported on Windows")
}

// AtomicReplace returns an error on Windows.
func AtomicReplace(_, _ string) error {
	return errors.New("self-update is not supported on Windows")
}
