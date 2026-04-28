// FILE: book/part2_core_language/chapter11_operators/examples/02_bitwise_patterns/main.go
// CHAPTER: 11 — Operators
// TOPIC: Bit-flag set/clear/test/toggle, power-of-2 modulo, XOR swap.
//
// Run (from the chapter folder):
//   go run ./examples/02_bitwise_patterns

package main

import "fmt"

type Perm uint8

const (
	PermRead Perm = 1 << iota
	PermWrite
	PermExec
	PermDelete
)

func main() {
	// ─── Set / clear / test / toggle ──────────────────────────────────────
	var p Perm
	p |= PermRead | PermWrite
	fmt.Printf("after set R+W:  %04b (Has Read=%v, Has Exec=%v)\n",
		p, p&PermRead == PermRead, p&PermExec == PermExec)

	p &^= PermWrite // clear write
	fmt.Printf("after clear W:  %04b (Has Write=%v)\n",
		p, p&PermWrite == PermWrite)

	p ^= PermExec // toggle exec
	fmt.Printf("after toggle X: %04b (Has Exec=%v)\n",
		p, p&PermExec == PermExec)

	p ^= PermExec // toggle again — back off
	fmt.Printf("toggle X again: %04b (Has Exec=%v)\n",
		p, p&PermExec == PermExec)

	// ─── Power-of-2 modulo via & ──────────────────────────────────────────
	fmt.Println("\n=== Ring buffer index trick ===")
	const N = 8 // must be power of 2 for the trick to work
	for i := 0; i < 12; i++ {
		mod := i % N        // generic modulo (uses divide)
		mask := i & (N - 1) // bitwise — same answer, faster on most CPUs
		fmt.Printf("  i=%2d   i%%%d=%d   i&(N-1)=%d\n", i, N, mod, mask)
	}

	// ─── XOR swap (cute, never use in production) ─────────────────────────
	a, b := 5, 9
	a ^= b
	b ^= a
	a ^= b
	fmt.Printf("\n=== XOR swap (5 ⊕ 9 → swapped) ===\n  a=%d b=%d\n", a, b)
	fmt.Println("(Don't actually use this — `a, b = b, a` is faster and clearer.)")
}
