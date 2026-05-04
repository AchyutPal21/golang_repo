// FILE: book/part6_production_engineering/chapter86_pprof/examples/02_memory_profile/main.go
// CHAPTER: 86 — pprof
// TOPIC: Memory profiling — heap allocations, escape analysis, sync.Pool,
//        and goroutine leak detection.
//
// Run:
//   go run ./examples/02_memory_profile

package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// ALLOCATION PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

// allocating — creates a new buffer every call (heap allocation)
func buildResponseSlow(code int, body string) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTP/1.1 %d OK\r\n", code))
	sb.WriteString("Content-Type: text/plain\r\n\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}

// pooled — reuses a buffer via sync.Pool
var bufPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(512)
		return sb
	},
}

func buildResponseFast(code int, body string) []byte {
	sb := bufPool.Get().(*strings.Builder)
	sb.Reset()
	fmt.Fprintf(sb, "HTTP/1.1 %d OK\r\nContent-Type: text/plain\r\n\r\n%s", code, body)
	result := []byte(sb.String())
	bufPool.Put(sb)
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// ESCAPE ANALYSIS DEMO
// ─────────────────────────────────────────────────────────────────────────────

// stays on stack — small struct, no external reference escapes
func stackAlloc() int {
	type Point struct{ X, Y int }
	p := Point{1, 2}
	return p.X + p.Y
}

// escapes to heap — returned pointer causes heap allocation
func heapAlloc() *int {
	x := 42
	return &x
}

// interface boxing causes escape — value boxed in interface{}
func boxed(v int) any {
	return v // int → interface{} causes heap alloc
}

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE LEAK DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

func goroutineCount() int { return runtime.NumGoroutine() }

func leakyGoroutine(ch chan struct{}) {
	go func() {
		<-ch // blocks forever if ch is never closed
	}()
}

func nonLeakyGoroutine(ch chan struct{}) {
	go func() {
		select {
		case <-ch:
		}
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
// MEMORY STATS SNAPSHOT
// ─────────────────────────────────────────────────────────────────────────────

type MemSnapshot struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
	Mallocs    uint64
	Frees      uint64
}

func snapshot() MemSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemSnapshot{m.Alloc, m.TotalAlloc, m.Sys, m.NumGC, m.Mallocs, m.Frees}
}

func (s MemSnapshot) String() string {
	return fmt.Sprintf("alloc=%dKB  total=%dKB  sys=%dKB  gc=%d  mallocs=%d  frees=%d",
		s.Alloc/1024, s.TotalAlloc/1024, s.Sys/1024, s.NumGC, s.Mallocs, s.Frees)
}

func allocsDelta(before, after MemSnapshot) string {
	return fmt.Sprintf("+%d allocs  +%dB",
		after.Mallocs-before.Mallocs, after.TotalAlloc-before.TotalAlloc)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Memory Profiling ===")
	fmt.Println()

	// ── MEMORY STATS ─────────────────────────────────────────────────────────
	fmt.Println("--- Memory stats before workload ---")
	before := snapshot()
	fmt.Println(" ", before)

	// ── ALLOCATION COMPARISON ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Allocation comparison: slow vs pooled (1000 calls) ---")

	n := 1000
	runtime.GC()
	s1 := snapshot()
	for i := 0; i < n; i++ {
		buildResponseSlow(200, "Hello World")
	}
	e1 := snapshot()

	runtime.GC()
	s2 := snapshot()
	for i := 0; i < n; i++ {
		buildResponseFast(200, "Hello World")
	}
	e2 := snapshot()

	fmt.Printf("  buildResponseSlow (n=%d): %s\n", n, allocsDelta(s1, e1))
	fmt.Printf("  buildResponseFast (n=%d): %s\n", n, allocsDelta(s2, e2))

	// ── ESCAPE ANALYSIS ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Escape analysis examples ---")
	_ = stackAlloc()
	_ = heapAlloc()
	_ = boxed(42)
	fmt.Println(`  stackAlloc()  → stays on stack; no heap allocation
  heapAlloc()   → returned *int forces heap allocation
  boxed(42)     → int boxed into interface{} → heap allocation

  Verify with:  go build -gcflags='-m' ./...
  Output:
    ./main.go:45:6: &x escapes to heap
    ./main.go:51:9: v escapes to heap`)

	// ── GOROUTINE LEAK DETECTION ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Goroutine leak detection ---")

	before_g := goroutineCount()

	// Start goroutines with a properly-closed channel.
	ch := make(chan struct{})
	for i := 0; i < 5; i++ {
		nonLeakyGoroutine(ch)
	}
	close(ch) // signals all goroutines to exit

	// Give goroutines time to exit.
	runtime.Gosched()
	runtime.Gosched()

	fmt.Printf("  goroutines before: %d\n", before_g)
	fmt.Printf("  goroutines after close: %d (should be ~%d)\n", goroutineCount(), before_g)

	// ── MEMORY PROFILING WORKFLOW ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Memory profiling workflow ---")
	fmt.Println(`  // Heap profile:
  go test -bench=BenchmarkBuildResponse -memprofile=mem.prof -benchmem ./...
  go tool pprof -alloc_objects mem.prof  // allocation count
  go tool pprof -alloc_space   mem.prof  // bytes allocated
  go tool pprof -inuse_objects mem.prof  // live objects
  go tool pprof -inuse_space   mem.prof  // live bytes

  // HTTP endpoint:
  curl http://localhost:6060/debug/pprof/heap > heap.prof
  go tool pprof heap.prof

  // Goroutine leak check:
  curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -50

  // Allocation profile in code:
  import "runtime/pprof"
  f, _ := os.Create("mem.prof")
  defer f.Close()
  pprof.WriteHeapProfile(f)`)

	// ── SYNC.POOL RULES ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- sync.Pool rules ---")
	fmt.Println(`  Use sync.Pool when:
    - Objects are short-lived and frequently allocated
    - The type is safe to reuse (Reset() between uses)
    - You've profiled and confirmed allocation is the bottleneck

  Don't use sync.Pool when:
    - Objects have different sizes (pool can't reuse effectively)
    - Objects hold external resources (file handles, connections)
    - The benefit isn't confirmed by benchmarks

  Pattern:
    pool := sync.Pool{New: func() any { return &Buffer{} }}
    buf := pool.Get().(*Buffer)
    buf.Reset()
    // ... use buf ...
    pool.Put(buf)  // return to pool for reuse`)
}
