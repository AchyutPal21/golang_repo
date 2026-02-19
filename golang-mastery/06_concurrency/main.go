package main

// =============================================================================
// MODULE 06: CONCURRENCY — Goroutines, Channels, sync
// =============================================================================
// Run: go run 06_concurrency/main.go
//
// Go's concurrency model: CSP (Communicating Sequential Processes)
// Philosophy: "Don't communicate by sharing memory;
//              share memory by communicating."
// =============================================================================

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// GOROUTINES — Lightweight threads managed by Go runtime
// =============================================================================
// go f() — launches f in a new goroutine.
// Goroutines are NOT OS threads — they're multiplexed onto OS threads.
// Start with 2KB stack (grows dynamically). OS threads: 1-8MB fixed.
// You can have millions of goroutines; OS threads: maybe thousands.

// =============================================================================
// CHANNELS — Typed pipes for communication between goroutines
// =============================================================================
// ch := make(chan T)           — unbuffered channel
// ch := make(chan T, capacity) — buffered channel
// ch <- value                 — send (blocks if full)
// value := <-ch               — receive (blocks if empty)
// close(ch)                   — signal no more values will be sent

// =============================================================================
// Unbuffered channel — synchronous handshake
// =============================================================================
func ping(ch chan<- string) { // chan<- : send-only channel
	ch <- "ping"
}

func pong(in <-chan string, out chan<- string) { // <-chan : receive-only channel
	msg := <-in
	out <- msg + "-pong"
}

// =============================================================================
// Goroutine with WaitGroup — wait for goroutines to finish
// =============================================================================
func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done() // decrements counter when function returns
	fmt.Printf("Worker %d starting\n", id)
	time.Sleep(time.Millisecond * time.Duration(100*id))
	fmt.Printf("Worker %d done\n", id)
}

// =============================================================================
// BUFFERED CHANNELS — async up to capacity
// =============================================================================
// Sends to a buffered channel block only when the buffer is full.
// Receives block only when the buffer is empty.

// =============================================================================
// RANGE over channel — receive until closed
// =============================================================================
func generateNumbers(ch chan<- int, n int) {
	for i := 1; i <= n; i++ {
		ch <- i
	}
	close(ch) // signal done — range loop will exit
}

// =============================================================================
// SELECT — multiplex channel operations
// =============================================================================
// select is like a switch but for channels.
// It blocks until one case is ready.
// If multiple cases are ready, one is chosen randomly.

func selectDemo() {
	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(100 * time.Millisecond)
		ch1 <- "one"
	}()

	go func() {
		time.Sleep(200 * time.Millisecond)
		ch2 <- "two"
	}()

	// Receive from whichever is ready first
	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-ch1:
			fmt.Println("received from ch1:", msg1)
		case msg2 := <-ch2:
			fmt.Println("received from ch2:", msg2)
		}
	}
}

// Non-blocking select using default
func nonBlockingSelect(ch chan int) {
	select {
	case val := <-ch:
		fmt.Println("received:", val)
	default:
		fmt.Println("no value ready (non-blocking)")
	}
}

// Timeout pattern using select
func withTimeout(ch <-chan string, timeout time.Duration) (string, bool) {
	select {
	case val := <-ch:
		return val, true
	case <-time.After(timeout):
		return "", false // timed out
	}
}

// =============================================================================
// DONE CHANNEL — cancellation pattern
// =============================================================================
func producer(done <-chan struct{}, out chan<- int) {
	for i := 0; ; i++ {
		select {
		case <-done: // cancelled
			fmt.Println("producer: stopping")
			return
		case out <- i:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// =============================================================================
// MUTEX — protect shared state from race conditions
// =============================================================================
// sync.Mutex: exclusive lock
//   mu.Lock()   — acquire lock (blocks if held)
//   mu.Unlock() — release lock
// Always use defer mu.Unlock() right after Lock()!

type SafeCounter struct {
	mu    sync.Mutex
	count int
}

func (c *SafeCounter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock() // ensures unlock even if panic
	c.count++
}

func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// RWMutex — for read-heavy workloads
// Multiple readers can hold lock simultaneously
// Writers get exclusive access
type SafeMap struct {
	mu   sync.RWMutex
	data map[string]int
}

func NewSafeMap() *SafeMap {
	return &SafeMap{data: make(map[string]int)}
}

func (m *SafeMap) Set(key string, val int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
}

func (m *SafeMap) Get(key string) (int, bool) {
	m.mu.RLock() // read lock — multiple goroutines can hold this
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

// =============================================================================
// ATOMIC — low-level, lock-free operations (faster than mutex for simple counters)
// =============================================================================
// Only for basic types: int32, int64, uint32, uint64, uintptr, Pointer

type AtomicCounter struct {
	count int64 // must use int64 for atomic ops
}

func (c *AtomicCounter) Increment() {
	atomic.AddInt64(&c.count, 1)
}

func (c *AtomicCounter) Value() int64 {
	return atomic.LoadInt64(&c.count)
}

// =============================================================================
// WORKER POOL PATTERN — most common concurrency pattern
// =============================================================================
// A fixed number of goroutines process jobs from a shared queue.

func workerPool(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs { // receives until channel closed
		fmt.Printf("worker %d processing job %d\n", id, job)
		time.Sleep(10 * time.Millisecond)
		results <- job * job // square the number
	}
}

// =============================================================================
// PIPELINE PATTERN — stages connected by channels
// =============================================================================
// Stage 1 → | channel | → Stage 2 → | channel | → Stage 3

func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out)
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * n
		}
		close(out)
	}()
	return out
}

func addTen(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n + 10
		}
		close(out)
	}()
	return out
}

// =============================================================================
// SYNC.ONCE — run initialization exactly once
// =============================================================================
type Database struct {
	once     sync.Once
	conn     string
}

func (db *Database) Connect() {
	db.once.Do(func() {
		fmt.Println("Connecting to database (only once!)")
		db.conn = "connected"
	})
}

// =============================================================================
// SYNC.COND — condition variable for complex coordination
// =============================================================================

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 06: CONCURRENCY ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Basic goroutine
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Basic Goroutine ---")

	go func() {
		fmt.Println("goroutine: hello from goroutine")
	}()

	time.Sleep(10 * time.Millisecond) // wait for goroutine (bad practice — use WaitGroup)
	fmt.Println("main: continuing")

	// -------------------------------------------------------------------------
	// SECTION 2: WaitGroup — proper goroutine synchronization
	// -------------------------------------------------------------------------
	fmt.Println("\n--- WaitGroup ---")

	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		wg.Add(1) // increment counter BEFORE starting goroutine
		go worker(i, &wg)
	}

	wg.Wait() // blocks until counter reaches 0
	fmt.Println("all workers done")

	// -------------------------------------------------------------------------
	// SECTION 3: Unbuffered channels
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Unbuffered Channels ---")

	ch := make(chan string)
	done := make(chan string)

	go ping(ch)
	go pong(ch, done)

	result := <-done
	fmt.Println(result)

	// Simple goroutine-to-main communication
	msgCh := make(chan string)
	go func() {
		msgCh <- "message from goroutine"
	}()
	msg := <-msgCh
	fmt.Println("received:", msg)

	// -------------------------------------------------------------------------
	// SECTION 4: Buffered channels
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Buffered Channels ---")

	buffered := make(chan int, 3) // buffer 3 items
	buffered <- 1 // doesn't block
	buffered <- 2 // doesn't block
	buffered <- 3 // doesn't block
	// buffered <- 4 // would BLOCK (buffer full)

	fmt.Println(<-buffered) // 1
	fmt.Println(<-buffered) // 2
	fmt.Println(<-buffered) // 3

	// -------------------------------------------------------------------------
	// SECTION 5: Range over channel
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Range over Channel ---")

	numCh := make(chan int, 10) // buffered so generate doesn't block
	go generateNumbers(numCh, 5)

	for n := range numCh { // loops until channel closed
		fmt.Print(n, " ")
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// SECTION 6: Select
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Select ---")
	selectDemo()

	// Non-blocking
	emtpyCh := make(chan int)
	nonBlockingSelect(emtpyCh)

	// Timeout pattern
	slowCh := make(chan string)
	go func() {
		time.Sleep(500 * time.Millisecond)
		slowCh <- "slow result"
	}()

	val, ok := withTimeout(slowCh, 100*time.Millisecond)
	if !ok {
		fmt.Println("operation timed out")
	} else {
		fmt.Println("got:", val)
	}

	// -------------------------------------------------------------------------
	// SECTION 7: Done channel (cancellation)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Done Channel (Cancellation) ---")

	doneCh := make(chan struct{})
	outCh := make(chan int, 10)

	go producer(doneCh, outCh)
	time.Sleep(50 * time.Millisecond)
	close(doneCh) // signal producer to stop

	// Drain remaining values
	time.Sleep(20 * time.Millisecond)
	close(outCh)
	count := 0
	for range outCh {
		count++
	}
	fmt.Println("produced", count, "values before cancellation")

	// -------------------------------------------------------------------------
	// SECTION 8: Mutex
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Mutex ---")

	counter := &SafeCounter{}
	var wg2 sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			counter.Increment()
		}()
	}
	wg2.Wait()
	fmt.Println("counter (mutex):", counter.Value()) // 1000

	// -------------------------------------------------------------------------
	// SECTION 9: Atomic counter
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Atomic ---")

	atomicCounter := &AtomicCounter{}
	var wg3 sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg3.Add(1)
		go func() {
			defer wg3.Done()
			atomicCounter.Increment()
		}()
	}
	wg3.Wait()
	fmt.Println("counter (atomic):", atomicCounter.Value()) // 1000

	// -------------------------------------------------------------------------
	// SECTION 10: Worker Pool Pattern
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Worker Pool ---")

	jobs := make(chan int, 20)
	results := make(chan int, 20)

	var wg4 sync.WaitGroup
	// Start 3 workers
	for w := 1; w <= 3; w++ {
		wg4.Add(1)
		go workerPool(w, jobs, results, &wg4)
	}

	// Send 9 jobs
	for j := 1; j <= 9; j++ {
		jobs <- j
	}
	close(jobs) // no more jobs — workers will exit their range loop

	// Wait for all workers, then close results
	go func() {
		wg4.Wait()
		close(results)
	}()

	// Collect results
	sum := 0
	for r := range results {
		sum += r
	}
	fmt.Println("sum of squares (1-9):", sum) // 1+4+9+16+25+36+49+64+81=285

	// -------------------------------------------------------------------------
	// SECTION 11: Pipeline Pattern
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Pipeline ---")

	// generate → square → addTen → print
	pipeline := addTen(square(generate(1, 2, 3, 4, 5)))
	for v := range pipeline {
		fmt.Print(v, " ") // (1²+10), (2²+10), ... = 11 14 19 26 35
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// SECTION 12: sync.Once
	// -------------------------------------------------------------------------
	fmt.Println("\n--- sync.Once ---")

	db := &Database{}
	var wg5 sync.WaitGroup
	// Call Connect from 5 goroutines — only executes once
	for i := 0; i < 5; i++ {
		wg5.Add(1)
		go func() {
			defer wg5.Done()
			db.Connect()
		}()
	}
	wg5.Wait()
	fmt.Println("db connection:", db.conn)

	// -------------------------------------------------------------------------
	// SECTION 13: Channel directions (type safety)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Channel Directions ---")
	// chan T     — bidirectional
	// chan<- T   — send only  (you can ONLY send to it)
	// <-chan T   — receive only (you can ONLY receive from it)
	// This enforces correct usage at compile time — prevents bugs

	fmt.Println(`
Channel direction rules:
  chan T    → can send and receive
  chan<- T  → can only send (prevents receiving)
  <-chan T  → can only receive (prevents sending)

  Bidirectional chan T converts to directional automatically.
  Directional channel CANNOT convert back to bidirectional.
`)

	// -------------------------------------------------------------------------
	// SECTION 14: Race condition detection
	// -------------------------------------------------------------------------
	fmt.Println("--- Race Condition Detection ---")
	fmt.Println(`
To detect races, run with -race flag:
  go run -race 06_concurrency/main.go
  go test -race ./...

The race detector reports any concurrent access to shared memory
without synchronization. ALWAYS test with -race before shipping!
`)

	fmt.Println("=== MODULE 06 COMPLETE ===")
}
