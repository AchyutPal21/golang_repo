// FILE: book/part2_core_language/chapter14_closures/examples/02_capture_model/main.go
// CHAPTER: 14 — Closures and the Capture Model
// TOPIC: The classic loop-variable capture bug, the Go 1.22 fix,
//        manual workarounds, and goroutine capture pitfalls.
//
// Run (from the chapter folder):
//   go run ./examples/02_capture_model

package main

import (
	"fmt"
	"sync"
)

// --- Loop variable capture ---

// buggyClosures demonstrates the pre-Go-1.22 loop-capture trap.
// All closures captured the same loop variable i, so they all
// printed the final value (3) when called.
//
// In Go 1.22+ each iteration creates a fresh i, so this prints 0 1 2.
// The output is correct either way when you call them immediately,
// but the difference matters when you defer the call (goroutines).
func loopCapture() {
	funcs := make([]func(), 3)
	for i := range 3 {
		funcs[i] = func() { fmt.Print(i, " ") }
	}
	// Go 1.22+: each i is a distinct variable → prints "0 1 2"
	for _, f := range funcs {
		f()
	}
	fmt.Println()
}

// manualFix shows the pre-1.22 workaround: shadow i with a new variable
// scoped to the loop body. Works in all Go versions.
func loopCaptureFixed() {
	funcs := make([]func(), 3)
	for i := range 3 {
		i := i // new variable per iteration
		funcs[i] = func() { fmt.Print(i, " ") }
	}
	for _, f := range funcs {
		f()
	}
	fmt.Println()
}

// --- Goroutine capture ---

// goroutineCapture shows the same issue with goroutines, which is
// dangerous because the goroutine runs after the loop finishes.
// Pre-1.22: all goroutines could print 5 (the final value).
// Go 1.22: correct, because each iteration has its own n.
func goroutineCapture() {
	var wg sync.WaitGroup
	results := make([]int, 5)

	for n := range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[n] = n * n // Go 1.22: n is this iteration's n
		}()
	}
	wg.Wait()
	fmt.Println("goroutine results:", results)
}

// goroutineCaptureExplicit passes n as an argument rather than capturing.
// This works in all Go versions and is explicit about the intent.
func goroutineCaptureExplicit() {
	var wg sync.WaitGroup
	results := make([]int, 5)

	for n := range 5 {
		wg.Add(1)
		go func(n int) { // n is a parameter, not a captured variable
			defer wg.Done()
			results[n] = n * n
		}(n)
	}
	wg.Wait()
	fmt.Println("explicit arg results:", results)
}

// --- Shared mutable state ---

// sharedCounter shows two closures sharing the same captured variable.
// This is intentional — both closures operate on the same counter.
func sharedMutableState() {
	count := 0
	inc := func() { count++ }
	dec := func() { count-- }
	get := func() int { return count }

	inc()
	inc()
	inc()
	dec()
	fmt.Println("shared count:", get()) // 2
}

// --- Capture of pointer vs value ---

// capturePointer illustrates that modifying a captured variable through
// a pointer affects all closures that captured the same variable.
func capturePointer() {
	x := 10
	read := func() int { return x }
	write := func(v int) { x = v }

	fmt.Println("before:", read())
	write(42)
	fmt.Println("after:", read()) // 42 — same x
}

func main() {
	fmt.Println("=== loop capture (Go 1.22: per-iteration var) ===")
	loopCapture()

	fmt.Println("=== manual fix (i := i shadow) ===")
	loopCaptureFixed()

	fmt.Println()
	fmt.Println("=== goroutine capture ===")
	goroutineCapture()
	goroutineCaptureExplicit()

	fmt.Println()
	fmt.Println("=== shared mutable state ===")
	sharedMutableState()

	fmt.Println()
	fmt.Println("=== pointer capture ===")
	capturePointer()
}
