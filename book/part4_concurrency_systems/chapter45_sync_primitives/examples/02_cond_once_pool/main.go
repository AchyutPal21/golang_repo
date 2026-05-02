// FILE: book/part4_concurrency_systems/chapter45_sync_primitives/examples/02_cond_once_pool/main.go
// CHAPTER: 45 — sync Primitives
// TOPIC: sync.Cond (condition variable), sync.Once (one-time init),
//        sync.Pool (object reuse), and sync.WaitGroup (goroutine fan-out).
//
// Run (from the chapter folder):
//   go run ./examples/02_cond_once_pool

package main

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// sync.Cond — condition variable
//
// Used when a goroutine must wait for a condition that is signalled by
// another goroutine. Lower level than channels; useful when multiple
// goroutines need to wait on the same condition and be woken selectively.
// ─────────────────────────────────────────────────────────────────────────────

type Queue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []int
}

func NewQueue() *Queue {
	q := &Queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *Queue) Push(v int) {
	q.mu.Lock()
	q.items = append(q.items, v)
	q.cond.Signal() // wake one waiting consumer
	q.mu.Unlock()
}

func (q *Queue) PushAll(vs []int) {
	q.mu.Lock()
	q.items = append(q.items, vs...)
	q.cond.Broadcast() // wake all waiting consumers
	q.mu.Unlock()
}

func (q *Queue) Pop() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 {
		q.cond.Wait() // atomically unlocks mu and suspends the goroutine
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v
}

func demoCond() {
	fmt.Println("=== sync.Cond ===")

	q := NewQueue()
	var wg sync.WaitGroup

	// 3 consumers waiting for items.
	results := make(chan int, 6)
	for i := range 3 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			v := q.Pop() // blocks until Push signals
			results <- v
		}(i)
	}

	time.Sleep(10 * time.Millisecond) // let consumers block on cond.Wait

	// Broadcast wakes all three consumers.
	q.PushAll([]int{10, 20, 30})

	wg.Wait()
	close(results)

	var vals []int
	for v := range results {
		vals = append(vals, v)
	}
	fmt.Printf("  consumers received: %v\n", vals)
}

// ─────────────────────────────────────────────────────────────────────────────
// sync.Once — one-time initialisation
//
// The function passed to Do runs exactly once, even if called concurrently.
// Zero value is ready to use.
// ─────────────────────────────────────────────────────────────────────────────

type ExpensiveService struct {
	once sync.Once
	data string
}

func (s *ExpensiveService) init() {
	s.once.Do(func() {
		time.Sleep(10 * time.Millisecond) // simulate expensive setup
		s.data = "initialised"
		fmt.Println("  init ran (exactly once)")
	})
}

func (s *ExpensiveService) Data() string {
	s.init()
	return s.data
}

func demoOnce() {
	fmt.Println()
	fmt.Println("=== sync.Once ===")

	svc := &ExpensiveService{}
	var wg sync.WaitGroup

	// 10 goroutines call Data() concurrently — init runs only once.
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.Data()
		}()
	}
	wg.Wait()
	fmt.Printf("  data = %q\n", svc.Data())
}

// ─────────────────────────────────────────────────────────────────────────────
// sync.Pool — object reuse to reduce GC pressure
//
// Pool stores objects that can be reused between uses. Objects may be
// evicted by the GC at any time — Pool is a hint, not a guarantee.
// ─────────────────────────────────────────────────────────────────────────────

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func buildMessage(parts ...string) string {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset() // always reset before use
	defer bufPool.Put(buf)

	for i, p := range parts {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(p)
	}
	return buf.String()
}

func demoPool() {
	fmt.Println()
	fmt.Println("=== sync.Pool ===")

	var wg sync.WaitGroup
	results := make(chan string, 5)

	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := buildMessage(fmt.Sprintf("goroutine-%d", id), "says", "hello")
			results <- msg
		}(i)
	}
	wg.Wait()
	close(results)

	for msg := range results {
		fmt.Printf("  %s\n", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// sync.WaitGroup — wait for a group of goroutines to finish
// ─────────────────────────────────────────────────────────────────────────────

func demoWaitGroup() {
	fmt.Println()
	fmt.Println("=== sync.WaitGroup ===")

	var wg sync.WaitGroup
	results := make([]int, 5)

	for i := range 5 {
		wg.Add(1) // add BEFORE launching the goroutine
		go func(idx int) {
			defer wg.Done()
			time.Sleep(time.Duration(idx) * 5 * time.Millisecond)
			results[idx] = idx * idx
		}(i)
	}

	wg.Wait() // blocks until all Done() calls match Add() calls
	fmt.Printf("  results: %v\n", results)

	// Common mistake: Add inside the goroutine (race with Wait).
	fmt.Println("  (never call wg.Add inside the goroutine — it races with wg.Wait)")
}

func main() {
	demoCond()
	demoOnce()
	demoPool()
	demoWaitGroup()
}
