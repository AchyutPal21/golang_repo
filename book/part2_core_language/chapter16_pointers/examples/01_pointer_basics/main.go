// FILE: book/part2_core_language/chapter16_pointers/examples/01_pointer_basics/main.go
// CHAPTER: 16 — Pointers and Memory Addressing
// TOPIC: & and *, nil pointer, pointer to struct, pointer comparison,
//        new(), value vs pointer semantics.
//
// Run (from the chapter folder):
//   go run ./examples/01_pointer_basics

package main

import "fmt"

// increment mutates n via pointer — this is the only way to modify the
// caller's variable from inside a function.
func increment(n *int) {
	*n++
}

// swap exchanges two ints in the caller's scope.
func swap(a, b *int) {
	*a, *b = *b, *a
}

// zero sets the pointed-to integer to zero.
func zero(p *int) {
	if p == nil {
		return // nil-safe
	}
	*p = 0
}

// Point is a simple struct used to show pointer-to-struct usage.
type Point struct{ X, Y float64 }

func (p *Point) Scale(factor float64) {
	p.X *= factor
	p.Y *= factor
}

func (p Point) String() string {
	return fmt.Sprintf("(%.1f, %.1f)", p.X, p.Y)
}

// newPoint allocates a Point on the heap and returns a pointer.
// Go's escape analysis decides whether the allocation is stack or heap.
func newPoint(x, y float64) *Point {
	return &Point{X: x, Y: y} // address-of composite literal
}

// -- Pointer comparison --

func sameVar(a, b *int) bool {
	return a == b // true only if both point to the exact same variable
}

func main() {
	// -- & and * --
	x := 42
	p := &x          // p is *int
	fmt.Println("x =", x)
	fmt.Println("p =", p)   // memory address
	fmt.Println("*p =", *p) // dereference: 42

	*p = 100
	fmt.Println("after *p=100, x =", x) // 100 — same variable

	fmt.Println()

	// -- increment via pointer --
	n := 5
	increment(&n)
	increment(&n)
	fmt.Println("after 2× increment:", n) // 7

	fmt.Println()

	// -- swap --
	a, b := 10, 20
	swap(&a, &b)
	fmt.Println("after swap: a=", a, "b=", b) // a=20 b=10

	fmt.Println()

	// -- nil pointer --
	var ptr *int
	fmt.Println("nil pointer:", ptr)
	zero(ptr)                  // safe: zero checks for nil
	zero(&n)                   // n becomes 0
	fmt.Println("n after zero:", n)

	fmt.Println()

	// -- pointer-to-struct --
	pt := newPoint(3, 4)
	fmt.Println("before Scale:", pt)
	pt.Scale(2) // Go auto-dereferences: pt.Scale(2) == (*pt).Scale(2)
	fmt.Println("after Scale(2):", pt)

	// address-of struct literal
	pt2 := &Point{X: 1, Y: 1}
	fmt.Println("pt2:", pt2)

	fmt.Println()

	// -- new() --
	pn := new(int) // allocates, zero-initialises, returns *int
	fmt.Println("new(int):", *pn)
	*pn = 99
	fmt.Println("after assignment:", *pn)

	fmt.Println()

	// -- pointer comparison --
	v1, v2 := 1, 1
	fmt.Println("sameVar(&v1, &v1):", sameVar(&v1, &v1)) // true
	fmt.Println("sameVar(&v1, &v2):", sameVar(&v1, &v2)) // false (same value, different var)
}
