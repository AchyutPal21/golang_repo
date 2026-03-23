// FILE: 10_advanced_patterns/08_performance_patterns.go
// TOPIC: Performance — profiling, allocation avoidance, escape analysis, GOMAXPROCS
//
// Run: go run 10_advanced_patterns/08_performance_patterns.go

package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// ── SYNC.POOL for allocation reuse ────────────────────────────────────────────
var bufPool = sync.Pool{
	New: func() interface{} { return &strings.Builder{} },
}

func buildStringWithPool(parts []string) string {
	sb := bufPool.Get().(*strings.Builder)
	sb.Reset() // MUST reset before use
	defer bufPool.Put(sb)
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// Without pool — allocates a new Builder each time:
func buildStringNoPool(parts []string) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// ── AVOID INTERFACE BOXING in hot paths ───────────────────────────────────────
// Storing a value in interface{} causes allocation if the value doesn't fit
// in a pointer. Use concrete types in hot paths.

func sumInterface(nums []interface{}) int {
	total := 0
	for _, n := range nums {
		total += n.(int) // boxing + type assertion overhead
	}
	return total
}

func sumConcrete(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n // no boxing, no assertion
	}
	return total
}

// ── PRE-ALLOCATE slices/maps when size is known ────────────────────────────────
func buildSliceNoHint(n int) []int {
	var s []int
	for i := 0; i < n; i++ {
		s = append(s, i) // multiple reallocations as capacity doubles
	}
	return s
}

func buildSliceWithHint(n int) []int {
	s := make([]int, 0, n) // pre-allocate: exactly n capacity, 0 reallocations
	for i := 0; i < n; i++ {
		s = append(s, i)
	}
	return s
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Performance Patterns")
	fmt.Println("════════════════════════════════════════")

	// ── THE 3 RULES OF OPTIMIZATION ──────────────────────────────────────
	fmt.Println(`
── The 3 Rules of Optimization ──

  1. MEASURE FIRST — profile before optimizing.
     Guessing is wrong. Use pprof.

  2. FIND THE BOTTLENECK — optimize the actual hot path.
     Optimizing code that runs 1% of the time gives 1% speedup maximum.

  3. VERIFY IMPROVEMENT — benchmark before and after.
     "Obvious" optimizations often make things WORSE due to cache effects.
`)

	// ── PROFILING COMMANDS ────────────────────────────────────────────────
	fmt.Println("── Profiling with pprof ──")
	fmt.Println(`
  CPU profile:
    go test -cpuprofile=cpu.prof -bench=. ./...
    go tool pprof cpu.prof
    (pprof) top10
    (pprof) web        ← visual call graph (requires graphviz)

  Memory profile:
    go test -memprofile=mem.prof -bench=. ./...
    go tool pprof -alloc_objects mem.prof

  Goroutine profile:
    import _ "net/http/pprof"  // add to main
    go tool pprof http://localhost:6060/debug/pprof/goroutine

  Trace:
    go test -trace=trace.out ./...
    go tool trace trace.out
`)

	// ── ESCAPE ANALYSIS ───────────────────────────────────────────────────
	fmt.Println("── Escape Analysis ──")
	fmt.Println(`
  Variables that escape to the heap are slower (GC pressure).
  Variables that stay on the stack are faster (no GC).

  Check what escapes:
    go build -gcflags="-m" .
    go build -gcflags="-m -m" .   ← more detail

  Common escape causes:
    - Storing pointer in interface{}   → heap
    - Return pointer to local variable → heap (usually)
    - Slice grows beyond stack limit   → heap
    - Closure captures variable        → heap
    - fmt.Sprintf (allocates string)   → heap

  Optimization: use concrete types, avoid interface{} in hot paths
`)

	// ── GOMAXPROCS ────────────────────────────────────────────────────────
	fmt.Println("── GOMAXPROCS ──")
	fmt.Printf("  Current GOMAXPROCS: %d  (default = NumCPU = %d)\n",
		runtime.GOMAXPROCS(0), runtime.NumCPU())
	fmt.Println(`
  CPU-bound work: GOMAXPROCS = NumCPU (default since Go 1.5)
  I/O-bound work: can benefit from higher GOMAXPROCS
  Set via env: GOMAXPROCS=4 go run main.go
`)

	// ── SYNC.POOL demo ────────────────────────────────────────────────────
	fmt.Println("── sync.Pool reduces allocations ──")
	parts := []string{"Hello", ", ", "World", "!"}

	// Demonstrate both work correctly:
	r1 := buildStringWithPool(parts)
	r2 := buildStringNoPool(parts)
	fmt.Printf("  Pool result:    %q\n", r1)
	fmt.Printf("  No-pool result: %q\n", r2)
	fmt.Println("  (Pool version reuses the Builder — fewer allocations in loops)")

	// ── PRE-ALLOCATION ─────────────────────────────────────────────────────
	fmt.Println("\n── Pre-allocate with make ──")
	n := 10
	s1 := buildSliceNoHint(n)
	s2 := buildSliceWithHint(n)
	fmt.Printf("  Both correct: %v == %v: %v\n", s1[:3], s2[:3], fmt.Sprint(s1) == fmt.Sprint(s2))
	fmt.Println("  make([]T, 0, n) → 0 reallocations vs O(log n) without hint")

	// ── STRING BUILDING ────────────────────────────────────────────────────
	fmt.Println("\n── String building performance ──")
	fmt.Println(`
  SLOWEST (O(n²)): string concatenation in loop
    result := ""
    for _, s := range items { result += s }  ← each += allocates a new string

  FAST: strings.Builder
    var sb strings.Builder
    for _, s := range items { sb.WriteString(s) }
    result := sb.String()

  FAST: strings.Join (when items already in slice)
    result := strings.Join(items, "")

  FASTEST: pre-size the Builder
    var sb strings.Builder
    sb.Grow(totalExpectedLen)
    for _, s := range items { sb.WriteString(s) }
`)

	// ── BENCHMARK FORMAT ──────────────────────────────────────────────────
	fmt.Println("── Writing good benchmarks ──")
	fmt.Println(`
  func BenchmarkMyFunc(b *testing.B) {
      // Setup (not timed):
      input := generateInput()
      b.ResetTimer()  // start timing HERE

      for i := 0; i < b.N; i++ {
          MyFunc(input)
      }
  }

  // Parallel benchmark:
  func BenchmarkMyFuncParallel(b *testing.B) {
      input := generateInput()
      b.RunParallel(func(pb *testing.PB) {
          for pb.Next() {
              MyFunc(input)
          }
      })
  }

  Run: go test -bench=BenchmarkMyFunc -benchmem -benchtime=3s -count=3 ./...
  Output: BenchmarkMyFunc-8  1000000  1234 ns/op  256 B/op  3 allocs/op
`)

	fmt.Println("─── SUMMARY ────────────────────────────────")
	fmt.Println("  Profile first: go test -cpuprofile/memprofile")
	fmt.Println("  Escape analysis: go build -gcflags=\"-m\"")
	fmt.Println("  sync.Pool: reuse objects, reduce GC in high-throughput code")
	fmt.Println("  make([]T, 0, n): pre-allocate when size is known")
	fmt.Println("  strings.Builder: O(n) vs + concatenation O(n²)")
	fmt.Println("  Avoid interface{} boxing in tight loops")
	fmt.Println("  GOMAXPROCS = NumCPU by default — good for CPU-bound work")
}
