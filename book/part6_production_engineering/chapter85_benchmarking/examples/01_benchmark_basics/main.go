// FILE: book/part6_production_engineering/chapter85_benchmarking/examples/01_benchmark_basics/main.go
// CHAPTER: 85 — Benchmarking
// TOPIC: Writing benchmarks, measuring ns/op and allocs/op, comparing
//        implementations, and reading benchstat output.
//        (Self-contained benchmark runner — no testing.B required.)
//
// Run:
//   go run ./examples/01_benchmark_basics

package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// FUNCTIONS TO BENCHMARK
// ─────────────────────────────────────────────────────────────────────────────

// ConcatPlus — naive string concatenation with +
func ConcatPlus(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}

// ConcatBuilder — efficient with strings.Builder
func ConcatBuilder(parts []string) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// ConcatJoin — strings.Join (single allocation)
func ConcatJoin(parts []string) string {
	return strings.Join(parts, "")
}

// Contains — linear scan
func ContainsLinear(haystack []int, needle int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// ContainsMap — O(1) lookup via pre-built map
func ContainsMap(index map[int]struct{}, needle int) bool {
	_, ok := index[needle]
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// MINIMAL BENCHMARK RUNNER (simulates testing.B)
// ─────────────────────────────────────────────────────────────────────────────

type BenchResult struct {
	Name      string
	N         int
	NsPerOp   float64
	AllocsPerOp uint64
	BytesPerOp  uint64
}

func (r BenchResult) String() string {
	return fmt.Sprintf("  %-40s  %8d  %10.1f ns/op  %4d allocs/op  %6d B/op",
		r.Name, r.N, r.NsPerOp, r.AllocsPerOp, r.BytesPerOp)
}

// bench runs fn for at least minDuration, returns stats.
func bench(name string, fn func()) BenchResult {
	// Warmup.
	for i := 0; i < 3; i++ {
		fn()
	}

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	n := 0
	start := time.Now()
	minDuration := 200 * time.Millisecond
	for time.Since(start) < minDuration {
		fn()
		n++
	}
	elapsed := time.Since(start)

	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	nsPerOp := float64(elapsed.Nanoseconds()) / float64(n)
	allocsPerOp := uint64(0)
	bytesPerOp := uint64(0)
	if n > 0 {
		allocsPerOp = (memAfter.Mallocs - memBefore.Mallocs) / uint64(n)
		bytesPerOp = (memAfter.TotalAlloc - memBefore.TotalAlloc) / uint64(n)
	}

	return BenchResult{name, n, nsPerOp, allocsPerOp, bytesPerOp}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Benchmarking Basics ===")
	fmt.Println()

	// ── STRING CONCATENATION ──────────────────────────────────────────────────
	fmt.Println("--- String concatenation (100 parts) ---")
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = fmt.Sprintf("part%d", i)
	}

	results := []BenchResult{
		bench("ConcatPlus", func() { ConcatPlus(parts) }),
		bench("ConcatBuilder", func() { ConcatBuilder(parts) }),
		bench("ConcatJoin", func() { ConcatJoin(parts) }),
	}

	fmt.Printf("  %-40s  %8s  %10s  %14s  %10s\n", "Name", "N", "ns/op", "allocs/op", "B/op")
	for _, r := range results {
		fmt.Println(r)
	}

	fmt.Println()
	fmt.Println("  ConcatPlus creates a new string on every iteration → O(n²) allocations.")
	fmt.Println("  ConcatBuilder and Join pre-allocate the buffer → far fewer allocs.")

	// ── MAP vs LINEAR ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Contains: linear scan vs map lookup (10k elements) ---")

	size := 10000
	slice := make([]int, size)
	index := make(map[int]struct{}, size)
	for i := 0; i < size; i++ {
		slice[i] = i
		index[i] = struct{}{}
	}
	needle := size / 2

	r2 := []BenchResult{
		bench("ContainsLinear", func() { ContainsLinear(slice, needle) }),
		bench("ContainsMap", func() { ContainsMap(index, needle) }),
	}
	fmt.Printf("  %-40s  %8s  %10s  %14s  %10s\n", "Name", "N", "ns/op", "allocs/op", "B/op")
	for _, r := range r2 {
		fmt.Println(r)
	}

	// ── REAL BENCHMARK REFERENCE ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Real Go benchmark pattern ---")
	ref := `  // In a _test.go file:
  func BenchmarkConcatBuilder(b *testing.B) {
      parts := makeParts(100)
      b.ResetTimer()          // don't count setup
      for b.Loop() {          // Go 1.24+; or: i := 0; i < b.N; i++
          ConcatBuilder(parts)
      }
  }

  func BenchmarkConcatBuilder_Allocs(b *testing.B) {
      parts := makeParts(100)
      b.ReportAllocs()        // show allocs/op in output
      b.ResetTimer()
      for b.Loop() {
          ConcatBuilder(parts)
      }
  }

  // Sub-benchmarks for different input sizes:
  func BenchmarkConcat(b *testing.B) {
      for _, n := range []int{10, 100, 1000} {
          b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
              parts := makeParts(n)
              b.ResetTimer()
              for b.Loop() { ConcatBuilder(parts) }
          })
      }
  }

  // Run:
  //   go test -bench=. -benchmem ./...
  //   go test -bench=BenchmarkConcat -benchmem -count=5 | tee bench.txt
  //   benchstat bench.txt   # statistical comparison`
	fmt.Println(ref)

	// ── BENCHSTAT WORKFLOW ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- benchstat comparison workflow ---")
	benchstatRef := `  # Baseline:
  git stash
  go test -bench=. -benchmem -count=5 ./... > old.txt

  # After optimization:
  git stash pop
  go test -bench=. -benchmem -count=5 ./... > new.txt

  # Compare:
  benchstat old.txt new.txt

  # Output (p<0.05 means statistically significant):
  #   ConcatBuilder-8   523ns +/- 2%   312ns +/- 1%   -40.3%
  #
  # Use -count=10 for tighter confidence intervals.`
	fmt.Println(benchstatRef)
}
