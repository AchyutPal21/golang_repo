// FILE: 01_fundamentals/07_operators.go
// TOPIC: Operators — Arithmetic, Comparison, Logical, Bitwise, Address/Deref
//
// Run: go run 01_fundamentals/07_operators.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Go's operators are similar to C but with key differences: no operator
//   overloading, no ternary operator (?:), and careful rules about which
//   types each operator works on. Bitwise operators are essential for
//   systems programming, flag handling, and performance-critical code.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Operators")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// ARITHMETIC OPERATORS
	// ─────────────────────────────────────────────────────────────────────
	//
	// +   addition
	// -   subtraction
	// *   multiplication
	// /   division (integer division truncates toward zero)
	// %   modulo (remainder) — sign follows the DIVIDEND, not divisor
	// ++  increment (statement only, NOT expression — no x = y++ !)
	// --  decrement (statement only)
	//
	// NOTE: ++ and -- are STATEMENTS in Go, not expressions.
	//   Go deliberately removed the "x = y++" pattern to prevent bugs.
	//   You CANNOT do: a[i++], fmt.Println(i++), etc.

	a, b := 17, 5
	fmt.Printf("\n── Arithmetic (a=%d, b=%d) ──\n", a, b)
	fmt.Printf("  a + b  = %d\n", a+b)
	fmt.Printf("  a - b  = %d\n", a-b)
	fmt.Printf("  a * b  = %d\n", a*b)
	fmt.Printf("  a / b  = %d  ← integer division truncates (17/5=3, not 3.4)\n", a/b)
	fmt.Printf("  a %% b  = %d  ← remainder (17 = 5*3 + 2)\n", a%b)

	// Integer division truncates TOWARD ZERO (not floor):
	fmt.Printf("  -7 / 2  = %d  ← truncates to -3 (not -4!)\n", -7/2)
	fmt.Printf("  -7 %% 2  = %d  ← sign follows dividend (-7), so -1\n", -7%2)
	fmt.Printf("   7 %% -2 = %d  ← sign follows dividend (7), so 1\n", 7%-2)

	// For float division:
	af, bf := float64(a), float64(b)
	fmt.Printf("  float: %.1f / %.1f = %.4f\n", af, bf, af/bf)

	// ++ and -- as statements:
	c := 10
	c++  // c = 11 (statement, not expression)
	c--  // c = 10
	fmt.Printf("  c after c++ then c-- = %d\n", c)

	// ─────────────────────────────────────────────────────────────────────
	// COMPARISON OPERATORS — Always return bool
	// ─────────────────────────────────────────────────────────────────────
	//
	// ==  equal
	// !=  not equal
	// <   less than
	// <=  less than or equal
	// >   greater than
	// >=  greater than or equal
	//
	// Both operands must be the SAME TYPE. No implicit conversion.
	// You can compare: all numeric types (with same type), bool, string,
	//   pointer (to same type), interface (by dynamic type+value), struct
	//   (if all fields are comparable), array (if element type is comparable).
	//
	// NOT comparable with == : slices, maps, functions
	//   (use reflect.DeepEqual or loops for those)

	x, y := 10, 20
	fmt.Printf("\n── Comparison (x=%d, y=%d) ──\n", x, y)
	fmt.Printf("  x == y : %v\n", x == y)
	fmt.Printf("  x != y : %v\n", x != y)
	fmt.Printf("  x <  y : %v\n", x < y)
	fmt.Printf("  x <= y : %v\n", x <= y)
	fmt.Printf("  x >  y : %v\n", x > y)
	fmt.Printf("  x >= y : %v\n", x >= y)

	// Struct comparison (all fields must be comparable):
	type Point struct{ X, Y int }
	p1 := Point{1, 2}
	p2 := Point{1, 2}
	p3 := Point{3, 4}
	fmt.Printf("  Point{1,2} == Point{1,2} : %v\n", p1 == p2)
	fmt.Printf("  Point{1,2} == Point{3,4} : %v\n", p1 == p3)

	// ─────────────────────────────────────────────────────────────────────
	// LOGICAL OPERATORS
	// ─────────────────────────────────────────────────────────────────────
	//
	// &&  logical AND (short-circuit: if left is false, right NOT evaluated)
	// ||  logical OR  (short-circuit: if left is true, right NOT evaluated)
	// !   logical NOT
	//
	// SHORT-CIRCUIT EVALUATION:
	//   In "a && b": if a is false, b is never evaluated.
	//   In "a || b": if a is true, b is never evaluated.
	//   This is useful for nil checks:
	//     if ptr != nil && ptr.Value > 0 { ... }  ← safe: ptr.Value only accessed if non-nil
	//
	// NOTE: Go has NO ternary operator (?:).
	// There's no "x = condition ? a : b" — use an if statement.
	// WHY? The Go team felt ternary encourages complex one-liners that hurt readability.

	t, f := true, false
	fmt.Printf("\n── Logical ──\n")
	fmt.Printf("  true && true  = %v\n", t && t)
	fmt.Printf("  true && false = %v\n", t && f)
	fmt.Printf("  true || false = %v\n", t || f)
	fmt.Printf("  false|| false = %v\n", f || f)
	fmt.Printf("  !true         = %v\n", !t)
	fmt.Printf("  !false        = %v\n", !f)

	// Short-circuit demo:
	fmt.Println("\n  Short-circuit evaluation:")
	sideEffect := false
	setSideEffect := func() bool {
		sideEffect = true
		return true
	}
	_ = (false && setSideEffect())  // setSideEffect() is NEVER called
	fmt.Printf("    false && f(): sideEffect=%v (f() not called)\n", sideEffect)
	_ = (true || setSideEffect())   // setSideEffect() is NEVER called
	fmt.Printf("    true  || f(): sideEffect=%v (f() not called)\n", sideEffect)
	_ = (true && setSideEffect())   // setSideEffect() IS called
	fmt.Printf("    true  && f(): sideEffect=%v (f() was called)\n", sideEffect)

	// ─────────────────────────────────────────────────────────────────────
	// BITWISE OPERATORS — Working at the bit level
	// ─────────────────────────────────────────────────────────────────────
	//
	// &   bitwise AND         (both bits must be 1)
	// |   bitwise OR          (at least one bit must be 1)
	// ^   bitwise XOR         (bits must differ) / bitwise NOT (unary)
	// &^  bit clear (AND NOT) (clear bits: a &^ b clears bits in a that are set in b)
	// <<  left shift          (multiply by 2^n)
	// >>  right shift         (divide by 2^n, sign-extended for signed types)
	//
	// USE CASES:
	//   & : check if a flag is set, mask bits
	//   | : set a flag, combine flags
	//   ^ : toggle a flag, compute differences
	//   &^: clear specific flags
	//   <<: multiply by power of 2 efficiently, build bit masks
	//   >>: divide by power of 2 efficiently, extract high bits

	u, v := uint8(0b10110110), uint8(0b11001100)  // binary literals (Go 1.13+)
	fmt.Printf("\n── Bitwise (u=%08b=%d, v=%08b=%d) ──\n", u, u, v, v)
	fmt.Printf("  u &  v = %08b = %d  (AND: bits set in both)\n", u&v, u&v)
	fmt.Printf("  u |  v = %08b = %d  (OR: bits set in either)\n", u|v, u|v)
	fmt.Printf("  u ^  v = %08b = %d  (XOR: bits set in exactly one)\n", u^v, u^v)
	fmt.Printf("  u &^ v = %08b = %d  (AND NOT: clear v's bits from u)\n", u&^v, u&^v)
	fmt.Printf("  u << 2 = %08b = %d  (left shift 2 = multiply by 4)\n", u<<2, u<<2)
	fmt.Printf("  u >> 2 = %08b = %d  (right shift 2 = divide by 4)\n", u>>2, u>>2)

	// Bitwise NOT (unary ^):
	fmt.Printf("  ^u     = %08b = %d  (NOT: flip all bits)\n", ^u, ^u)

	// Practical: bit flag manipulation
	fmt.Println("\n  Practical bit flags:")
	var flags uint8 = 0
	const (
		FlagA uint8 = 1 << 0  // 00000001
		FlagB uint8 = 1 << 1  // 00000010
		FlagC uint8 = 1 << 2  // 00000100
	)
	flags |= FlagA            // set FlagA
	flags |= FlagC            // set FlagC
	fmt.Printf("    After setting A and C: %08b\n", flags)
	fmt.Printf("    FlagA set? %v\n", flags&FlagA != 0)
	fmt.Printf("    FlagB set? %v\n", flags&FlagB != 0)
	flags &^= FlagA           // clear FlagA
	fmt.Printf("    After clearing A:      %08b\n", flags)
	flags ^= FlagC            // toggle FlagC
	fmt.Printf("    After toggling C:      %08b\n", flags)

	// ─────────────────────────────────────────────────────────────────────
	// ADDRESS & DEREFERENCE OPERATORS
	// ─────────────────────────────────────────────────────────────────────
	//
	// &  address-of operator: &x gives a pointer to x
	// *  dereference operator: *p gives the value that pointer p points to
	//
	// These are covered in depth in 10_pointers.go.
	// Brief preview here for completeness.

	n := 42
	ptr := &n   // ptr is *int, holds the memory address of n
	fmt.Printf("\n── Address & Dereference ──\n")
	fmt.Printf("  n   = %d\n", n)
	fmt.Printf("  &n  = %p  (memory address)\n", ptr)
	fmt.Printf("  *ptr = %d  (value at that address)\n", *ptr)
	*ptr = 100  // modify n through the pointer
	fmt.Printf("  After *ptr=100, n=%d\n", n)

	// ─────────────────────────────────────────────────────────────────────
	// OPERATOR PRECEDENCE (high to low)
	// ─────────────────────────────────────────────────────────────────────
	//
	// 5 (highest): *  /  %  <<  >>  &  &^
	// 4:           +  -  |  ^
	// 3:           ==  !=  <  <=  >  >=
	// 2:           &&
	// 1 (lowest):  ||
	//
	// Unary operators (+, -, !, ^, *, &) have higher precedence than all binary.
	//
	// TIP: When in doubt, use parentheses. They cost nothing at runtime
	// but make your intent clear.

	fmt.Printf("\n── Precedence example ──\n")
	fmt.Printf("  2 + 3 * 4     = %d  (* before +)\n", 2+3*4)
	fmt.Printf("  (2 + 3) * 4   = %d  (parens override)\n", (2+3)*4)
	fmt.Printf("  3&1 == 1      = %v  (&  before ==)\n", 3&1 == 1)
	fmt.Printf("  true || false && false = %v  (&& before ||)\n", true || false && false)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Arithmetic: + - * / % ++ --  (++ and -- are STATEMENTS, not expressions)")
	fmt.Println("  Comparison: == != < <= > >=   (returns bool, same types only)")
	fmt.Println("  Logical: && || !              (short-circuit evaluation)")
	fmt.Println("  Bitwise: & | ^ &^ << >>       (essential for flags and systems code)")
	fmt.Println("  No ternary operator in Go — use if/else")
	fmt.Println("  When precedence is unclear, use parentheses")
}
