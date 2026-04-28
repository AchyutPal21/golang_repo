// FILE: book/part2_core_language/chapter21_methods/examples/01_method_basics/main.go
// CHAPTER: 21 — Methods: Functions With Receivers
// TOPIC: Value vs pointer receivers, method expressions, method values,
//        methods on non-struct types, nil receiver.
//
// Run (from the chapter folder):
//   go run ./examples/01_method_basics

package main

import (
	"fmt"
	"math"
)

// --- Value receiver: does not modify the receiver ---

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

func (c Circle) String() string {
	return fmt.Sprintf("Circle(r=%.2f)", c.Radius)
}

// --- Pointer receiver: can modify the receiver ---

type Counter struct {
	count int
}

func (c *Counter) Increment() { c.count++ }
func (c *Counter) Reset()     { c.count = 0 }
func (c *Counter) Value() int { return c.count }

// String on *Counter (note: pointer receiver satisfies fmt.Stringer)
func (c *Counter) String() string {
	return fmt.Sprintf("Counter(%d)", c.count)
}

// --- Methods on non-struct types ---

// Duration is a named type based on float64.
type Duration float64

const (
	Second Duration = 1
	Minute          = 60 * Second
	Hour            = 60 * Minute
)

func (d Duration) String() string {
	h := int(d / Hour)
	m := int(d/Minute) % 60
	s := int(d) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// --- Nil receiver ---

// Tree shows that a nil pointer receiver is valid and useful.
type Tree struct {
	Value       int
	Left, Right *Tree
}

func (t *Tree) Sum() int {
	if t == nil {
		return 0
	}
	return t.Value + t.Left.Sum() + t.Right.Sum()
}

func (t *Tree) Insert(v int) *Tree {
	if t == nil {
		return &Tree{Value: v}
	}
	if v < t.Value {
		t.Left = t.Left.Insert(v)
	} else {
		t.Right = t.Right.Insert(v)
	}
	return t
}

func main() {
	// --- value receiver ---
	c := Circle{Radius: 5}
	fmt.Printf("area=%.2f  perimeter=%.2f\n", c.Area(), c.Perimeter())
	fmt.Println(c) // calls String()

	// Value receiver: method works on pointer too (Go auto-dereferences)
	cp := &Circle{Radius: 3}
	fmt.Printf("pointer: area=%.2f\n", cp.Area())

	fmt.Println()

	// --- pointer receiver ---
	var ctr Counter
	ctr.Increment()
	ctr.Increment()
	ctr.Increment()
	fmt.Println(ctr.Value()) // 3
	fmt.Println(&ctr)        // String() — need pointer for fmt.Stringer

	// Pointer receiver: method works on addressable value (Go takes address)
	ctr.Reset() // Go: (&ctr).Reset()
	fmt.Println(ctr.Value()) // 0

	fmt.Println()

	// --- method on non-struct ---
	var t Duration = 1*Hour + 23*Minute + 45*Second
	fmt.Println("duration:", t)

	fmt.Println()

	// --- method expression: func with explicit receiver as first arg ---
	areaFn := Circle.Area      // type: func(Circle) float64
	fmt.Println("method expr:", areaFn(Circle{Radius: 4}))

	incrFn := (*Counter).Increment // type: func(*Counter)
	var c2 Counter
	incrFn(&c2)
	incrFn(&c2)
	fmt.Println("method expr pointer:", c2.Value())

	// --- method value: bound to a specific receiver ---
	c3 := Circle{Radius: 7}
	boundArea := c3.Area // type: func() float64 — c3 captured
	fmt.Println("method value:", boundArea())

	fmt.Println()

	// --- nil receiver ---
	var root *Tree
	root = root.Insert(5)
	root = root.Insert(3)
	root = root.Insert(8)
	root = root.Insert(1)
	fmt.Println("tree sum:", root.Sum()) // 5+3+8+1 = 17
}
