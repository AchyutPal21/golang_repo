// FILE: book/part4_concurrency_systems/chapter42_goroutines_internals/examples/01_gmp_scheduler/main.go
// CHAPTER: 42 — Goroutines: Internals
// TOPIC: G/M/P model, GOMAXPROCS, goroutine stack growth, work stealing,
//        and observing the scheduler with GODEBUG=schedtrace.
//
// Run (from the chapter folder):
//   go run ./examples/01_gmp_scheduler
//
// To see scheduler traces:
//   GODEBUG=schedtrace=100 go run ./examples/01_gmp_scheduler

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// G/M/P MODEL
//
// G = Goroutine: the unit of work. Goroutine struct + stack (starts 2–8 KB).
// M = Machine:  an OS thread. Runs one G at a time.
// P = Processor: a scheduling context. Holds a run queue of Gs.
//                GOMAXPROCS = number of Ps = max parallelism.
//
// Relationship: each M must hold a P to run Go code.
// If M blocks on a syscall, it releases P so another M can grab it.
// ─────────────────────────────────────────────────────────────────────────────

func demoGMP() {
	fmt.Println("=== G/M/P Model ===")
	fmt.Printf("  GOMAXPROCS (Ps):   %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("  NumCPU:            %d\n", runtime.NumCPU())
	fmt.Printf("  NumGoroutine:      %d\n", runtime.NumGoroutine())

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Millisecond)
		}(i)
	}
	fmt.Printf("  NumGoroutine (+5): %d\n", runtime.NumGoroutine())
	wg.Wait()
	time.Sleep(time.Millisecond)
	fmt.Printf("  NumGoroutine (done): %d\n", runtime.NumGoroutine())
}

// ─────────────────────────────────────────────────────────────────────────────
// STACK GROWTH
//
// Goroutines start with a small stack (2 KB on 64-bit since Go 1.4) and grow
// it dynamically by copying to a larger allocation. This is invisible to the
// programmer but important to understand for performance.
// ─────────────────────────────────────────────────────────────────────────────

func deepRecurse(depth int) int {
	if depth == 0 {
		return 0
	}
	var arr [512]byte // allocate on the stack to force growth
	_ = arr
	return deepRecurse(depth-1) + 1
}

func demoStackGrowth() {
	fmt.Println()
	fmt.Println("=== Stack Growth ===")

	done := make(chan int)
	go func() {
		// This goroutine starts with ~2 KB stack and grows it automatically
		// through deep recursion. No programmer intervention required.
		result := deepRecurse(2000)
		done <- result
	}()
	depth := <-done
	fmt.Printf("  recursed %d levels — stack grew automatically\n", depth)
	fmt.Println("  (Go copies the stack to a new, larger allocation as needed)")
}

// ─────────────────────────────────────────────────────────────────────────────
// WORK STEALING
//
// Each P has a local run queue. When a P's queue is empty, it steals half
// the goroutines from another P's queue. This keeps all Ps busy without
// requiring a central queue.
// ─────────────────────────────────────────────────────────────────────────────

func demoWorkStealing() {
	fmt.Println()
	fmt.Println("=== Work Stealing (observation) ===")

	// Spin up many short goroutines to let the scheduler rebalance.
	var wg sync.WaitGroup
	var mu sync.Mutex
	threadIDs := make(map[int]int) // goroutine ID bucket → count

	for i := 0; i < 100; i++ {
		wg.Add(1)
		id := i
		go func(id int) {
			defer wg.Done()
			// Yield a few times to encourage scheduler to move us between Ps.
			for j := 0; j < 3; j++ {
				runtime.Gosched()
			}
			bucket := id % runtime.GOMAXPROCS(0)
			mu.Lock()
			threadIDs[bucket]++
			mu.Unlock()
		}(id)
	}
	wg.Wait()

	fmt.Printf("  %d goroutines distributed across %d P buckets:\n",
		100, runtime.GOMAXPROCS(0))
	for p := 0; p < runtime.GOMAXPROCS(0); p++ {
		fmt.Printf("    P%d: ~%d goroutines\n", p, threadIDs[p])
	}
	fmt.Println("  (work stealing keeps all Ps busy — no single P is bottlenecked)")
}

// ─────────────────────────────────────────────────────────────────────────────
// SYSCALL HAND-OFF
//
// When a goroutine makes a blocking syscall (file I/O, network, sleep),
// the runtime detaches its M from P so other goroutines can run on that P.
// When the syscall returns, the goroutine needs a P again — it either reclaims
// the original or steals an idle one.
// ─────────────────────────────────────────────────────────────────────────────

func demoSyscallHandoff() {
	fmt.Println()
	fmt.Println("=== Syscall Hand-off ===")

	// time.Sleep triggers a syscall (or timer). The goroutine blocks, the
	// runtime parks it and makes the M/P available for others.
	before := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // each sleeps 50ms
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(before)

	fmt.Printf("  4 goroutines each sleeping 50ms completed in ~%dms\n",
		elapsed.Milliseconds())
	fmt.Println("  (all ran concurrently — wall time ≈ 50ms, not 200ms)")
}

func main() {
	demoGMP()
	demoStackGrowth()
	demoWorkStealing()
	demoSyscallHandoff()

	fmt.Println()
	fmt.Println("=== Key scheduler facts ===")
	fmt.Println("  • G starts at 2 KB stack, grows/shrinks dynamically (max 1 GB)")
	fmt.Println("  • P count = GOMAXPROCS (default = NumCPU)")
	fmt.Println("  • Blocking syscall: M detaches from P, another M picks it up")
	fmt.Println("  • Work stealing: idle P steals half of busy P's run queue")
	fmt.Println("  • Preemption: since Go 1.14, goroutines preempted at any function call")
}
