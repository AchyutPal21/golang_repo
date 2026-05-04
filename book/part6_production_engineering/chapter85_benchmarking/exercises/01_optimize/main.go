// FILE: book/part6_production_engineering/chapter85_benchmarking/exercises/01_optimize/main.go
// CHAPTER: 85 — Benchmarking
// TOPIC: Benchmark-driven optimisation — identify slow path, optimise, verify.
//        Four functions are optimised: JSON builder, duplicate filter,
//        CSV parser, and in-memory cache.
//
// Run:
//   go run ./exercises/01_optimize

package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. JSON BUILDER
// ─────────────────────────────────────────────────────────────────────────────

// Slow: allocates a new string for each key-value pair
func BuildJSONSlow(fields map[string]string) string {
	result := "{"
	first := true
	for k, v := range fields {
		if !first {
			result += ","
		}
		result += `"` + k + `":"` + v + `"`
		first = false
	}
	return result + "}"
}

// Fast: strings.Builder, pre-grow estimate
func BuildJSONFast(fields map[string]string) string {
	var sb strings.Builder
	sb.Grow(len(fields) * 32)
	sb.WriteByte('{')
	first := true
	for k, v := range fields {
		if !first {
			sb.WriteByte(',')
		}
		sb.WriteByte('"')
		sb.WriteString(k)
		sb.WriteString(`":"`)
		sb.WriteString(v)
		sb.WriteByte('"')
		first = false
	}
	sb.WriteByte('}')
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. DUPLICATE FILTER
// ─────────────────────────────────────────────────────────────────────────────

// Slow: O(n²) nested loop
func DeduplicateSlow(items []string) []string {
	var result []string
	for _, item := range items {
		found := false
		for _, r := range result {
			if r == item {
				found = true
				break
			}
		}
		if !found {
			result = append(result, item)
		}
	}
	return result
}

// Fast: O(n) with map
func DeduplicateFast(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. CSV PARSER
// ─────────────────────────────────────────────────────────────────────────────

// Slow: strings.Split per line, string conversions
func ParseCSVSlow(csv string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(csv, "\n") {
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, ","))
	}
	return rows
}

// Fast: use SplitN, avoid empty line allocation
func ParseCSVFast(csv string) [][]string {
	lines := strings.Split(csv, "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		rows = append(rows, strings.SplitN(line, ",", -1))
	}
	return rows
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. CACHE (locked map vs sync.Map)
// ─────────────────────────────────────────────────────────────────────────────

type LockedCache struct {
	mu    sync.Mutex
	store map[string]string
}

func NewLockedCache() *LockedCache {
	return &LockedCache{store: make(map[string]string)}
}

func (c *LockedCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.store[key]
	return v, ok
}

func (c *LockedCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
}

type RWCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewRWCache() *RWCache {
	return &RWCache{store: make(map[string]string)}
}

func (c *RWCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[key]
	return v, ok
}

func (c *RWCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
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
	speedup := ""
	return fmt.Sprintf("  %-40s  %9.1f ns/op  %5d allocs  %7d B/op%s",
		r.Name, r.NsPerOp, r.AllocsPerOp, r.BytesPerOp, speedup)
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

func speedup(slow, fast Result) string {
	if fast.NsPerOp == 0 {
		return "N/A"
	}
	ratio := slow.NsPerOp / fast.NsPerOp
	return fmt.Sprintf("%.1fx faster", ratio)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Benchmark-Driven Optimisation ===")
	fmt.Println()
	hdr := fmt.Sprintf("  %-40s  %9s  %12s  %10s", "Name", "ns/op", "allocs/op", "B/op")

	// ── 1. JSON BUILDER ───────────────────────────────────────────────────────
	fmt.Println("--- 1. JSON builder ---")
	fields := make(map[string]string, 20)
	for i := 0; i < 20; i++ {
		fields["key"+strconv.Itoa(i)] = "value" + strconv.Itoa(i)
	}
	fmt.Println(hdr)
	slow1 := bench("BuildJSONSlow (+concat)", func() { BuildJSONSlow(fields) })
	fast1 := bench("BuildJSONFast (Builder)", func() { BuildJSONFast(fields) })
	fmt.Println(slow1)
	fmt.Println(fast1)
	fmt.Printf("  Speedup: %s\n", speedup(slow1, fast1))

	// ── 2. DEDUP ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- 2. Duplicate filter ---")
	items := make([]string, 200)
	for i := range items {
		items[i] = "item" + strconv.Itoa(i%50) // 50% duplicates
	}
	fmt.Println(hdr)
	slow2 := bench("DeduplicateSlow (O(n²))", func() { DeduplicateSlow(items) })
	fast2 := bench("DeduplicateFast (map)", func() { DeduplicateFast(items) })
	fmt.Println(slow2)
	fmt.Println(fast2)
	fmt.Printf("  Speedup: %s\n", speedup(slow2, fast2))

	// ── 3. CSV PARSER ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- 3. CSV parser ---")
	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, fmt.Sprintf("col1_%d,col2_%d,col3_%d", i, i*2, i*3))
	}
	csv := strings.Join(lines, "\n")
	fmt.Println(hdr)
	slow3 := bench("ParseCSVSlow", func() { ParseCSVSlow(csv) })
	fast3 := bench("ParseCSVFast (pre-alloc)", func() { ParseCSVFast(csv) })
	fmt.Println(slow3)
	fmt.Println(fast3)
	fmt.Printf("  Speedup: %s\n", speedup(slow3, fast3))

	// ── 4. CACHE READS ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- 4. Cache: Mutex vs RWMutex (read-heavy) ---")
	lc := NewLockedCache()
	rc := NewRWCache()
	for i := 0; i < 100; i++ {
		k := "k" + strconv.Itoa(i)
		lc.Set(k, "v")
		rc.Set(k, "v")
	}
	fmt.Println(hdr)
	slow4 := bench("LockedCache.Get (Mutex)", func() { lc.Get("k50") })
	fast4 := bench("RWCache.Get (RWMutex)", func() { rc.Get("k50") })
	fmt.Println(slow4)
	fmt.Println(fast4)
	fmt.Printf("  Speedup: %s\n", speedup(slow4, fast4))

	// ── SUMMARY ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Lessons ---")
	fmt.Println(`  JSON builder:  + concat creates O(n) strings; Builder does one alloc.
  Dedup:         Nested loop is O(n²); map lookup is O(1) per item.
  CSV parser:    Pre-allocating the result slice saves repeated growth copies.
  Cache reads:   sync.Mutex blocks all readers; RWMutex allows concurrent reads.`)
}
