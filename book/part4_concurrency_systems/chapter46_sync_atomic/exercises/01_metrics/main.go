// FILE: book/part4_concurrency_systems/chapter46_sync_atomic/exercises/01_metrics/main.go
// CHAPTER: 46 — sync/atomic
// EXERCISE: Lock-free metrics collector using atomic.Int64, atomic.Value
//           for snapshot export, and a rate calculator.
//
// Run (from the chapter folder):
//   go run ./exercises/01_metrics

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// COUNTER — lock-free increment/decrement
// ─────────────────────────────────────────────────────────────────────────────

type Counter struct{ v atomic.Int64 }

func (c *Counter) Inc()           { c.v.Add(1) }
func (c *Counter) Add(n int64)    { c.v.Add(n) }
func (c *Counter) Value() int64   { return c.v.Load() }
func (c *Counter) Reset() int64   { return c.v.Swap(0) }

// ─────────────────────────────────────────────────────────────────────────────
// GAUGE — tracks a current value (can go up and down)
// ─────────────────────────────────────────────────────────────────────────────

type Gauge struct{ v atomic.Int64 }

func (g *Gauge) Set(n int64)    { g.v.Store(n) }
func (g *Gauge) Inc()           { g.v.Add(1) }
func (g *Gauge) Dec()           { g.v.Add(-1) }
func (g *Gauge) Value() int64   { return g.v.Load() }

// ─────────────────────────────────────────────────────────────────────────────
// METRICS REGISTRY — holds named counters and gauges, exports snapshots
// ─────────────────────────────────────────────────────────────────────────────

type Snapshot struct {
	Counters map[string]int64
	Gauges   map[string]int64
	Time     time.Time
}

type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	snapshot atomic.Value // stores *Snapshot
}

func NewRegistry() *Registry {
	r := &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
	}
	r.takeSnapshot()
	return r
}

func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{}
	r.counters[name] = c
	return c
}

func (r *Registry) Gauge(name string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.gauges[name]; ok {
		return g
	}
	g := &Gauge{}
	r.gauges[name] = g
	return g
}

func (r *Registry) takeSnapshot() {
	r.mu.RLock()
	snap := &Snapshot{
		Counters: make(map[string]int64, len(r.counters)),
		Gauges:   make(map[string]int64, len(r.gauges)),
		Time:     time.Now(),
	}
	for k, c := range r.counters {
		snap.Counters[k] = c.Value()
	}
	for k, g := range r.gauges {
		snap.Gauges[k] = g.Value()
	}
	r.mu.RUnlock()
	r.snapshot.Store(snap)
}

func (r *Registry) Snapshot() *Snapshot {
	r.takeSnapshot()
	return r.snapshot.Load().(*Snapshot)
}

// ─────────────────────────────────────────────────────────────────────────────
// RATE CALCULATOR — requests per second using atomic window
// ─────────────────────────────────────────────────────────────────────────────

type RateCounter struct {
	current  atomic.Int64
	previous atomic.Int64
	lastTick atomic.Int64 // unix nanos
}

func NewRateCounter() *RateCounter {
	r := &RateCounter{}
	r.lastTick.Store(time.Now().UnixNano())
	return r
}

func (r *RateCounter) Inc() { r.current.Add(1) }

// Rate returns estimated events per second since last call.
func (r *RateCounter) Rate() float64 {
	now := time.Now().UnixNano()
	last := r.lastTick.Swap(now)
	elapsed := time.Duration(now - last)
	if elapsed <= 0 {
		return 0
	}
	count := r.current.Swap(0)
	return float64(count) / elapsed.Seconds()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	reg := NewRegistry()

	requests := reg.Counter("http.requests")
	errors := reg.Counter("http.errors")
	activeConns := reg.Gauge("connections.active")

	// Simulate concurrent HTTP traffic.
	fmt.Println("=== Simulating concurrent HTTP traffic ===")

	var wg sync.WaitGroup
	for i := range 500 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			requests.Inc()
			activeConns.Inc()
			defer activeConns.Dec()

			time.Sleep(time.Duration(id%5) * time.Millisecond)

			if id%10 == 0 { // 10% error rate
				errors.Inc()
			}
		}(i)
	}
	wg.Wait()

	snap := reg.Snapshot()
	fmt.Printf("  requests:    %d\n", snap.Counters["http.requests"])
	fmt.Printf("  errors:      %d  (%.1f%%)\n",
		snap.Counters["http.errors"],
		float64(snap.Counters["http.errors"])/float64(snap.Counters["http.requests"])*100)
	fmt.Printf("  active conns: %d  (should be 0 after all done)\n",
		snap.Gauges["connections.active"])

	// Rate counter demo.
	fmt.Println()
	fmt.Println("=== Rate counter ===")

	rate := NewRateCounter()
	done := make(chan struct{})

	// Generator: 1000 events/sec.
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rate.Inc()
			}
		}
	}()

	// Sample rate every 100ms for 3 samples.
	for i := range 3 {
		time.Sleep(100 * time.Millisecond)
		r := rate.Rate()
		fmt.Printf("  sample %d: %.0f events/sec\n", i+1, r)
	}
	close(done)

	// CAS-based max tracker.
	fmt.Println()
	fmt.Println("=== CAS max tracker ===")

	var maxVal atomic.Int64
	var wg2 sync.WaitGroup
	for i := int64(0); i < 100; i++ {
		wg2.Add(1)
		go func(v int64) {
			defer wg2.Done()
			for {
				cur := maxVal.Load()
				if v <= cur {
					return
				}
				if maxVal.CompareAndSwap(cur, v) {
					return
				}
			}
		}(i)
	}
	wg2.Wait()
	fmt.Printf("  max of 0..99: %d  (always 99)\n", maxVal.Load())
}
