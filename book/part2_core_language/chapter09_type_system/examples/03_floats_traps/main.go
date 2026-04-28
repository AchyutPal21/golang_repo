// FILE: book/part2_core_language/chapter09_type_system/examples/03_floats_traps/main.go
// CHAPTER: 09 — The Type System
// TOPIC: The four IEEE-754 traps everyone hits.
//
// Run (from the chapter folder):
//   go run ./examples/03_floats_traps

package main

import (
	"fmt"
	"math"
)

func main() {
	// ─── Trap 1: 0.1 + 0.2 != 0.3 ────────────────────────────────────────
	a := 0.1 + 0.2
	fmt.Printf("Trap 1: 0.1 + 0.2 = %.17f\n", a)
	fmt.Printf("        0.1 + 0.2 == 0.3? %v\n", a == 0.3)
	fmt.Printf("        within epsilon?   %v\n", math.Abs(a-0.3) < 1e-9)

	// ─── Trap 2: NaN != NaN ──────────────────────────────────────────────
	nan := math.NaN()
	fmt.Printf("\nTrap 2: NaN == NaN? %v (always false — IEEE-754 rule)\n", nan == nan)
	fmt.Printf("        math.IsNaN(nan)? %v\n", math.IsNaN(nan))

	// ─── Trap 3: divide by zero gives Inf, not panic ─────────────────────
	//
	// Note: literal 1.0/0.0 is a COMPILE error (Go's compiler refuses it).
	// You only get the IEEE-754 behavior at runtime, when the zero is in a
	// variable. We use intermediate variables to demonstrate.
	zero := 0.0
	posInf := 1.0 / zero
	negInf := -1.0 / zero
	weirdNaN := zero / zero
	fmt.Printf("\nTrap 3: 1.0/zero = %v\n", posInf)
	fmt.Printf("        -1.0/zero = %v\n", negInf)
	fmt.Printf("        zero/zero = %v (NaN, not panic)\n", weirdNaN)
	fmt.Printf("        (Integer 1/0 IS a panic; floats are different.)\n")

	// ─── Trap 4: money as float64 ────────────────────────────────────────
	var balance float64
	for i := 0; i < 10; i++ {
		balance += 0.1 // add 10 cents, ten times
	}
	fmt.Printf("\nTrap 4: 0.1 added 10 times = %.17f\n", balance)
	fmt.Printf("        That's $%.4f instead of $1.0000.\n", balance)
	fmt.Printf("        Imagine multiplying this error across millions of transactions.\n")

	// ─── The fix: integer cents ──────────────────────────────────────────
	var cents int64
	for i := 0; i < 10; i++ {
		cents += 10 // 10 cents
	}
	fmt.Printf("        Integer cents: %d cents = $%d.%02d (exact)\n",
		cents, cents/100, cents%100)
}
