// FILE: book/part2_core_language/chapter11_operators/examples/03_precedence_traps/main.go
// CHAPTER: 11 — Operators
// TOPIC: A handful of expressions where precedence is non-obvious.
//
// Run (from the chapter folder):
//   go run ./examples/03_precedence_traps

package main

import "fmt"

func main() {
	// Trap 1: & is precedence 5 (higher than ==, which is 3)
	// So `a & mask == expected` parses as `a & (mask == expected)` — WRONG?
	// Actually no: `mask == expected` is bool, and `a & bool` is a compile
	// error. But `a&mask == expected` parses as `(a&mask) == expected`,
	// which is what we want (& binds tighter than ==).
	a, mask, expected := 0b1100, 0b0100, 0b0100
	fmt.Printf("a&mask == expected → %v   (parses as (a&mask) == expected)\n",
		a&mask == expected)

	// Trap 2: && is precedence 2, || is 1. `a || b && c` is `a || (b && c)`.
	// Most people read left-to-right and get this wrong.
	x := false || true && false
	fmt.Printf("false || true && false = %v   (parses as false || (true && false))\n", x)

	// Trap 3: unary operators bind tighter than binary.
	// `-a * b` is `(-a) * b`, never `-(a*b)`. Both happen to give the same
	// answer for ints, but the order matters when you mix with parens.
	n := 3
	fmt.Printf("-n*2 = %d   (parses as (-n)*2)\n", -n*2)

	// Trap 4: shift before add. `1<<2 + 3` is `(1<<2) + 3` = 7, not `1<<(2+3)` = 32.
	// (This one is intuitive only AFTER you remember << is precedence 5,
	// like multiplication.)
	fmt.Printf("1<<2 + 3 = %d  (parses as (1<<2) + 3)\n", 1<<2+3)

	// Lesson: parenthesize anything mixing arithmetic, shifts, and
	// comparisons. `gofmt` keeps your parens.
	fmt.Println("\nWhen in doubt, parenthesize. gofmt doesn't strip parens.")
}
