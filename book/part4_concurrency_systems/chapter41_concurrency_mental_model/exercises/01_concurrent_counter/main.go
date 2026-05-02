// FILE: book/part4_concurrency_systems/chapter41_concurrency_mental_model/exercises/01_concurrent_counter/main.go
// CHAPTER: 41 — Concurrency Mental Model
// EXERCISE: Implement the same counter three ways — shared mutex, CSP actor,
//           and atomic — and observe that all produce the same result.
//
// Run (from the chapter folder):
//   go run ./exercises/01_concurrent_counter

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. MUTEX-GUARDED SHARED STATE
// ─────────────────────────────────────────────────────────────────────────────

type MutexCounter struct {
	mu    sync.Mutex
	value int
}

func (c *MutexCounter) Add(n int) {
	c.mu.Lock()
	c.value += n
	c.mu.Unlock()
}

func (c *MutexCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. CSP ACTOR — owns its state, receives messages on a channel
// ─────────────────────────────────────────────────────────────────────────────

type addMsg struct {
	n      int
	result chan int // non-nil means "send back the current value then stop"
}

type ActorCounter struct {
	msgs chan addMsg
}

func NewActorCounter() *ActorCounter {
	c := &ActorCounter{msgs: make(chan addMsg, 64)}
	go func() {
		value := 0
		for msg := range c.msgs {
			value += msg.n
			if msg.result != nil {
				msg.result <- value
				return
			}
		}
	}()
	return c
}

func (c *ActorCounter) Add(n int) {
	c.msgs <- addMsg{n: n}
}

func (c *ActorCounter) Value() int {
	result := make(chan int)
	c.msgs <- addMsg{result: result}
	return <-result
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. ATOMIC COUNTER
// ─────────────────────────────────────────────────────────────────────────────

type AtomicCounter struct {
	value int64
}

func (c *AtomicCounter) Add(n int) {
	atomic.AddInt64(&c.value, int64(n))
}

func (c *AtomicCounter) Value() int {
	return int(atomic.LoadInt64(&c.value))
}

// ─────────────────────────────────────────────────────────────────────────────
// BENCHMARK HELPER — run N goroutines each adding M
// ─────────────────────────────────────────────────────────────────────────────

type Counter interface {
	Add(n int)
	Value() int
}

func runConcurrent(c Counter, goroutines, addsPerGoroutine int) int {
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				c.Add(1)
			}
		}()
	}
	wg.Wait()
	return c.Value()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	const (
		goroutines       = 100
		addsPerGoroutine = 100
		expected         = goroutines * addsPerGoroutine
	)

	fmt.Printf("Running %d goroutines × %d adds = %d expected\n\n",
		goroutines, addsPerGoroutine, expected)

	mutexResult := runConcurrent(&MutexCounter{}, goroutines, addsPerGoroutine)
	fmt.Printf("  MutexCounter:  %d  (correct: %v)\n", mutexResult, mutexResult == expected)

	actorResult := runConcurrent(NewActorCounter(), goroutines, addsPerGoroutine)
	fmt.Printf("  ActorCounter:  %d  (correct: %v)\n", actorResult, actorResult == expected)

	atomicResult := runConcurrent(&AtomicCounter{}, goroutines, addsPerGoroutine)
	fmt.Printf("  AtomicCounter: %d  (correct: %v)\n", atomicResult, atomicResult == expected)

	fmt.Println()
	fmt.Println("All three approaches serialize access differently:")
	fmt.Println("  Mutex   — lock contention; general purpose")
	fmt.Println("  Actor   — single owner goroutine; no lock, but channel overhead")
	fmt.Println("  Atomic  — hardware CAS; fastest for simple integers")
}
