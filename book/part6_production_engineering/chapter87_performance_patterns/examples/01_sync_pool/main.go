// FILE: book/part6_production_engineering/chapter87_performance_patterns/examples/01_sync_pool/main.go
// CHAPTER: 87 — Performance Patterns
// TOPIC: sync.Pool — object reuse, GC-aware pooling, measuring allocation
//        reduction. Demonstrates correct Put/Get patterns, the GC-reset
//        behaviour, and how to benchmark pool benefit.
//
// Run:
//   go run ./part6_production_engineering/chapter87_performance_patterns/examples/01_sync_pool

package main

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BUFFER POOL — bytes.Buffer reuse via sync.Pool
// ─────────────────────────────────────────────────────────────────────────────

var bufferPool = sync.Pool{
	New: func() any {
		// Allocate a buffer with a reasonable initial capacity.
		b := &bytes.Buffer{}
		b.Grow(512)
		return b
	},
}

func acquireBuffer() *bytes.Buffer {
	b := bufferPool.Get().(*bytes.Buffer)
	b.Reset() // always reset before use
	return b
}

func releaseBuffer(b *bytes.Buffer) {
	// Don't keep huge buffers in the pool; they waste memory.
	if b.Cap() > 64*1024 {
		return
	}
	bufferPool.Put(b)
}

// formatRequest simulates building an HTTP-like request string.
func formatRequest(method, path string, headers map[string]string, body []byte) string {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	fmt.Fprintf(buf, "%s %s HTTP/1.1\r\n", method, path)
	for k, v := range headers {
		fmt.Fprintf(buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")
	buf.Write(body)
	return buf.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// STRUCT POOL — avoid allocating short-lived structs
// ─────────────────────────────────────────────────────────────────────────────

type Request struct {
	ID      int
	Method  string
	Path    string
	Payload []byte
}

var requestPool = sync.Pool{
	New: func() any { return &Request{} },
}

func acquireRequest() *Request  { return requestPool.Get().(*Request) }
func releaseRequest(r *Request) {
	// Zero out before returning — avoid data leaks between callers.
	r.ID = 0
	r.Method = ""
	r.Path = ""
	r.Payload = r.Payload[:0]
	requestPool.Put(r)
}

// ─────────────────────────────────────────────────────────────────────────────
// MEASUREMENT HELPERS
// ─────────────────────────────────────────────────────────────────────────────

type allocStats struct {
	allocs uint64
	bytes  uint64
}

func readMemStats() allocStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return allocStats{ms.Mallocs, ms.TotalAlloc}
}

func (a allocStats) delta(b allocStats) allocStats {
	return allocStats{b.allocs - a.allocs, b.bytes - a.bytes}
}

// ─────────────────────────────────────────────────────────────────────────────
// BENCHMARKS
// ─────────────────────────────────────────────────────────────────────────────

const iterations = 100_000

func withoutPool() {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	body := []byte(`{"ok":true}`)
	for i := 0; i < iterations; i++ {
		_ = formatRequestNoPool("POST", "/api/v1/events", headers, body)
	}
}

func formatRequestNoPool(method, path string, headers map[string]string, body []byte) string {
	var buf bytes.Buffer // new allocation every call
	fmt.Fprintf(&buf, "%s %s HTTP/1.1\r\n", method, path)
	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")
	buf.Write(body)
	return buf.String()
}

func withPool() {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	body := []byte(`{"ok":true}`)
	for i := 0; i < iterations; i++ {
		_ = formatRequest("POST", "/api/v1/events", headers, body)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// POOL GC INTERACTION DEMO
// ─────────────────────────────────────────────────────────────────────────────

func demonstrateGCReset() {
	fmt.Println("--- Pool GC interaction ---")

	// Put several objects in the pool.
	pool := &sync.Pool{New: func() any { return new(int) }}
	for i := 0; i < 5; i++ {
		v := pool.New().(*int)
		*v = i
		pool.Put(v)
	}

	// A GC cycle WILL clear the pool (this is by design).
	runtime.GC()

	// After GC the pool.New factory is called again — count allocations.
	v := pool.Get().(*int)
	fmt.Printf("  Value after GC: %d (pool was cleared, New called)\n", *v)
	pool.Put(v)
	fmt.Println("  Lesson: never rely on pool to STORE data across GC cycles.")
	fmt.Println("  Pool is only a performance hint — not a cache.")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 87: sync.Pool & Object Reuse ===")
	fmt.Println()

	// ── BUFFER POOL DEMO ──────────────────────────────────────────────────────
	fmt.Println("--- Buffer pool demo ---")
	out := formatRequest("GET", "/health", map[string]string{"Accept": "text/plain"}, nil)
	fmt.Printf("  Formatted request (%d bytes):\n", len(out))
	for _, line := range bytes.SplitN([]byte(out), []byte("\r\n"), 3) {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// ── STRUCT POOL DEMO ──────────────────────────────────────────────────────
	fmt.Println("--- Struct pool demo ---")
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := acquireRequest()
			defer releaseRequest(r)
			r.ID = id
			r.Method = "POST"
			r.Path = "/events"
			r.Payload = append(r.Payload, []byte(`{"id":1}`)...)
			// Process request (simulated)
			_ = r.Method + " " + r.Path
		}(i)
	}
	wg.Wait()
	fmt.Println("  5 concurrent requests processed with pooled structs.")
	fmt.Println()

	// ── ALLOCATION COMPARISON ─────────────────────────────────────────────────
	fmt.Println("--- Allocation comparison (100k iterations) ---")
	runtime.GC()
	b1 := readMemStats()
	t1 := time.Now()
	withoutPool()
	d1 := time.Since(t1)
	a1 := b1.delta(readMemStats())

	runtime.GC()
	b2 := readMemStats()
	t2 := time.Now()
	withPool()
	d2 := time.Since(t2)
	a2 := b2.delta(readMemStats())

	fmt.Printf("  Without pool: %7d allocs  %8d bytes  %v\n", a1.allocs, a1.bytes, d1.Round(time.Millisecond))
	fmt.Printf("  With pool:    %7d allocs  %8d bytes  %v\n", a2.allocs, a2.bytes, d2.Round(time.Millisecond))
	if a1.allocs > 0 {
		fmt.Printf("  Allocation reduction: %.1f%%\n", 100*(1-float64(a2.allocs)/float64(a1.allocs)))
	}
	fmt.Println()

	demonstrateGCReset()

	fmt.Println()
	fmt.Println("Key rules:")
	fmt.Println("  1. Always Reset/zero the object before use (not before Put).")
	fmt.Println("  2. Do not store large objects — they waste memory pool-wide.")
	fmt.Println("  3. Pool objects disappear on any GC — never treat as cache.")
	fmt.Println("  4. Pool shines for frequent, short-lived, same-size objects.")
}
