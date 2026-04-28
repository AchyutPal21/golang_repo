// FILE: book/part2_core_language/chapter11_operators/examples/01_operator_table/main.go
// CHAPTER: 11 — Operators, Precedence, Bitwise
// TOPIC: Every operator, applied to representative values.
//
// Run (from the chapter folder):
//   go run ./examples/01_operator_table

package main

import "fmt"

func main() {
	a, b := 12, 7

	fmt.Println("=== Arithmetic on int (a=12, b=7) ===")
	fmt.Printf("  a + b = %d\n", a+b)
	fmt.Printf("  a - b = %d\n", a-b)
	fmt.Printf("  a * b = %d\n", a*b)
	fmt.Printf("  a / b = %d   (integer divide truncates toward zero)\n", a/b)
	fmt.Printf("  a %% b = %d   (remainder; sign matches dividend)\n", a%b)
	fmt.Printf("  -a    = %d\n", -a)
	fmt.Printf("  -7 %% 3 = %d (sign matches -7)\n", -7%3)

	fmt.Println("\n=== Comparison ===")
	fmt.Printf("  a == b: %v\n", a == b)
	fmt.Printf("  a != b: %v\n", a != b)
	fmt.Printf("  a > b:  %v\n", a > b)
	fmt.Printf("  a < b:  %v\n", a < b)

	fmt.Println("\n=== Logical (short-circuit) ===")
	fmt.Printf("  true && false = %v\n", true && false)
	fmt.Printf("  true || false = %v\n", true || false)
	fmt.Printf("  !true         = %v\n", !true)

	fmt.Println("\n=== Bitwise on uint8 (a=0b1100=12, b=0b0111=7) ===")
	au, bu := uint8(12), uint8(7)
	fmt.Printf("  a & b   = %08b = %d (AND)\n", au&bu, au&bu)
	fmt.Printf("  a | b   = %08b = %d (OR)\n", au|bu, au|bu)
	fmt.Printf("  a ^ b   = %08b = %d (XOR)\n", au^bu, au^bu)
	fmt.Printf("  a &^ b  = %08b = %d (BIT CLEAR — keep bits in a NOT in b)\n", au&^bu, au&^bu)
	fmt.Printf("  ^a      = %08b = %d (NOT)\n", ^au, ^au)
	fmt.Printf("  a << 1  = %08b = %d (left shift)\n", au<<1, au<<1)
	fmt.Printf("  a >> 1  = %08b = %d (right shift)\n", au>>1, au>>1)

	fmt.Println("\n=== Compound assignment ===")
	x := 10
	x += 5
	fmt.Printf("  x += 5  → %d\n", x)
	x *= 2
	fmt.Printf("  x *= 2  → %d\n", x)
	x &^= 0b0001 // clear lowest bit
	fmt.Printf("  x &^= 1 → %d (bit-clear assignment)\n", x)

	fmt.Println("\n=== Statement-only ++ and -- ===")
	y := 10
	y++
	fmt.Printf("  y++ → %d  (NOTE: `j = i++` does NOT compile in Go)\n", y)

	fmt.Println("\n=== String + (concatenation only; allocates a new string) ===")
	s1 := "hello, "
	s2 := "world"
	fmt.Printf("  %q + %q = %q\n", s1, s2, s1+s2)
}
