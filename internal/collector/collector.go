package collector

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudapp3/vminfo"
)

const (
	maxCPUHistory  = 200
	sampleInterval = 200 * time.Millisecond
)

// ringBuffer is a fixed-size circular buffer for float64 values.
type ringBuffer struct {
	data []float64
	size int
	head int
	full bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{data: make([]float64, size), size: size}
}

func (r *ringBuffer) push(v float64) {
	r.data[r.head] = v
	r.head = (r.head + 1) % r.size
	if r.head == 0 {
		r.full = true
	}
}

func (r *ringBuffer) slice() []float64 {
	if !r.full {
		out := make([]float64, r.head)
		copy(out, r.data[:r.head])
		return out
	}
	out := make([]float64, r.size)
	copy(out, r.data[r.head:])
	copy(out[r.size-r.head:], r.data[:r.head])
	return out
}

// Collector periodically gathers system data and broadcasts snapshots.
type Collector struct {
	mu         sync.RWMutex
	interval   time.Duration
	snapshot   *Snapshot
	history    *ringBuffer
	cachedJSON []byte // pre-serialized snapshot JSON

	subMu sync.RWMutex
	subs  map[string]chan *Snapshot

	stopCh        chan struct{}
	procConsumers int32 // atomic: >0 means someone wants process data
}

// New creates a new Collector that refreshes at the given interval.
func New(interval time.Duration) *Collector {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &Collector{
		interval: interval,
		history:  newRingBuffer(maxCPUHistory),
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

// LatestJSON returns the pre-serialized JSON of the latest snapshot.
func (c *Collector) LatestJSON() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cachedJSON
}

// LatestWithProcesses returns the most recent snapshot, hydrating process
// details on demand when the cached snapshot only contains the total count.
func (c *Collector) LatestWithProcesses(ctx context.Context) *Snapshot {
	c.mu.RLock()
	if c.snapshot == nil {
		c.mu.RUnlock()
		return nil
	}
	snap := *c.snapshot
	c.mu.RUnlock()

	if !needsProcessHydration(snap) {
		return &snap
	}

	procs, err := vminfo.ListProcesses(ctx)
	if err != nil {
		return &snap
	}

	snap.Processes = ProcessInfo{
		Total: len(procs),
		List:  buildProcessEntries(procs),
	}
	return &snap
}

// LatestJSONWithProcesses returns the latest snapshot JSON, hydrating process
// details on demand when the cached snapshot only contains the total count.
func (c *Collector) LatestJSONWithProcesses(ctx context.Context) []byte {
	c.mu.RLock()
	if c.snapshot == nil {
		c.mu.RUnlock()
		return nil
	}
	snap := *c.snapshot
	cached := c.cachedJSON
	c.mu.RUnlock()

	if !needsProcessHydration(snap) {
		return cached
	}

	procs, err := vminfo.ListProcesses(ctx)
	if err != nil {
		return cached
	}

	snap.Processes = ProcessInfo{
		Total: len(procs),
		List:  buildProcessEntries(procs),
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return cached
	}
	return data
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

// RequestProcesses increments the process consumer counter.
// When the counter is >0, collectOnce will gather process data.
func (c *Collector) RequestProcesses() {
	atomic.AddInt32(&c.procConsumers, 1)
}

// ReleaseProcesses decrements the process consumer counter.
func (c *Collector) ReleaseProcesses() {
	atomic.AddInt32(&c.procConsumers, -1)
}

func needsProcessHydration(snap Snapshot) bool {
	return snap.Processes.Total > len(snap.Processes.List)
}

func (c *Collector) collectOnce(ctx context.Context) {
	staticInfo, stats, err := vminfo.CollectAll(ctx, vminfo.Options{SampleInterval: sampleInterval})
	if err != nil {
		log.Printf("collector error: %v", err)
		return
	}

	// Collect processes only when a consumer has requested them
	var procs []vminfo.ProcessInfo
	if atomic.LoadInt32(&c.procConsumers) > 0 {
		procs, _ = vminfo.ListProcesses(ctx)
	}

	// Update CPU history and snapshot in a single lock acquisition
	c.history.push(stats.CPU)
	historyCopy := c.history.slice()

	snap := BuildSnapshot(staticInfo, stats, procs, historyCopy)
	data, _ := json.Marshal(snap)

	c.mu.Lock()
	c.snapshot = &snap
	c.cachedJSON = data
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
