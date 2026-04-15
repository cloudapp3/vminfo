package collector

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cloudapp3/vminfo"
)

const (
	maxCPUHistory  = 200
	sampleInterval = 200 * time.Millisecond
)

// Collector periodically gathers system data and broadcasts snapshots.
type Collector struct {
	mu       sync.RWMutex
	interval time.Duration
	snapshot *Snapshot
	history  []float64

	subMu sync.RWMutex
	subs  map[string]chan *Snapshot

	stopCh chan struct{}
}

// New creates a new Collector that refreshes at the given interval.
func New(interval time.Duration) *Collector {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &Collector{
		interval: interval,
		history:  make([]float64, 0, maxCPUHistory),
		subs:     make(map[string]chan *Snapshot),
		stopCh:   make(chan struct{}),
	}
}

// Subscribe registers a subscriber and returns a channel for snapshots.
func (c *Collector) Subscribe(id string) <-chan *Snapshot {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	ch := make(chan *Snapshot, 5)
	c.subs[id] = ch
	return ch
}

// Unsubscribe removes a subscriber.
func (c *Collector) Unsubscribe(id string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	if ch, ok := c.subs[id]; ok {
		close(ch)
		delete(c.subs, id)
	}
}

// Latest returns the most recent snapshot (for REST API).
func (c *Collector) Latest() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

// Start begins the collection loop. Blocks until Stop is called.
func (c *Collector) Start(ctx context.Context) {
	c.collectOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collectOnce(ctx)
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop signals the collector to stop.
func (c *Collector) Stop() {
	select {
	case c.stopCh <- struct{}{}:
	default:
	}
}

func (c *Collector) collectOnce(ctx context.Context) {
	staticInfo, stats, err := vminfo.CollectAll(ctx, vminfo.Options{SampleInterval: sampleInterval})
	if err != nil {
		log.Printf("collector error: %v", err)
		return
	}

	// Collect processes (best-effort)
	procs, _ := vminfo.ListProcesses(ctx)

	// Update CPU history
	c.mu.Lock()
	c.history = append(c.history, stats.CPU)
	if len(c.history) > maxCPUHistory {
		c.history = c.history[len(c.history)-maxCPUHistory:]
	}
	historyCopy := make([]float64, len(c.history))
	copy(historyCopy, c.history)
	c.mu.Unlock()

	snap := BuildSnapshot(staticInfo, stats, procs, historyCopy)

	c.mu.Lock()
	c.snapshot = &snap
	c.mu.Unlock()

	// Broadcast to subscribers (non-blocking)
	c.subMu.RLock()
	for _, ch := range c.subs {
		select {
		case ch <- &snap:
		default:
		}
	}
	c.subMu.RUnlock()
}
