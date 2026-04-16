package vminfo

import (
	"context"
	"testing"
	"time"
)

func TestCollectAllRefreshesDynamicUptimeWithStaticCache(t *testing.T) {
	original := defaultStaticCache
	defaultStaticCache = &staticCache{ttl: time.Minute}
	t.Cleanup(func() {
		defaultStaticCache = original
	})

	ctx := context.Background()
	_, first, err := CollectAll(ctx, Options{SampleInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(400 * time.Millisecond)
		_, next, err := CollectAll(ctx, Options{SampleInterval: 5 * time.Millisecond})
		if err != nil {
			t.Fatalf("CollectAll returned error: %v", err)
		}
		if next.Uptime > first.Uptime {
			return
		}
	}

	t.Fatalf("expected uptime to increase with cache hit; first=%d", first.Uptime)
}
