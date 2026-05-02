// FILE: book/part4_concurrency_systems/chapter41_concurrency_mental_model/examples/01_csp_goroutines/main.go
// CHAPTER: 41 — Concurrency Mental Model
// TOPIC: CSP vs shared-memory, goroutines as independent workers,
//        channels as the communication primitive, and "don't communicate
//        by sharing memory — share memory by communicating."
//
// Run (from the chapter folder):
//   go run ./examples/01_csp_goroutines

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SHARED MEMORY APPROACH (the wrong mental model)
//
// Multiple goroutines read and write the same variable guarded by a mutex.
// This works, but it is not the idiomatic Go model — it looks like C with locks.
// ─────────────────────────────────────────────────────────────────────────────

func sharedMemoryCounter() int {
	var mu sync.Mutex
	count := 0

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			count++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return count
}

// ─────────────────────────────────────────────────────────────────────────────
// CSP APPROACH — communicate by passing values through channels
//
// Only one goroutine (the counter) owns the state.
// Other goroutines send increment requests and receive the final value.
// ─────────────────────────────────────────────────────────────────────────────

type counterMsg int

const (
	increment counterMsg = iota
	getAndQuit
)

func counterActor(msgs <-chan counterMsg, result chan<- int) {
	count := 0
	for msg := range msgs {
		switch msg {
		case increment:
			count++
		case getAndQuit:
			result <- count
			return
		}
	}
}

func cspCounter() int {
	msgs := make(chan counterMsg, 10)
	result := make(chan int)

	go counterActor(msgs, result)

	for i := 0; i < 5; i++ {
		msgs <- increment
	}
	msgs <- getAndQuit

	return <-result
}

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE — the fundamental CSP pattern
//
// Each stage is an independent goroutine that reads from one channel and
// writes to the next. Stages are composable and can run in parallel.
// ─────────────────────────────────────────────────────────────────────────────

// generate produces integers 0..n-1 on a channel, then closes it.
func generate(n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			out <- i
		}
	}()
	return out
}

// square reads integers, squares them, and sends results downstream.
func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

// filter keeps only values satisfying pred.
func filter(in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if pred(v) {
				out <- v
			}
		}
	}()
	return out
}

// collect drains a channel into a slice.
func collect(in <-chan int) []int {
	var result []int
	for v := range in {
		result = append(result, v)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE AS INDEPENDENT WORKER — fire and forget vs wait for result
// ─────────────────────────────────────────────────────────────────────────────

func simulateWork(id int, duration time.Duration, results chan<- string) {
	time.Sleep(duration)
	results <- fmt.Sprintf("worker %d done after %s", id, duration)
}

func demoFireAndCollect() {
	fmt.Println("=== Fire-and-collect goroutines ===")

	results := make(chan string, 3)
	go simulateWork(1, 30*time.Millisecond, results)
	go simulateWork(2, 10*time.Millisecond, results)
	go simulateWork(3, 20*time.Millisecond, results)

	for i := 0; i < 3; i++ {
		fmt.Println(" ", <-results)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// 1. Shared memory vs CSP counter.
	fmt.Println("=== Shared-memory vs CSP counter ===")
	fmt.Printf("  shared-memory result: %d\n", sharedMemoryCounter())
	fmt.Printf("  CSP result:           %d\n", cspCounter())

	// 2. Pipeline composition.
	fmt.Println()
	fmt.Println("=== Pipeline: generate → square → filter(even) ===")
	// 0²=0, 1²=1, 2²=4, 3²=9, 4²=16, 5²=25, 6²=36, 7²=49
	// even squares: 0, 4, 16, 36
	nums := generate(8)
	squares := square(nums)
	evens := filter(squares, func(v int) bool { return v%2 == 0 })
	fmt.Printf("  even squares of 0..7: %v\n", collect(evens))

	// 3. Fire and collect.
	fmt.Println()
	demoFireAndCollect()

	// 4. The Go mental model in one sentence.
	fmt.Println()
	fmt.Println("=== Mental model ===")
	fmt.Println("  Goroutines = independent workers.")
	fmt.Println("  Channels   = typed, directional message pipes.")
	fmt.Println("  Rule       = don't share memory, pass ownership through channels.")
}
