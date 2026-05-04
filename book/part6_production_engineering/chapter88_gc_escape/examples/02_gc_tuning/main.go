// FILE: book/part6_production_engineering/chapter88_gc_escape/examples/02_gc_tuning/main.go
// CHAPTER: 88 — GC & Escape Analysis
// TOPIC: GC tuning — GOGC, GOMEMLIMIT, GC pause measurement, finalizers,
//        and runtime.GC() usage patterns. Pure in-process simulation.
//
// Run:
//   go run ./part6_production_engineering/chapter88_gc_escape/examples/02_gc_tuning/

package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// GC BASICS
// ─────────────────────────────────────────────────────────────────────────────

// gcStats captures a snapshot of GC counters.
type gcStats struct {
	numGC      uint32
	pauseTotal time.Duration
	heapAlloc  uint64
	heapSys    uint64
}

func readGCStats() gcStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return gcStats{
		numGC:      m.NumGC,
		pauseTotal: time.Duration(m.PauseTotalNs),
		heapAlloc:  m.HeapAlloc,
		heapSys:    m.HeapSys,
	}
}

func (a gcStats) delta(b gcStats) gcStats {
	return gcStats{
		numGC:      b.numGC - a.numGC,
		pauseTotal: b.pauseTotal - a.pauseTotal,
		heapAlloc:  b.heapAlloc,
		heapSys:    b.heapSys,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// WORKLOAD GENERATORS
// ─────────────────────────────────────────────────────────────────────────────

// highAlloc generates many short-lived heap objects.
func highAlloc(n int) {
	sink := make([]*[]byte, 0, 16)
	for i := 0; i < n; i++ {
		b := make([]byte, 1024) // 1 KB per allocation
		b[0] = byte(i)
		if i%100 == 0 {
			sink = append(sink, &b) // keep some alive
		}
	}
	_ = sink
}

// lowAlloc reuses a single buffer (pool-like).
func lowAlloc(n int) {
	buf := make([]byte, 1024)
	for i := 0; i < n; i++ {
		buf[0] = byte(i) // reuse — no allocation per iteration
	}
	_ = buf
}

// ─────────────────────────────────────────────────────────────────────────────
// GOGC EFFECT SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

// readableGOGC returns the current GOGC setting via debug.SetGCPercent.
// Passing -1 reads without changing.
func readGOGC() int {
	// SetGCPercent returns old value; we restore immediately.
	old := debug.SetGCPercent(-1)
	debug.SetGCPercent(old)
	return old
}

type gogcExperiment struct {
	gcpct    int
	numGC    uint32
	totalNs  time.Duration
}

func runWithGOGC(pct int, iters int) gogcExperiment {
	old := debug.SetGCPercent(pct)
	defer debug.SetGCPercent(old)

	runtime.GC() // clean start
	before := readGCStats()
	highAlloc(iters)
	runtime.GC()
	after := readGCStats()
	d := before.delta(after)
	return gogcExperiment{pct, d.numGC, d.pauseTotal}
}

// ─────────────────────────────────────────────────────────────────────────────
// FINALIZERS
// ─────────────────────────────────────────────────────────────────────────────

type Resource struct {
	Name   string
	closed bool
}

func NewResource(name string) *Resource {
	r := &Resource{Name: name}
	// Finalizer fires when GC collects r — NOT a substitute for Close().
	runtime.SetFinalizer(r, func(res *Resource) {
		if !res.closed {
			// In production: log a warning; never rely on this for correctness.
		}
	})
	return r
}

func (r *Resource) Close() {
	r.closed = true
	runtime.SetFinalizer(r, nil) // remove finalizer after explicit close
}

// ─────────────────────────────────────────────────────────────────────────────
// GC TRACE REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const gcTraceRef = `
GODEBUG=gctrace=1 go run ./...

gc 1 @0.012s 2%: 0.014+1.3+0.004 ms clock, 0.11+0.33/1.2/0+0.036 ms cpu,
   4->4->2 MB, 5 MB goal, 0 MB stacks, 0 MB globals, 8 P

Fields:
  gc N        — GC cycle number
  @Xs         — time since program start
  Y%          — % of CPU used by GC
  A+B+C ms    — wall-clock: STW sweep termination + concurrent mark + STW mark termination
  X->Y->Z MB  — heap before GC -> after GC -> live (retained) MB
  N MB goal   — GOGC target (live*GOGC/100)
  P           — number of Ps (GOMAXPROCS)

Key tuning levers:
  GOGC=100        default — GC when heap doubles
  GOGC=200        less frequent GC, higher memory use
  GOGC=off        disable GC (only safe for batch jobs)
  GOMEMLIMIT=512MiB  hard cap — triggers GC aggressively near limit
  runtime/debug.SetMemoryLimit(512<<20)  — programmatic
`

// ─────────────────────────────────────────────────────────────────────────────
// HEAP PROFILING REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const heapProfileRef = `
Heap profiling workflow:

  import _ "net/http/pprof"
  go http.ListenAndServe(":6060", nil)

  # snapshot
  curl -s http://localhost:6060/debug/pprof/heap > heap.prof
  go tool pprof -http=:8080 heap.prof

  # allocs profile (past allocations, not just live)
  curl -s http://localhost:6060/debug/pprof/allocs > allocs.prof
  go tool pprof -http=:8081 allocs.prof

  pprof commands:
    (pprof) top       — top allocating functions
    (pprof) list Foo  — annotated source for function Foo
    (pprof) inuse_space / alloc_space — switch views
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 88: GC Tuning ===")
	fmt.Println()

	// ── CURRENT GC STATE ──────────────────────────────────────────────────────
	fmt.Println("--- Current GC state ---")
	s := readGCStats()
	fmt.Printf("  NumGC=%d  HeapAlloc=%d KB  HeapSys=%d KB\n",
		s.numGC, s.heapAlloc/1024, s.heapSys/1024)
	fmt.Printf("  GOGC=%d  GOMAXPROCS=%d\n", readGOGC(), runtime.GOMAXPROCS(0))
	fmt.Println()

	// ── HIGH vs LOW ALLOC WORKLOAD ────────────────────────────────────────────
	fmt.Println("--- High-alloc vs low-alloc workload (10k iterations) ---")
	runtime.GC()
	b1 := readGCStats()
	t1 := time.Now()
	highAlloc(10_000)
	runtime.GC()
	d1 := time.Since(t1)
	e1 := b1.delta(readGCStats())

	runtime.GC()
	b2 := readGCStats()
	t2 := time.Now()
	lowAlloc(10_000)
	runtime.GC()
	d2 := time.Since(t2)
	e2 := b2.delta(readGCStats())

	fmt.Printf("  highAlloc: time=%v  GC cycles=%d  GC pause=%v\n",
		d1.Round(time.Millisecond), e1.numGC, e1.pauseTotal.Round(time.Microsecond))
	fmt.Printf("  lowAlloc:  time=%v  GC cycles=%d  GC pause=%v\n",
		d2.Round(time.Millisecond), e2.numGC, e2.pauseTotal.Round(time.Microsecond))
	fmt.Println()

	// ── GOGC EXPERIMENT ───────────────────────────────────────────────────────
	fmt.Println("--- GOGC effect on GC frequency ---")
	experiments := []int{50, 100, 200, 400}
	for _, pct := range experiments {
		exp := runWithGOGC(pct, 5_000)
		fmt.Printf("  GOGC=%-4d  GC cycles=%-3d  total pause=%v\n",
			exp.gcpct, exp.numGC, exp.totalNs.Round(time.Microsecond))
	}
	fmt.Println("  Lower GOGC = more frequent GC = lower memory, higher CPU")
	fmt.Println("  Higher GOGC = less frequent GC = higher memory, lower CPU")
	fmt.Println()

	// ── GOMEMLIMIT REFERENCE ──────────────────────────────────────────────────
	fmt.Println("--- GOMEMLIMIT (Go 1.19+) ---")
	limit := debug.SetMemoryLimit(-1) // read current
	limitStr := "unlimited"
	if limit < 1<<62 {
		limitStr = strconv.FormatInt(limit/1024/1024, 10) + " MiB"
	}
	fmt.Printf("  Current GOMEMLIMIT: %s\n", limitStr)
	fmt.Println("  Set programmatically: debug.SetMemoryLimit(512 << 20)")
	fmt.Println("  Set via env: GOMEMLIMIT=512MiB go run ./...")
	fmt.Println("  Effect: GC triggers aggressively when heap approaches limit.")
	fmt.Println()

	// ── FINALIZERS ────────────────────────────────────────────────────────────
	fmt.Println("--- Finalizers ---")
	r := NewResource("db-conn-1")
	fmt.Printf("  Resource %q created with finalizer\n", r.Name)
	r.Close()
	fmt.Printf("  Resource %q explicitly closed — finalizer removed\n", r.Name)
	fmt.Println("  Rules:")
	fmt.Println("    - finalizers are non-deterministic — don't rely for correctness")
	fmt.Println("    - always provide an explicit Close/Destroy method")
	fmt.Println("    - remove the finalizer in Close (runtime.SetFinalizer(r, nil))")
	fmt.Println("    - cyclic references prevent finalizer from firing")
	fmt.Println()

	// ── GC TRACE REFERENCE ────────────────────────────────────────────────────
	fmt.Println("--- GODEBUG=gctrace=1 output reference ---")
	fmt.Print(gcTraceRef)

	fmt.Println("--- Heap profiling reference ---")
	fmt.Print(heapProfileRef)
}
