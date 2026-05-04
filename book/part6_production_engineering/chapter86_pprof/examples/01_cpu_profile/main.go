// FILE: book/part6_production_engineering/chapter86_pprof/examples/01_cpu_profile/main.go
// CHAPTER: 86 — pprof
// TOPIC: CPU profiling — identifying hotspots, flame graphs, and reading
//        pprof output. Self-contained simulation of a CPU-bound workload.
//
// Run:
//   go run ./examples/01_cpu_profile

package main

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CPU-BOUND WORKLOADS (intentionally slow vs optimised)
// ─────────────────────────────────────────────────────────────────────────────

// isPrimeSlow — trial division with no early exit
func isPrimeSlow(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i < n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// isPrimeFast — only check up to sqrt(n), skip even numbers
func isPrimeFast(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	limit := int(math.Sqrt(float64(n)))
	for i := 3; i <= limit; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func countPrimes(limit int, check func(int) bool) int {
	count := 0
	for i := 2; i <= limit; i++ {
		if check(i) {
			count++
		}
	}
	return count
}

// ─────────────────────────────────────────────────────────────────────────────
// STRING PROCESSING (allocation-heavy)
// ─────────────────────────────────────────────────────────────────────────────

func processWordsSlow(words []string) map[string]int {
	result := map[string]int{}
	for _, w := range words {
		lower := strings.ToLower(w)
		trimmed := strings.TrimSpace(lower)
		if len(trimmed) > 0 {
			result[trimmed]++
		}
	}
	return result
}

func processWordsFast(words []string) map[string]int {
	result := make(map[string]int, len(words)/2)
	for _, w := range words {
		w = strings.TrimSpace(strings.ToLower(w))
		if len(w) > 0 {
			result[w]++
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// SORT-HEAVY WORKLOAD
// ─────────────────────────────────────────────────────────────────────────────

func sortEveryTime(batches [][]int) [][]int {
	result := make([][]int, len(batches))
	for i, batch := range batches {
		cp := make([]int, len(batch))
		copy(cp, batch)
		sort.Ints(cp)
		result[i] = cp
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// LIGHTWEIGHT PROFILING TIMER
// ─────────────────────────────────────────────────────────────────────────────

type Timing struct {
	Name     string
	Duration time.Duration
	Result   any
}

func timed(name string, fn func() any) Timing {
	start := time.Now()
	result := fn()
	return Timing{name, time.Since(start), result}
}

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE STACK DUMP (simulated)
// ─────────────────────────────────────────────────────────────────────────────

func captureGoroutineCount() int {
	return runtime.NumGoroutine()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== CPU Profiling ===")
	fmt.Println()

	// ── PRIME COUNTING ────────────────────────────────────────────────────────
	fmt.Println("--- Prime counting: slow vs fast ---")
	limit := 5000

	t1 := timed("isPrimeSlow (trial division)", func() any {
		return countPrimes(limit, isPrimeSlow)
	})
	t2 := timed("isPrimeFast (sqrt limit)", func() any {
		return countPrimes(limit, isPrimeFast)
	})

	fmt.Printf("  %-35s  %8v  primes=%d\n", t1.Name, t1.Duration.Round(time.Microsecond), t1.Result)
	fmt.Printf("  %-35s  %8v  primes=%d\n", t2.Name, t2.Duration.Round(time.Microsecond), t2.Result)
	fmt.Printf("  Speedup: %.1fx\n", float64(t1.Duration)/float64(t2.Duration))

	// ── STRING PROCESSING ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- String processing ---")
	words := make([]string, 10000)
	wordsBase := []string{"the", "fox", "dog", "cat", "bird", "fish", "lion", "bear"}
	for i := range words {
		words[i] = "  " + wordsBase[i%len(wordsBase)] + "  "
	}
	t3 := timed("processWordsSlow", func() any {
		return len(processWordsSlow(words))
	})
	t4 := timed("processWordsFast", func() any {
		return len(processWordsFast(words))
	})
	fmt.Printf("  %-35s  %8v  unique=%d\n", t3.Name, t3.Duration.Round(time.Microsecond), t3.Result)
	fmt.Printf("  %-35s  %8v  unique=%d\n", t4.Name, t4.Duration.Round(time.Microsecond), t4.Result)

	// ── REAL PPROF WORKFLOW ───────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CPU profiling workflow (real Go) ---")
	fmt.Println(`  // Option 1: benchmark-based
  go test -bench=BenchmarkPrimes -cpuprofile=cpu.prof ./...
  go tool pprof -http=:8080 cpu.prof

  // Option 2: runtime/pprof in main
  import "runtime/pprof"

  f, _ := os.Create("cpu.prof")
  pprof.StartCPUProfile(f)
  defer pprof.StopCPUProfile()
  // ... your workload ...

  // Option 3: net/http/pprof endpoint (always-on in production)
  import _ "net/http/pprof"

  go http.ListenAndServe(":6060", nil)
  // curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
  // go tool pprof cpu.prof`)

	// ── READING PPROF OUTPUT ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Reading pprof output ---")
	fmt.Println(`  (pprof) top 10
    Showing nodes accounting for 4.20s, 98.60% of 4.26s total
    Showing top 10 nodes out of 42
        flat  flat%   sum%        cum   cum%
       3.80s 89.20% 89.20%      3.80s 89.20%  main.isPrimeSlow
       0.30s  7.04% 96.24%      0.30s  7.04%  runtime.mallocgc
       ...

  flat  = time spent IN this function
  cum   = time spent IN this function AND all functions it calls

  (pprof) list isPrimeSlow
      3.80s      3.80s  src/main.go:22
      ...
      3.80s      3.80s    for i := 2; i < n; i++ {  // ← hotspot

  (pprof) web     → flame graph in browser (requires graphviz)
  (pprof) svg     → save flame graph as SVG`)

	// ── GOROUTINE COUNT ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  Current goroutine count: %d\n", captureGoroutineCount())
	fmt.Println()
	fmt.Println("--- net/http/pprof endpoints ---")
	fmt.Println(`  /debug/pprof/          — index
  /debug/pprof/goroutine  — current goroutine stacks
  /debug/pprof/heap       — heap allocations
  /debug/pprof/profile    — 30s CPU profile
  /debug/pprof/trace      — execution trace
  /debug/pprof/allocs     — past allocations
  /debug/pprof/block      — goroutine blocking events
  /debug/pprof/mutex      — mutex contention`)
}
