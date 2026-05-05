// FILE: book/part6_production_engineering/chapter98_incidents/examples/01_goroutine_dumps/main.go
// CHAPTER: 98 — Incident Management & Debugging
// TOPIC: Goroutine dump techniques, leak detection, and deadlock identification.
//
// Run:
//   go run ./book/part6_production_engineering/chapter98_incidents/examples/01_goroutine_dumps

package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE STACK CAPTURE
// ─────────────────────────────────────────────────────────────────────────────

func captureStack(all bool) string {
	buf := make([]byte, 64*1024)
	n := runtime.Stack(buf, all)
	return string(buf[:n])
}

func goroutineCount() int { return runtime.NumGoroutine() }

// ─────────────────────────────────────────────────────────────────────────────
// LEAK DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

type LeakDetector struct {
	baseline int
}

func (ld *LeakDetector) Before() {
	runtime.GC() // Ensure finalizers have run
	ld.baseline = goroutineCount()
}

func (ld *LeakDetector) Check(label string, settleMs int) {
	time.Sleep(time.Duration(settleMs) * time.Millisecond)
	after := goroutineCount()
	delta := after - ld.baseline
	if delta > 0 {
		fmt.Printf("  LEAK detected [%s]: goroutines %d → %d (+%d)\n",
			label, ld.baseline, after, delta)
	} else {
		fmt.Printf("  OK [%s]: goroutines %d → %d (no leak)\n",
			label, ld.baseline, after)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK EXAMPLES
// ─────────────────────────────────────────────────────────────────────────────

// leaky starts goroutines that block forever on an unread channel.
func leaky(n int) {
	ch := make(chan struct{}) // never closed
	for i := 0; i < n; i++ {
		go func() { <-ch }()
	}
}

// fixed starts goroutines that exit when done channel is closed.
func fixed(n int) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
			}
		}()
	}
	close(done) // signals all goroutines
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// STACK TRACE ANALYSIS
// ─────────────────────────────────────────────────────────────────────────────

type GoroutineInfo struct {
	ID    string
	State string
	Lines []string
}

func parseGoroutineStacks(dump string) []GoroutineInfo {
	var infos []GoroutineInfo
	sections := strings.Split(dump, "\n\n")
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if !strings.HasPrefix(section, "goroutine ") {
			continue
		}
		lines := strings.Split(section, "\n")
		if len(lines) == 0 {
			continue
		}
		// First line: "goroutine 1 [running]:"
		header := lines[0]
		state := ""
		if start := strings.Index(header, "["); start >= 0 {
			if end := strings.Index(header, "]"); end > start {
				state = header[start+1 : end]
			}
		}
		id := ""
		if parts := strings.Fields(header); len(parts) >= 2 {
			id = parts[1]
		}
		infos = append(infos, GoroutineInfo{
			ID:    id,
			State: state,
			Lines: lines[1:],
		})
	}
	return infos
}

// ─────────────────────────────────────────────────────────────────────────────
// PPROF REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const pprofRef = `  Always-on pprof endpoint (add to main):
    import _ "net/http/pprof"
    go http.ListenAndServe(":6060", nil)

  Goroutine dump:
    curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -200

  Heap snapshot:
    curl http://localhost:6060/debug/pprof/heap > heap.prof
    go tool pprof heap.prof

  CPU profile (30 seconds):
    curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
    go tool pprof -http=:8080 cpu.prof

  Send SIGQUIT to dump all stacks to stderr:
    kill -QUIT $(pgrep myapp)   # Linux
    kill -SIGQUIT $(pgrep myapp) # macOS`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 98: Goroutine Dumps & Leak Detection ===")
	fmt.Println()

	// ── CURRENT GOROUTINES ────────────────────────────────────────────────────
	fmt.Println("--- Current goroutine count ---")
	fmt.Printf("  Goroutines at startup: %d\n\n", goroutineCount())

	// ── LEAK DETECTION ───────────────────────────────────────────────────────
	fmt.Println("--- Leak detection ---")
	ld := &LeakDetector{}

	ld.Before()
	leaky(5) // these will never be collected
	ld.Check("leaky goroutines (expected to detect)", 10)

	ld.Before()
	fixed(5)
	ld.Check("fixed goroutines (closed channel)", 100)
	fmt.Println()

	// ── STACK TRACE CAPTURE ───────────────────────────────────────────────────
	fmt.Println("--- Current goroutine stack (this goroutine) ---")
	stack := captureStack(false)
	lines := strings.Split(stack, "\n")
	for _, l := range lines[:min(10, len(lines))] {
		fmt.Printf("  %s\n", l)
	}
	if len(lines) > 10 {
		fmt.Printf("  ... (%d more lines)\n", len(lines)-10)
	}
	fmt.Println()

	// ── STACK PARSING ─────────────────────────────────────────────────────────
	fmt.Println("--- Parse goroutine states ---")
	allStacks := captureStack(true)
	goroutines := parseGoroutineStacks(allStacks)
	states := make(map[string]int)
	for _, g := range goroutines {
		states[g.State]++
	}
	fmt.Printf("  Total goroutines: %d\n", len(goroutines))
	for state, count := range states {
		fmt.Printf("  State %-20s: %d\n", "'"+state+"'", count)
	}
	fmt.Println()

	// ── PPROF REFERENCE ───────────────────────────────────────────────────────
	fmt.Println("--- pprof endpoint reference ---")
	fmt.Println(pprofRef)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
