// FILE: book/part2_core_language/chapter15_defer_panic_recover/examples/01_defer_mechanics/main.go
// CHAPTER: 15 — defer, panic, recover
// TOPIC: defer execution order (LIFO), argument evaluation timing,
//        defer with named returns, defer cost.
//
// Run (from the chapter folder):
//   go run ./examples/01_defer_mechanics

package main

import "fmt"

// lifoOrder shows that deferred calls run last-in, first-out.
func lifoOrder() {
	fmt.Println("lifoOrder called")
	defer fmt.Println("  defer 1 — registered first, runs last")
	defer fmt.Println("  defer 2")
	defer fmt.Println("  defer 3 — registered last, runs first")
	fmt.Println("lifoOrder returning")
}

// argEvaluation shows that defer arguments are evaluated immediately,
// not when the deferred call runs.
func argEvaluation() {
	x := 10
	defer fmt.Println("deferred x =", x) // x evaluated NOW: captures 10
	x = 99
	fmt.Println("live x =", x) // 99
	// Deferred print will still say 10.
}

// deferLoop shows defer inside a loop — each iteration registers a new
// deferred call. All run when the function returns (not per-iteration).
func deferLoop() {
	for i := range 3 {
		defer fmt.Printf("  defer loop i=%d\n", i)
	}
	fmt.Println("deferLoop returning (defers run after this line)")
}

// withNamedReturn shows defer interacting with named return values.
// A deferred function can read and modify named return values.
func withNamedReturn() (result string) {
	defer func() {
		result = "modified by defer: " + result
	}()
	result = "original"
	return // returns "modified by defer: original"
}

// cleanup simulates the canonical defer use-case: ensure a resource
// is released regardless of how the function exits.
func cleanup(name string) {
	fmt.Println("opening", name)
	defer fmt.Println("closing", name) // always runs
	fmt.Println("using", name)
}

// deferReturnValue shows deferred function can change named return.
func mustPositive(n int) (result int, err error) {
	defer func() {
		if result < 0 {
			result = 0
			err = fmt.Errorf("clamped negative %d to 0", n)
		}
	}()
	result = n
	return
}

func main() {
	fmt.Println("=== LIFO order ===")
	lifoOrder()

	fmt.Println()
	fmt.Println("=== argument evaluation ===")
	argEvaluation()

	fmt.Println()
	fmt.Println("=== defer in loop ===")
	deferLoop()

	fmt.Println()
	fmt.Println("=== named return ===")
	fmt.Println("result:", withNamedReturn())

	fmt.Println()
	fmt.Println("=== cleanup pattern ===")
	cleanup("file.txt")

	fmt.Println()
	fmt.Println("=== defer modifies return ===")
	r1, err1 := mustPositive(5)
	fmt.Printf("mustPositive(5)  → %d, %v\n", r1, err1)
	r2, err2 := mustPositive(-3)
	fmt.Printf("mustPositive(-3) → %d, %v\n", r2, err2)
}
