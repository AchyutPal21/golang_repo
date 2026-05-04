// FILE: book/part6_production_engineering/chapter85_benchmarking/examples/02_profiling_benchmarks/main.go
// CHAPTER: 85 — Benchmarking
// TOPIC: Memory profiling, CPU profiling, allocation hotspots, and
//        benchmark-driven optimisation cycle.
//
// Run:
//   go run ./examples/02_profiling_benchmarks

package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// IMPLEMENTATIONS TO COMPARE
// ─────────────────────────────────────────────────────────────────────────────

// --- Word frequency counter ---

// V1: allocates a new map every call, no pre-sizing
func WordFreqV1(text string) map[string]int {
	freq := make(map[string]int)
	for _, word := range strings.Fields(text) {
		freq[strings.ToLower(word)]++
	}
	return freq
}

// V2: pre-size map hint, reuse lower conversion
func WordFreqV2(text string) map[string]int {
	words := strings.Fields(text)
	freq := make(map[string]int, len(words)/2)
	for _, word := range words {
		freq[strings.ToLower(word)]++
	}
	return freq
}

// --- Top-N ---

type WordCount struct {
	Word  string
	Count int
}

// V1: sort all entries
func TopNV1(freq map[string]int, n int) []WordCount {
	all := make([]WordCount, 0, len(freq))
	for w, c := range freq {
		all = append(all, WordCount{w, c})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Count > all[j].Count })
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// V2: partial sort — only maintain a top-N heap (simplified with slice cap)
func TopNV2(freq map[string]int, n int) []WordCount {
	top := make([]WordCount, 0, n+1)
	for w, c := range freq {
		top = append(top, WordCount{w, c})
		if len(top) > n {
			// Remove minimum.
			minIdx := 0
			for i := 1; i < len(top); i++ {
				if top[i].Count < top[minIdx].Count {
					minIdx = i
				}
			}
			top[minIdx] = top[len(top)-1]
			top = top[:len(top)-1]
		}
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Count > top[j].Count })
	return top
}

// --- JSON-like serialiser ---

// V1: fmt.Sprintf (allocates format string each call)
func SerialiseV1(m map[string]int) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%q:%d", k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// V2: strings.Builder (single allocation path)
func SerialiseV2(m map[string]int) string {
	var sb strings.Builder
	sb.WriteByte('{')
	first := true
	for k, v := range m {
		if !first {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%q:%d", k, v)
		first = false
	}
	sb.WriteByte('}')
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// BENCHMARK RUNNER
// ─────────────────────────────────────────────────────────────────────────────

type Result struct {
	Name        string
	NsPerOp     float64
	AllocsPerOp uint64
	BytesPerOp  uint64
}

func (r Result) String() string {
	return fmt.Sprintf("  %-35s  %9.1f ns/op  %5d allocs  %7d B/op",
		r.Name, r.NsPerOp, r.AllocsPerOp, r.BytesPerOp)
}

func bench(name string, fn func()) Result {
	for i := 0; i < 5; i++ {
		fn()
	}
	var mBefore, mAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mBefore)
	n := 0
	start := time.Now()
	for time.Since(start) < 200*time.Millisecond {
		fn()
		n++
	}
	elapsed := time.Since(start)
	runtime.GC()
	runtime.ReadMemStats(&mAfter)
	nsPerOp := float64(elapsed.Nanoseconds()) / float64(n)
	var allocsPerOp, bytesPerOp uint64
	if n > 0 {
		allocsPerOp = (mAfter.Mallocs - mBefore.Mallocs) / uint64(n)
		bytesPerOp = (mAfter.TotalAlloc - mBefore.TotalAlloc) / uint64(n)
	}
	return Result{name, nsPerOp, allocsPerOp, bytesPerOp}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Profiling-Driven Benchmarks ===")
	fmt.Println()

	corpus := strings.Repeat("the quick brown fox jumps over the lazy dog ", 500)

	// ── WORD FREQUENCY ────────────────────────────────────────────────────────
	fmt.Println("--- Word frequency: V1 vs V2 ---")
	hdr := fmt.Sprintf("  %-35s  %9s  %12s  %10s", "Name", "ns/op", "allocs/op", "B/op")
	fmt.Println(hdr)
	fmt.Println(bench("WordFreqV1 (no hint)", func() { WordFreqV1(corpus) }))
	fmt.Println(bench("WordFreqV2 (pre-sized)", func() { WordFreqV2(corpus) }))

	freq := WordFreqV2(corpus)

	// ── TOP-N ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- TopN (sort-all vs partial) ---")
	fmt.Println(hdr)
	fmt.Println(bench("TopNV1 (full sort)", func() { TopNV1(freq, 5) }))
	fmt.Println(bench("TopNV2 (partial)", func() { TopNV2(freq, 5) }))

	// ── SERIALISE ─────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Serialise: Sprintf vs Builder ---")
	small := map[string]int{"the": 100, "fox": 50, "dog": 25}
	fmt.Println(hdr)
	fmt.Println(bench("SerialiseV1 (Sprintf parts)", func() { SerialiseV1(small) }))
	fmt.Println(bench("SerialiseV2 (Builder)", func() { SerialiseV2(small) }))

	// ── CPU PROFILING REFERENCE ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CPU and memory profiling reference ---")
	fmt.Println(`  // CPU profile:
  go test -bench=BenchmarkWordFreq -cpuprofile=cpu.prof ./...
  go tool pprof -http=:8080 cpu.prof
  // or: go tool pprof cpu.prof → (pprof) top 10

  // Memory profile:
  go test -bench=BenchmarkWordFreq -memprofile=mem.prof ./...
  go tool pprof -alloc_objects mem.prof   // allocation count
  go tool pprof -alloc_space   mem.prof   // bytes allocated

  // Key pprof commands:
  //   top        — top N functions by CPU/alloc
  //   list Foo   — annotated source for function Foo
  //   web        — flame graph in browser (requires Graphviz)
  //   peek Foo   — callers and callees of Foo

  // Escape analysis (which variables go to heap):
  go build -gcflags='-m -m' ./...  // -m=1 brief, -m=2 verbose
  // Look for: "escapes to heap" — those are allocations you can eliminate`)

	// ── OPTIMISATION CHECKLIST ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Optimisation checklist ---")
	fmt.Println(`  1. Measure first — never guess. Profile before optimising.
  2. Reduce allocations — heap allocations are the #1 Go perf cost.
     - Pre-size slices/maps: make([]T, 0, expectedLen)
     - Reuse buffers with sync.Pool
     - Avoid interface boxing in hot paths
  3. Avoid copying large structs — pass pointers or use slices.
  4. Use strings.Builder for string concatenation in loops.
  5. Use []byte instead of string when you don't need immutability.
  6. Cache expensive computations (but measure the cache overhead too).
  7. Prefer table-lookup over repeated conditional logic in tight loops.
  8. Profile again to verify improvement didn't regress another metric.`)
}
