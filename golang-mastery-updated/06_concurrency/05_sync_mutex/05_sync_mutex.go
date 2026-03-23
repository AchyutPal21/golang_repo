// 05_sync_mutex.go
//
// SYNC.MUTEX AND SYNC.RWMUTEX — protecting shared state
//
// "Share memory by communicating" is the ideal. But sometimes you have a
// shared data structure that many goroutines need to access. Channels would
// require serializing all access through a single goroutine — a bottleneck.
// Mutexes (mutual exclusion locks) are the right tool for protecting shared state.
//
// WHAT IS A RACE CONDITION?
// -------------------------
// A race condition occurs when the behavior of a program depends on the
// relative timing of goroutines. Specifically:
//
//   - DATA RACE: two goroutines access the same memory location concurrently,
//     at least one is a write, and there's no synchronization between them.
//
// Data races are undefined behavior in Go. The CPU can:
//   - Reorder instructions (CPU out-of-order execution)
//   - Cache values in registers without flushing to memory
//   - Load stale values from cache on another core
//
// THE GO RACE DETECTOR
// --------------------
// Run any Go program with:    go run -race main.go
// Or test with:               go test -race ./...
//
// The race detector instruments every memory access and reports data races
// at runtime with the goroutine stack traces. It adds ~5-10x overhead but
// is indispensable for finding concurrent bugs.
//
// MUTEX TERMINOLOGY
// -----------------
// Mutex = mutual exclusion lock. At most one goroutine holds it at a time.
//   Lock()   — acquire the lock (blocks if held by another goroutine)
//   Unlock() — release the lock
//
// The region between Lock() and Unlock() is the "critical section".
// Only one goroutine executes the critical section at a time.

package main

import (
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// SECTION 1: Race condition demonstration
// =============================================================================
//
// Run this with:  go run -race 05_sync_mutex.go
// The race detector will report the data race on 'counter'.

type UnsafeCounter struct {
	value int // unprotected — data race when accessed concurrently
}

func (c *UnsafeCounter) Increment() {
	// THIS IS A DATA RACE:
	// Read current value → add 1 → write back
	// Multiple goroutines can read the same value, both add 1, both write back —
	// result: one increment is lost. Classic "lost update" problem.
	c.value++ // not atomic: compiles to LOAD, ADD, STORE (3 separate ops)
}

func (c *UnsafeCounter) Value() int {
	return c.value
}

func demoRaceCondition() {
	fmt.Println("=== Race Condition (run with -race to detect) ===")

	counter := &UnsafeCounter{}
	var wg sync.WaitGroup

	// 1000 goroutines each increment the counter once.
	// Expected result: 1000. Actual result: less (updates are lost).
	// With -race flag: race detector reports this.
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Increment()
		}()
	}

	wg.Wait()
	fmt.Printf("  unsafe counter (expected 1000, got): %d\n", counter.Value())
	fmt.Println("  (result varies each run — non-deterministic)")
	fmt.Println()
}

// =============================================================================
// SECTION 2: sync.Mutex — fixing the race condition
// =============================================================================

type SafeCounter struct {
	mu    sync.Mutex // the zero value is an unlocked mutex
	value int
}

// Increment is now safe for concurrent use. The mutex ensures only one
// goroutine executes the critical section at a time.
func (c *SafeCounter) Increment() {
	c.mu.Lock()   // acquire the lock — blocks if another goroutine holds it
	c.value++     // critical section — only ONE goroutine here at a time
	c.mu.Unlock() // release the lock — allows next waiter to proceed
}

// IncrementDefer uses defer for unlock — the recommended pattern.
// WHY DEFER: if the function panics or has multiple return paths,
// defer guarantees Unlock() always runs, preventing a deadlock.
func (c *SafeCounter) IncrementDefer() {
	c.mu.Lock()
	defer c.mu.Unlock() // runs when function returns, regardless of how
	c.value++
}

func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value // must hold lock even for reads (another goroutine could write)
}

// Reset demonstrates that you can call Lock multiple times on the SAME goroutine —
// wait, no: sync.Mutex is NOT reentrant. Calling Lock() while holding it DEADLOCKS.
func (c *SafeCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
	// DO NOT call c.Value() here — it tries to Lock() again → DEADLOCK
	// Instead, access c.value directly (we already hold the lock)
}

func demoMutex() {
	fmt.Println("=== sync.Mutex ===")

	counter := &SafeCounter{}
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.IncrementDefer()
		}()
	}

	wg.Wait()
	fmt.Printf("  safe counter (always 1000): %d\n", counter.Value())

	counter.Reset()
	fmt.Printf("  after reset: %d\n", counter.Value())
	fmt.Println()
}

// =============================================================================
// SECTION 3: sync.RWMutex — optimized for read-heavy workloads
// =============================================================================
//
// sync.Mutex allows only ONE goroutine at a time (readers block each other).
// sync.RWMutex is smarter:
//
//   - MULTIPLE goroutines can hold RLock() simultaneously (concurrent reads OK)
//   - Only ONE goroutine can hold Lock() (exclusive write — no concurrent reads)
//
// RWMutex methods:
//   RLock()   / RUnlock()  — shared (read) lock
//   Lock()    / Unlock()   — exclusive (write) lock
//
// WHEN TO USE:
//   - Read operations >> Write operations
//   - Reads don't modify shared state
//   - Read operations take non-trivial time (worth the overhead of RWMutex)
//
// IF READS ARE FAST (just reading a field), Mutex is often fine due to
// RWMutex's higher overhead. Profile before optimizing.
//
// WRITER STARVATION PREVENTION: Go's RWMutex is fair — once a writer is
// waiting, new readers must wait for the writer to finish first.

type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]string)}
}

// Get acquires a READ lock — multiple goroutines can Get() concurrently.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()         // acquire shared read lock
	defer c.mu.RUnlock() // release when done
	val, ok := c.data[key]
	return val, ok
}

// Set acquires an EXCLUSIVE write lock — no other goroutines can Get() or Set().
func (c *Cache) Set(key, value string) {
	c.mu.Lock()         // acquire exclusive write lock
	defer c.mu.Unlock() // release when done
	c.data[key] = value
}

// Delete also needs the write lock.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func demoRWMutex() {
	fmt.Println("=== sync.RWMutex ===")

	cache := NewCache()
	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n)
			cache.Set(key, fmt.Sprintf("value-%d", n))
			fmt.Printf("  wrote %s\n", key)
		}(i)
	}

	// Reader goroutines — many can run concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond) // let some writes happen first
			key := fmt.Sprintf("key-%d", n%5)
			val, ok := cache.Get(key)
			if ok {
				fmt.Printf("  read %s = %s\n", key, val)
			} else {
				fmt.Printf("  %s not found yet\n", key)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println()
}

// =============================================================================
// SECTION 4: Mutex vs Channel — when to use which
// =============================================================================
//
// This is a common design question. The guidelines:
//
// USE MUTEX when:
//   - You have shared state (struct fields, maps) accessed by multiple goroutines
//   - The operations are short (just increment, read, update)
//   - You're wrapping an existing non-concurrent data structure
//   - Performance matters for frequent access
//
// USE CHANNEL when:
//   - You're passing ownership of data between goroutines
//   - You're coordinating (signalling, event notification)
//   - You want to limit concurrency (buffered channel as semaphore)
//   - You want a pipeline of transformations
//
// The Go team's guidance: "Use whichever is more natural." Both are correct.
// Channels don't magically prevent data races — if you pass a pointer through
// a channel and both sides access it, you still have a race.

// Mutex-based shared state:
type MutexCounter struct {
	mu    sync.Mutex
	count int
}

func (c *MutexCounter) Inc() { c.mu.Lock(); c.count++; c.mu.Unlock() }
func (c *MutexCounter) Get() int { c.mu.Lock(); defer c.mu.Unlock(); return c.count }

// Channel-based "actor" counter — all access goes through a single goroutine.
// Advantage: no mutex needed, logic is serialized.
// Disadvantage: goroutine overhead, higher latency for each operation.
type ChannelCounter struct {
	inc chan struct{}
	get chan chan int
}

func NewChannelCounter() *ChannelCounter {
	c := &ChannelCounter{
		inc: make(chan struct{}, 100),
		get: make(chan chan int),
	}
	go func() {
		count := 0
		for {
			select {
			case <-c.inc:
				count++
			case reply := <-c.get:
				reply <- count
			}
		}
	}()
	return c
}

func (c *ChannelCounter) Inc() { c.inc <- struct{}{} }
func (c *ChannelCounter) Get() int {
	reply := make(chan int, 1)
	c.get <- reply
	return <-reply
}

func demoMutexVsChannel() {
	fmt.Println("=== Mutex vs Channel Counter ===")

	var wg sync.WaitGroup
	n := 500

	// Mutex counter
	mc := &MutexCounter{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); mc.Inc() }()
	}
	wg.Wait()
	fmt.Printf("  mutex counter: %d (expected %d)\n", mc.Get(), n)

	// Channel counter
	cc := NewChannelCounter()
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); cc.Inc() }()
	}
	wg.Wait()
	time.Sleep(10 * time.Millisecond) // let buffered increments process
	fmt.Printf("  channel counter: %d (expected %d)\n", cc.Get(), n)

	fmt.Println()
}

// =============================================================================
// SECTION 5: Deadlock — causes and prevention
// =============================================================================
//
// A deadlock occurs when goroutines are all waiting for each other — none
// can proceed. The Go runtime detects deadlocks where ALL goroutines are
// blocked and panics with: "all goroutines are asleep - deadlock!"
//
// COMMON CAUSES:
//   1. Lock ordering inconsistency (A locks mu1 then mu2; B locks mu2 then mu1)
//   2. Calling Lock() inside a function that already holds the lock (mutex not reentrant)
//   3. Forgetting to unlock (e.g., early return without defer)
//   4. Channel deadlock (nobody receives from an unbuffered channel)
//
// PREVENTION:
//   1. Always use defer mu.Unlock() immediately after Lock()
//   2. Establish a global lock ordering and always acquire in that order
//   3. Keep critical sections small — don't call external functions while locked
//   4. Avoid holding a lock while waiting on a channel

func demoDeadlockPrevention() {
	fmt.Println("=== Deadlock Prevention ===")

	// Simulate lock ordering problem and solution:
	var mu1, mu2 sync.Mutex

	// WRONG (could deadlock if run concurrently):
	// goroutine 1: mu1.Lock() → mu2.Lock()
	// goroutine 2: mu2.Lock() → mu1.Lock()
	// Each holds one lock and waits for the other.

	// CORRECT: always lock in the same order (mu1 before mu2, everywhere)
	safeLockBoth := func(id int) {
		mu1.Lock()         // always mu1 first
		defer mu1.Unlock() // deferred: unlocks even if panic occurs
		mu2.Lock()         // then mu2
		defer mu2.Unlock()
		fmt.Printf("  goroutine %d: holding both locks\n", id)
		time.Sleep(5 * time.Millisecond)
	}

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			safeLockBoth(id)
		}(i)
	}
	wg.Wait()

	// Demonstrating mutex not reentrant:
	// DO NOT do this:
	//   mu1.Lock()
	//   doSomethingThatCallsMu1Lock() // mu1.Lock() again → DEADLOCK
	//
	// Solution: design your API so locked functions access data directly,
	// and public methods (that grab the lock) don't call each other.

	fmt.Println("  All goroutines completed without deadlock")
	fmt.Println()
}

// =============================================================================
// SECTION 6: sync.Mutex — embedding for cleaner API
// =============================================================================
//
// Embedding sync.Mutex in a struct is an option, but can expose Lock/Unlock
// in the public API (if the struct is exported). Usually, keep mu unexported.

type SafeMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func NewSafeMap() *SafeMap {
	return &SafeMap{m: make(map[string]int)}
}

func (sm *SafeMap) Set(k string, v int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[k] = v
}

func (sm *SafeMap) Get(k string) (int, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.m[k]
	return v, ok
}

// Keys returns a snapshot of all keys — must hold read lock for the duration.
func (sm *SafeMap) Keys() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	keys := make([]string, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}
	return keys // safe: we copied the keys, not a reference to the map
}

func demoSafeMap() {
	fmt.Println("=== SafeMap with RWMutex ===")

	sm := NewSafeMap()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sm.Set(fmt.Sprintf("k%d", n), n*10)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			v, ok := sm.Get(fmt.Sprintf("k%d", n))
			if ok {
				fmt.Printf("  k%d = %d\n", n, v)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         SYNC.MUTEX & SYNC.RWMUTEX — Deep Dive        ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	demoRaceCondition()
	demoMutex()
	demoRWMutex()
	demoMutexVsChannel()
	demoDeadlockPrevention()
	demoSafeMap()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. Data race: concurrent access to shared memory, at least one write")
	fmt.Println("  2. Run with -race flag to detect races at runtime")
	fmt.Println("  3. sync.Mutex: only 1 goroutine in critical section at a time")
	fmt.Println("  4. Always: mu.Lock(); defer mu.Unlock() — defer prevents forgotten unlock")
	fmt.Println("  5. Mutex is NOT reentrant — second Lock() on same goroutine deadlocks")
	fmt.Println("  6. sync.RWMutex: many concurrent readers OR one exclusive writer")
	fmt.Println("  7. Mutex for shared state; channel for passing ownership/signalling")
	fmt.Println("  8. Deadlock prevention: consistent lock order + always defer unlock")
	fmt.Println("═══════════════════════════════════════════════════════")
}
