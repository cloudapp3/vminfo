//go:build linux

package vminfo

import (
	"context"
	"testing"
)

func TestDecodeTCPState(t *testing.T) {
	tests := []struct {
		hex  string
		want string
	}{
		{"01", "ESTABLISHED"},
		{"02", "SYN_SENT"},
		{"03", "SYN_RECV"},
		{"04", "FIN_WAIT1"},
		{"05", "FIN_WAIT2"},
		{"06", "TIME_WAIT"},
		{"07", "CLOSE"},
		{"08", "CLOSE_WAIT"},
		{"09", "LAST_ACK"},
		{"0A", "LISTEN"}, // uppercase hex
		{"0a", "LISTEN"}, // lowercase hex
		{"0B", "CLOSING"},
		{"FF", ""},      // unknown code
		{"", ""},        // empty
		{"garbage", ""}, // non-hex
	}
	for _, tt := range tests {
		if got := decodeTCPState(tt.hex); got != tt.want {
			t.Errorf("decodeTCPState(%q) = %q, want %q", tt.hex, got, tt.want)
		}
	}
}

func TestConnectionCountsHonorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := countUDPConnections(ctx); got != 0 {
		t.Fatalf("countUDPConnections() = %d, want 0 for canceled context", got)
	}
	count, states := readTCPStates(ctx)
	if count != 0 || len(states) != 0 {
		t.Fatalf("readTCPStates() = (%d, %v), want empty result for canceled context", count, states)
	}
}
