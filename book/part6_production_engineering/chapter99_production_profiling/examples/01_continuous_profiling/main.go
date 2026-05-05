// FILE: book/part6_production_engineering/chapter99_production_profiling/examples/01_continuous_profiling/main.go
// CHAPTER: 99 — Production Profiling
// TOPIC: Always-on pprof setup, overhead budget, collection scheduling,
//        mutex/block profiling, and self-profiling measurement.
//
// Run:
//   go run ./book/part6_production_engineering/chapter99_production_profiling/examples/01_continuous_profiling

package main

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PPROF ENDPOINT INVENTORY
// ─────────────────────────────────────────────────────────────────────────────

type PprofEndpoint struct {
	Path      string
	What      string
	Overhead  string
	NeedsEnable string
}

var pprofEndpoints = []PprofEndpoint{
	{"/debug/pprof/profile?seconds=30", "30s CPU profile", "~5% CPU (while active)", ""},
	{"/debug/pprof/heap", "Heap snapshot (live objects)", "Negligible", ""},
	{"/debug/pprof/allocs", "All allocations since start", "Negligible", ""},
	{"/debug/pprof/goroutine?debug=2", "All goroutine stacks", "Momentary STW", ""},
	{"/debug/pprof/mutex", "Mutex contention", "~1% CPU", "SetMutexProfileFraction(5)"},
	{"/debug/pprof/block", "Blocking events", "~1% CPU", "SetBlockProfileRate(1000)"},
	{"/debug/pprof/threadcreate", "OS thread creation", "Negligible", ""},
	{"/debug/pprof/cmdline", "Process command line", "None", ""},
	{"/debug/pprof/", "Index of all profiles", "None", ""},
}

func printEndpoints() {
	fmt.Printf("  %-45s  %-35s  %-20s  %s\n", "Endpoint", "Captures", "Overhead", "Enable")
	fmt.Printf("  %s\n", strings.Repeat("-", 120))
	for _, e := range pprofEndpoints {
		enable := e.NeedsEnable
		if enable == "" {
			enable = "(always on)"
		}
		fmt.Printf("  %-45s  %-35s  %-20s  %s\n", e.Path, e.What, e.Overhead, enable)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// OVERHEAD CALCULATOR
// ─────────────────────────────────────────────────────────────────────────────

type OverheadBudget struct {
	WorkloadOpsPerSec  int
	ProfileSeconds     int
	FrequencyMinutes   int
	CPUOverheadPct     float64 // fraction of ops lost per profile window
}

func (b OverheadBudget) OpsLostPerWindow() float64 {
	return float64(b.WorkloadOpsPerSec) * float64(b.ProfileSeconds) * b.CPUOverheadPct / 100
}

func (b OverheadBudget) WindowsPerHour() float64 {
	return 60.0 / float64(b.FrequencyMinutes)
}

func (b OverheadBudget) HourlyOpsLost() float64 {
	return b.OpsLostPerWindow() * b.WindowsPerHour()
}

func (b OverheadBudget) TotalOpsPerHour() float64 {
	return float64(b.WorkloadOpsPerSec) * 3600
}

func (b OverheadBudget) OverheadPct() float64 {
	return 100 * b.HourlyOpsLost() / b.TotalOpsPerHour()
}

func (b OverheadBudget) Recommendation() string {
	pct := b.OverheadPct()
	switch {
	case pct < 0.1:
		return "SAFE — negligible impact"
	case pct < 0.5:
		return "ACCEPTABLE — within 0.5% budget"
	case pct < 1.0:
		return "MARGINAL — consider reducing frequency"
	default:
		return "TOO HIGH — reduce frequency or profile window"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PROFILE CAPTURE (in-process, no file I/O)
// ─────────────────────────────────────────────────────────────────────────────

type ProfileSnapshot struct {
	Name      string
	Timestamp time.Time
	SizeBytes int
}

func captureHeapProfile() ProfileSnapshot {
	var buf bytes.Buffer
	if err := pprof.WriteHeapProfile(&buf); err != nil {
		return ProfileSnapshot{Name: "heap", Timestamp: time.Now()}
	}
	return ProfileSnapshot{
		Name:      "heap",
		Timestamp: time.Now(),
		SizeBytes: buf.Len(),
	}
}

func captureGoroutineProfile() ProfileSnapshot {
	var buf bytes.Buffer
	p := pprof.Lookup("goroutine")
	if p == nil {
		return ProfileSnapshot{Name: "goroutine", Timestamp: time.Now()}
	}
	p.WriteTo(&buf, 2) //nolint:errcheck
	return ProfileSnapshot{
		Name:      "goroutine",
		Timestamp: time.Now(),
		SizeBytes: buf.Len(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COLLECTION SCHEDULER SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type CollectionSchedule struct {
	ProfileType      string
	FrequencyMinutes int
	RetentionHours   int
}

var defaultSchedule = []CollectionSchedule{
	{"CPU profile (30s)", 5, 24},
	{"Heap snapshot", 1, 72},
	{"Goroutine dump", 1, 24},
	{"Mutex profile", 15, 6},
	{"Block profile", 15, 6},
}

func printSchedule() {
	fmt.Printf("  %-25s  %10s  %12s  %s\n", "Profile type", "Frequency", "Retention", "Profiles/day")
	fmt.Printf("  %s\n", strings.Repeat("-", 65))
	for _, s := range defaultSchedule {
		perDay := 24 * 60 / s.FrequencyMinutes
		fmt.Printf("  %-25s  every %3dm  %8dh  %d\n",
			s.ProfileType, s.FrequencyMinutes, s.RetentionHours, perDay)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SELF-PROFILING: measure cost of capturing profiles
// ─────────────────────────────────────────────────────────────────────────────

func measureProfileCost() {
	// Heap
	start := time.Now()
	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		buf.Reset()
		pprof.WriteHeapProfile(&buf) //nolint:errcheck
	}
	heapCost := time.Since(start) / 10

	// Goroutine
	start = time.Now()
	for i := 0; i < 10; i++ {
		buf.Reset()
		p := pprof.Lookup("goroutine")
		if p != nil {
			p.WriteTo(&buf, 1) //nolint:errcheck
		}
	}
	goroutineCost := time.Since(start) / 10

	fmt.Printf("  Heap profile capture:      %v\n", heapCost.Round(time.Microsecond))
	fmt.Printf("  Goroutine profile capture: %v\n", goroutineCost.Round(time.Microsecond))
	fmt.Printf("  (These are one-shot costs — not sustained overhead)\n")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 99: Continuous Profiling ===")
	fmt.Println()

	// Enable mutex and block profiling
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(1000)

	// ── ENDPOINT INVENTORY ────────────────────────────────────────────────────
	fmt.Println("--- pprof endpoint inventory ---")
	printEndpoints()
	fmt.Println()

	// ── OVERHEAD BUDGET ───────────────────────────────────────────────────────
	fmt.Println("--- Overhead budget calculator ---")
	scenarios := []OverheadBudget{
		{10_000, 30, 5, 5.0},   // high traffic, frequent
		{10_000, 30, 60, 5.0},  // high traffic, infrequent
		{1_000, 30, 5, 5.0},    // lower traffic, frequent
		{100_000, 30, 5, 5.0},  // very high traffic
	}
	fmt.Printf("  %-12s  %-8s  %-10s  %-15s  %-8s  %s\n",
		"ops/sec", "prof(s)", "freq(min)", "ops_lost/hr", "overhead%", "verdict")
	fmt.Printf("  %s\n", strings.Repeat("-", 80))
	for _, s := range scenarios {
		fmt.Printf("  %-12d  %-8d  %-10d  %-15.0f  %-8.3f  %s\n",
			s.WorkloadOpsPerSec, s.ProfileSeconds, s.FrequencyMinutes,
			s.HourlyOpsLost(), s.OverheadPct(), s.Recommendation())
	}
	fmt.Println()

	// ── COLLECTION SCHEDULE ───────────────────────────────────────────────────
	fmt.Println("--- Recommended collection schedule ---")
	printSchedule()
	fmt.Println()

	// ── PROFILE CAPTURE ───────────────────────────────────────────────────────
	fmt.Println("--- Capturing profiles (in-process) ---")
	heap := captureHeapProfile()
	goroutine := captureGoroutineProfile()
	fmt.Printf("  Heap snapshot:     %d bytes at %s\n",
		heap.SizeBytes, heap.Timestamp.Format("15:04:05"))
	fmt.Printf("  Goroutine dump:    %d bytes at %s (%d goroutines)\n",
		goroutine.SizeBytes, goroutine.Timestamp.Format("15:04:05"), runtime.NumGoroutine())
	fmt.Println()

	// ── SELF-PROFILING COST ───────────────────────────────────────────────────
	fmt.Println("--- Self-profiling: cost of capturing a profile ---")
	measureProfileCost()
	fmt.Println()

	// ── SETUP REFERENCE ──────────────────────────────────────────────────────
	fmt.Println("--- Always-on pprof setup ---")
	fmt.Println(`  // main.go — add these lines before starting your server
  import _ "net/http/pprof"

  func main() {
      // Internal-only: bind to loopback or restrict at network level
      go http.ListenAndServe("127.0.0.1:6060", nil)

      // Enable mutex + block profiling (small overhead, big insight)
      runtime.SetMutexProfileFraction(5)
      runtime.SetBlockProfileRate(1000)

      // ... rest of main
  }

  // NEVER expose :6060 on a public IP — it leaks call graphs and memory layout.
  // Use an internal load balancer rule, VPN, or ssh tunnel:
  //   ssh -L 6060:localhost:6060 prod-host-1
  //   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30`)
}
