// EXERCISE 14.1 — Thread-safe counter using closures.
//
// Implement makeThreadSafeCounter() which returns three functions:
//   inc()      — increments the counter by 1
//   dec()      — decrements the counter by 1
//   value() int — returns the current value
//
// The counter must be safe to use from multiple goroutines concurrently.
//
// Then implement makeRateLimiter(limit int) that returns a func() bool.
// Each call returns true if the caller is within the rate limit, false if
// the limit has been exceeded. The limit resets each time you call reset().
//
// Run (from the chapter folder):
//   go run ./exercises/01_counter

package main

import (
	"fmt"
	"sync"
)

func makeThreadSafeCounter() (inc func(), dec func(), value func() int) {
	var mu sync.Mutex
	count := 0

	inc = func() {
		mu.Lock()
		count++
		mu.Unlock()
	}
	dec = func() {
		mu.Lock()
		count--
		mu.Unlock()
	}
	value = func() int {
		mu.Lock()
		defer mu.Unlock()
		return count
	}
	return
}

func makeRateLimiter(limit int) (allow func() bool, reset func()) {
	var mu sync.Mutex
	used := 0

	allow = func() bool {
		mu.Lock()
		defer mu.Unlock()
		if used >= limit {
			return false
		}
		used++
		return true
	}
	reset = func() {
		mu.Lock()
		used = 0
		mu.Unlock()
	}
	return
}

func main() {
	// Thread-safe counter
	inc, dec, value := makeThreadSafeCounter()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inc()
		}()
	}
	for range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec()
		}()
	}
	wg.Wait()
	fmt.Println("counter after 100 inc + 30 dec:", value()) // 70

	fmt.Println()

	// Rate limiter
	allow, reset := makeRateLimiter(3)
	for i := range 5 {
		fmt.Printf("call %d: allowed=%v\n", i+1, allow())
	}
	fmt.Println("resetting...")
	reset()
	for i := range 2 {
		fmt.Printf("call %d after reset: allowed=%v\n", i+1, allow())
	}
}
