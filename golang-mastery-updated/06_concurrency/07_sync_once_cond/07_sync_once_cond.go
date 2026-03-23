// FILE: 06_concurrency/07_sync_once_cond.go
// TOPIC: sync.Once, sync.Cond, sync.Pool — advanced sync primitives
//
// Run: go run 06_concurrency/07_sync_once_cond.go

package main

import (
	"fmt"
	"sync"
	"time"
)

// ── sync.Once — run initialization exactly once ───────────────────────────────
// Even if 1000 goroutines call once.Do(f) simultaneously, f runs exactly ONCE.
// All callers block until f completes, then all proceed.
// This is the thread-safe singleton pattern.

type Database struct{ Name string }

var (
	dbOnce     sync.Once
	dbInstance *Database
)

func getDB() *Database {
	dbOnce.Do(func() {
		fmt.Println("  [DB] Connecting... (expensive, runs once)")
		time.Sleep(10 * time.Millisecond) // simulate slow init
		dbInstance = &Database{Name: "postgres://localhost/mydb"}
	})
	return dbInstance
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: sync.Once, sync.Cond, sync.Pool")
	fmt.Println("════════════════════════════════════════")

	// ── sync.Once ─────────────────────────────────────────────────────────
	fmt.Println("\n── sync.Once (singleton init) ──")
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			db := getDB()  // 5 goroutines call this, init runs ONCE
			fmt.Printf("  goroutine %d got db: %s\n", id, db.Name)
		}(i)
	}
	wg.Wait()

	// ── sync.Pool — reuse temporary objects to reduce GC pressure ─────────
	// sync.Pool holds objects that can be reused.
	// When GC runs, it may clear the pool.
	// Use for: frequently allocated short-lived objects (buffers, request contexts).
	fmt.Println("\n── sync.Pool ──")
	pool := &sync.Pool{
		New: func() interface{} {
			fmt.Println("  Pool: creating new buffer")
			return make([]byte, 1024)
		},
	}

	// Get from pool (calls New if empty):
	buf1 := pool.Get().([]byte)
	fmt.Printf("  Got buffer: len=%d\n", len(buf1))

	// Return to pool for reuse:
	pool.Put(buf1)

	// Get again — reuses the same buffer (no allocation):
	buf2 := pool.Get().([]byte)
	fmt.Printf("  Got buffer again: len=%d (reused, no 'creating' log)\n", len(buf2))
	pool.Put(buf2)

	// ── sync.Cond — condition variable ────────────────────────────────────
	// sync.Cond lets goroutines wait for a condition to become true.
	// Less common than channels, but useful for producer-consumer with
	// multiple consumers waiting on the same condition.
	fmt.Println("\n── sync.Cond (producer-consumer) ──")

	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	var ready bool
	var results []int

	// Consumers: wait until data is ready
	var consumerWg sync.WaitGroup
	for i := 0; i < 3; i++ {
		consumerWg.Add(1)
		go func(id int) {
			defer consumerWg.Done()
			mu.Lock()
			for !ready {  // loop: re-check condition after wakeup (spurious wakeups)
				cond.Wait()  // atomically: unlock mu, sleep, relock mu on wakeup
			}
			fmt.Printf("  Consumer %d: results=%v\n", id, results)
			mu.Unlock()
		}(i)
	}

	// Producer: produce data, then broadcast
	time.Sleep(20 * time.Millisecond) // let consumers start waiting
	mu.Lock()
	results = []int{1, 2, 3}
	ready = true
	cond.Broadcast()  // wake ALL waiting goroutines (Signal wakes ONE)
	mu.Unlock()

	consumerWg.Wait()

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  sync.Once: run init exactly once, thread-safe singleton")
	fmt.Println("  sync.Pool: reuse objects, reduce GC pressure (cleared on GC)")
	fmt.Println("  sync.Cond: wait for condition, Broadcast (all) or Signal (one)")
	fmt.Println("  Always loop-check condition with Wait (spurious wakeups)")
}
