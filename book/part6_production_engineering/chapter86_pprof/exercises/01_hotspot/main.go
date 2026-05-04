// FILE: book/part6_production_engineering/chapter86_pprof/exercises/01_hotspot/main.go
// CHAPTER: 86 — pprof
// TOPIC: Find and fix hotspots — CPU and memory profiling exercise with
//        a report-generation pipeline that has three identifiable bottlenecks.
//
// Run:
//   go run ./exercises/01_hotspot

package main

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN: sales report pipeline
// ─────────────────────────────────────────────────────────────────────────────

type Sale struct {
	ProductID string
	Amount    int
	Region    string
}

type Report struct {
	TotalRevenue  int
	TopProducts   []ProductStat
	RegionRevenue map[string]int
}

type ProductStat struct {
	ProductID string
	Revenue   int
}

// ─────────────────────────────────────────────────────────────────────────────
// SLOW PIPELINE — three bottlenecks for the student to find
// ─────────────────────────────────────────────────────────────────────────────

// Bottleneck 1: O(n²) dedup
func uniqueProductsSlow(sales []Sale) []string {
	var unique []string
	for _, s := range sales {
		found := false
		for _, u := range unique {
			if u == s.ProductID {
				found = true
				break
			}
		}
		if !found {
			unique = append(unique, s.ProductID)
		}
	}
	return unique
}

// Bottleneck 2: string concatenation in report formatter
func formatReportSlow(r *Report) string {
	result := "=== Sales Report ===\n"
	result += fmt.Sprintf("Total Revenue: %d\n\n", r.TotalRevenue)
	result += "Top Products:\n"
	for _, p := range r.TopProducts {
		result += fmt.Sprintf("  %s: %d\n", p.ProductID, p.Revenue)
	}
	result += "\nRevenue by Region:\n"
	for region, rev := range r.RegionRevenue {
		result += fmt.Sprintf("  %s: %d\n", region, rev)
	}
	return result
}

// Bottleneck 3: re-sorting on every GetTopN call
func getTopNSlow(productRevenue map[string]int, n int) []ProductStat {
	stats := make([]ProductStat, 0, len(productRevenue))
	for id, rev := range productRevenue {
		stats = append(stats, ProductStat{id, rev})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Revenue > stats[j].Revenue
	})
	if n > len(stats) {
		n = len(stats)
	}
	return stats[:n]
}

func generateReportSlow(sales []Sale) *Report {
	// Aggregate revenue by product and region.
	productRevenue := make(map[string]int)
	regionRevenue := make(map[string]int)
	total := 0
	for _, s := range sales {
		productRevenue[s.ProductID] += s.Amount
		regionRevenue[s.Region] += s.Amount
		total += s.Amount
	}
	_ = uniqueProductsSlow(sales) // called but result unused — still costs O(n²)
	return &Report{
		TotalRevenue:  total,
		TopProducts:   getTopNSlow(productRevenue, 5),
		RegionRevenue: regionRevenue,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FAST PIPELINE — bottlenecks fixed
// ─────────────────────────────────────────────────────────────────────────────

// Fix 1: O(n) map dedup
func uniqueProductsFast(sales []Sale) []string {
	seen := make(map[string]struct{}, len(sales)/4)
	for _, s := range sales {
		seen[s.ProductID] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result
}

// Fix 2: strings.Builder
func formatReportFast(r *Report) string {
	var sb strings.Builder
	sb.Grow(512)
	sb.WriteString("=== Sales Report ===\n")
	fmt.Fprintf(&sb, "Total Revenue: %d\n\nTop Products:\n", r.TotalRevenue)
	for _, p := range r.TopProducts {
		fmt.Fprintf(&sb, "  %s: %d\n", p.ProductID, p.Revenue)
	}
	sb.WriteString("\nRevenue by Region:\n")
	for region, rev := range r.RegionRevenue {
		fmt.Fprintf(&sb, "  %s: %d\n", region, rev)
	}
	return sb.String()
}

// Fix 3: sort once, cache result
type ReportCache struct {
	mu    sync.RWMutex
	cache map[string]*Report
}

func NewReportCache() *ReportCache {
	return &ReportCache{cache: make(map[string]*Report)}
}

func (rc *ReportCache) Get(key string) (*Report, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	r, ok := rc.cache[key]
	return r, ok
}

func (rc *ReportCache) Set(key string, r *Report) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache[key] = r
}

func generateReportFast(sales []Sale) *Report {
	productRevenue := make(map[string]int, 50)
	regionRevenue := make(map[string]int, 10)
	total := 0
	for _, s := range sales {
		productRevenue[s.ProductID] += s.Amount
		regionRevenue[s.Region] += s.Amount
		total += s.Amount
	}
	_ = uniqueProductsFast(sales)
	return &Report{
		TotalRevenue:  total,
		TopProducts:   getTopNSlow(productRevenue, 5), // same sort, but dedup is fast
		RegionRevenue: regionRevenue,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MEMORY MEASUREMENT
// ─────────────────────────────────────────────────────────────────────────────

func measureAllocs(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.GC()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// ─────────────────────────────────────────────────────────────────────────────
// GENERATE TEST DATA
// ─────────────────────────────────────────────────────────────────────────────

func generateSales(n int) []Sale {
	products := []string{"prod-A", "prod-B", "prod-C", "prod-D", "prod-E",
		"prod-F", "prod-G", "prod-H", "prod-I", "prod-J"}
	regions := []string{"north", "south", "east", "west", "central"}
	sales := make([]Sale, n)
	for i := range sales {
		sales[i] = Sale{
			ProductID: products[i%len(products)],
			Amount:    100 + (i*7)%500,
			Region:    regions[i%len(regions)],
		}
	}
	return sales
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Hotspot Identification Exercise ===")
	fmt.Println()

	sales := generateSales(2000)

	// ── REPORT GENERATION ─────────────────────────────────────────────────────
	fmt.Println("--- Report generation: slow vs fast ---")

	slowAllocs := measureAllocs(func() {
		for i := 0; i < 10; i++ {
			generateReportSlow(sales)
		}
	})
	fastAllocs := measureAllocs(func() {
		for i := 0; i < 10; i++ {
			generateReportFast(sales)
		}
	})

	start := time.Now()
	for i := 0; i < 10; i++ {
		generateReportSlow(sales)
	}
	slowDur := time.Since(start)

	start = time.Now()
	for i := 0; i < 10; i++ {
		generateReportFast(sales)
	}
	fastDur := time.Since(start)

	fmt.Printf("  Slow: %v  allocs=%dKB\n", slowDur.Round(time.Microsecond), slowAllocs/1024)
	fmt.Printf("  Fast: %v  allocs=%dKB\n", fastDur.Round(time.Microsecond), fastAllocs/1024)
	fmt.Printf("  Speedup: %.1fx\n", float64(slowDur)/math.Max(float64(fastDur), 1))

	// ── FORMAT COMPARISON ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Report formatting: + concat vs Builder ---")

	report := generateReportFast(sales)

	fmtSlow := measureAllocs(func() {
		for i := 0; i < 100; i++ {
			formatReportSlow(report)
		}
	})
	fmtFast := measureAllocs(func() {
		for i := 0; i < 100; i++ {
			formatReportFast(report)
		}
	})

	fmt.Printf("  formatReportSlow (100 calls): allocs=%dKB\n", fmtSlow/1024)
	fmt.Printf("  formatReportFast (100 calls): allocs=%dKB\n", fmtFast/1024)

	// ── CACHE IMPACT ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Caching reports ---")
	cache := NewReportCache()

	start = time.Now()
	for i := 0; i < 100; i++ {
		key := "report-daily"
		if _, ok := cache.Get(key); !ok {
			r := generateReportFast(sales)
			cache.Set(key, r)
		}
	}
	cached := time.Since(start)
	fmt.Printf("  100 calls with cache: %v (1 compute + 99 cache hits)\n", cached.Round(time.Microsecond))

	// ── BOTTLENECK SUMMARY ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Three bottlenecks identified ---")
	fmt.Println(`  1. uniqueProductsSlow: O(n²) nested scan → O(n) map lookup
     Impact: grows quadratically with number of sales records.

  2. formatReportSlow: string + concat in loop → strings.Builder
     Impact: creates a new string per iteration; Builder reuses buffer.

  3. No caching: report regenerated on every request → cache with RWMutex
     Impact: compute once per time window, serve many readers concurrently.

  How to find these with pprof:
     go test -bench=BenchmarkGenerateReport -cpuprofile=cpu.prof ./...
     (pprof) top 10          → uniqueProductsSlow appears at top
     (pprof) list formatReport  → line with + shows high flat time
     go test -bench=. -memprofile=mem.prof ./...
     (pprof) alloc_space     → formatReport has highest alloc_space`)
}
