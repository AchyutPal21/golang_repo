// EXERCISE 10.2 — Replace panicky assertions with comma-ok.
//
// Run (from the chapter folder):
//   go run ./exercises/01_safe_assert
//
// The program below uses panicky assertions in places where the input is
// untrusted. Find each one, convert to the safe form, and handle the
// false case gracefully.

package main

import "fmt"

func describe(x any) {
	// FIXED: was `n := x.(int)`, now safe.
	if n, ok := x.(int); ok {
		fmt.Printf("int: %d\n", n)
		return
	}
	if s, ok := x.(string); ok {
		fmt.Printf("string: %q\n", s)
		return
	}
	fmt.Printf("unknown: %v (%T)\n", x, x)
}

func main() {
	for _, v := range []any{42, "hello", 3.14, true, nil} {
		describe(v)
	}
}
