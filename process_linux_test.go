//go:build linux

package vminfo

import (
	"context"
	"testing"
)

func TestListProcesses(t *testing.T) {
	items, err := listProcesses(context.Background())
	if err != nil {
		t.Fatalf("listProcesses() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("listProcesses() returned no processes")
	}
}

func TestLookupUserUsesLocalPasswdSnapshot(t *testing.T) {
	users := map[uint32]string{1000: "local-user"}
	if got := lookupUser(1000, users); got != "local-user" {
		t.Fatalf("lookupUser() = %q, want local-user", got)
	}
	if got := lookupUser(424242, users); got != "424242" {
		t.Fatalf("lookupUser() = %q, want numeric UID", got)
	}
}
