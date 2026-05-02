// FILE: book/part4_concurrency_systems/chapter44_select_timeouts_cancel/examples/01_select_basics/main.go
// CHAPTER: 44 — select / Timeouts / Cancel
// TOPIC: select mechanics, default case, pseudo-random fairness,
//        priority select, and select with nil channels.
//
// Run (from the chapter folder):
//   go run ./examples/01_select_basics

package main

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SELECT BASICS
//
// select waits on multiple channel operations simultaneously.
// When multiple cases are ready at the same time, one is chosen at random.
// If no case is ready and there is a default, default runs immediately.
// If no case is ready and there is no default, select blocks.
// ─────────────────────────────────────────────────────────────────────────────

func demoBasicSelect() {
	fmt.Println("=== Basic select ===")

	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	ch1 <- "from ch1"
	ch2 <- "from ch2"

	// Both are ready — Go picks one at random.
	for range 4 {
		select {
		case v := <-ch1:
			fmt.Printf("  case ch1: %s\n", v)
			ch1 <- "from ch1"
		case v := <-ch2:
			fmt.Printf("  case ch2: %s\n", v)
			ch2 <- "from ch2"
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEFAULT CASE — non-blocking try
// ─────────────────────────────────────────────────────────────────────────────

func demoDefault() {
	fmt.Println()
	fmt.Println("=== Default (non-blocking) ===")

	ch := make(chan int, 1)

	// Try to receive — nothing there yet.
	select {
	case v := <-ch:
		fmt.Println("  received:", v)
	default:
		fmt.Println("  nothing ready — default fired")
	}

	ch <- 42

	// Now something is there.
	select {
	case v := <-ch:
		fmt.Println("  received:", v)
	default:
		fmt.Println("  nothing ready — default fired")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRIORITY SELECT — check high-priority channel first
//
// select is randomly fair. To give one channel priority over another,
// use a nested select with a default.
// ─────────────────────────────────────────────────────────────────────────────

func demoPriority() {
	fmt.Println()
	fmt.Println("=== Priority select ===")

	high := make(chan string, 10)
	low := make(chan string, 10)

	for i := range 5 {
		high <- fmt.Sprintf("HIGH-%d", i)
		low <- fmt.Sprintf("low-%d", i)
	}

	processed := 0
	for processed < 10 {
		// Drain high-priority channel first.
		select {
		case v := <-high:
			fmt.Printf("  priority: %s\n", v)
			processed++
			continue
		default:
		}
		// Fall through to low only when high is empty.
		select {
		case v := <-high:
			fmt.Printf("  priority: %s\n", v)
		case v := <-low:
			fmt.Printf("  normal:   %s\n", v)
		}
		processed++
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SELECT WITH NIL — disabling cases
// ─────────────────────────────────────────────────────────────────────────────

func demoNilCase() {
	fmt.Println()
	fmt.Println("=== Nil channel disables case ===")

	a := make(chan int, 3)
	b := make(chan int, 3)

	for i := range 3 {
		a <- i
		b <- i + 10
	}
	close(a)
	close(b)

	var chA, chB chan int = a, b
	received := 0
	for received < 6 {
		select {
		case v, ok := <-chA:
			if !ok {
				chA = nil // disable this case
				fmt.Println("  chA exhausted")
				continue
			}
			fmt.Printf("  a: %d\n", v)
			received++
		case v, ok := <-chB:
			if !ok {
				chB = nil
				fmt.Println("  chB exhausted")
				continue
			}
			fmt.Printf("  b: %d\n", v)
			received++
		}
		if chA == nil && chB == nil {
			break
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SELECT SEND/RECEIVE MIX — select works on sends too
// ─────────────────────────────────────────────────────────────────────────────

func demoSendReceiveMix() {
	fmt.Println()
	fmt.Println("=== Send/receive mix in select ===")

	results := make(chan int, 5)
	quit := make(chan struct{})

	go func() {
		for i := range 10 {
			select {
			case results <- i * i:
				// sent successfully
			case <-quit:
				fmt.Println("  generator: quit signal received")
				return
			}
		}
		close(results)
	}()

	// Read first 5 results then signal quit.
	count := 0
	for v := range results {
		fmt.Printf("  result: %d\n", v)
		count++
		if count == 5 {
			close(quit)
			break
		}
	}
	time.Sleep(time.Millisecond) // let generator goroutine print its message
}

func main() {
	demoBasicSelect()
	demoDefault()
	demoPriority()
	demoNilCase()
	demoSendReceiveMix()
}
