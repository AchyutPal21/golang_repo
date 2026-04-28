// FILE: book/part2_core_language/chapter14_closures/examples/01_closure_basics/main.go
// CHAPTER: 14 — Closures and the Capture Model
// TOPIC: What a closure is, captured variables, closures as stateful objects.
//
// Run (from the chapter folder):
//   go run ./examples/01_closure_basics

package main

import "fmt"

// makeCounter returns a closure that increments and returns a counter.
// Each call to makeCounter produces an independent counter backed by
// its own `count` variable — there is no global state.
func makeCounter() func() int {
	count := 0
	return func() int {
		count++ // captures `count` by reference, not by value
		return count
	}
}

// makeAccumulator returns a closure that accumulates a running total.
func makeAccumulator() func(int) int {
	total := 0
	return func(n int) int {
		total += n
		return total
	}
}

// makeGreeter returns a closure over a name.
// The name is captured at construction time and never changes.
func makeGreeter(name string) func(string) string {
	return func(greeting string) string {
		return greeting + ", " + name + "!"
	}
}

// toggleFactory returns a closure that alternates between two values.
func toggleFactory[T any](a, b T) func() T {
	state := false
	return func() T {
		state = !state
		if state {
			return a
		}
		return b
	}
}

func main() {
	// Two independent counters — separate closure environments.
	c1 := makeCounter()
	c2 := makeCounter()

	fmt.Println("c1:", c1(), c1(), c1()) // 1 2 3
	fmt.Println("c2:", c2(), c2())       // 1 2
	fmt.Println("c1:", c1())             // 4 — c1 is not affected by c2

	fmt.Println()

	// Accumulator
	acc := makeAccumulator()
	for _, n := range []int{10, 5, 3, 7} {
		fmt.Printf("acc += %2d → %d\n", n, acc(n))
	}

	fmt.Println()

	// Greeter — name captured once
	helloAlice := makeGreeter("Alice")
	hiBob := makeGreeter("Bob")
	fmt.Println(helloAlice("Hello"))
	fmt.Println(hiBob("Hi"))
	fmt.Println(helloAlice("Good morning")) // same closure, new greeting

	fmt.Println()

	// Toggle
	onOff := toggleFactory("ON", "OFF")
	for range 6 {
		fmt.Print(onOff(), " ")
	}
	fmt.Println()
}
