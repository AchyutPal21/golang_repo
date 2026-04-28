// FILE: book/part2_core_language/chapter22_interfaces/examples/01_interface_basics/main.go
// CHAPTER: 22 — Interfaces: Go's Killer Feature
// TOPIC: Interface declaration, implicit satisfaction, the empty interface,
//        type assertions, interface composition, fmt.Stringer.
//
// Run (from the chapter folder):
//   go run ./examples/01_interface_basics

package main

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// --- Interface declaration ---

type Shape interface {
	Area() float64
	Perimeter() float64
}

// Implicit satisfaction: no "implements" keyword.
// A type satisfies an interface if it has all the required methods.

type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circle) String() string     { return fmt.Sprintf("Circle(r=%.2f)", c.Radius) }

type Rect struct{ W, H float64 }

func (r Rect) Area() float64      { return r.W * r.H }
func (r Rect) Perimeter() float64 { return 2 * (r.W + r.H) }
func (r Rect) String() string     { return fmt.Sprintf("Rect(%.2fx%.2f)", r.W, r.H) }

func printShape(s Shape) {
	fmt.Printf("%-25s area=%.2f  perim=%.2f\n", fmt.Sprint(s), s.Area(), s.Perimeter())
}

// --- Interface composition ---

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type ReadWriter interface {
	Reader
	Writer
}

// --- any (empty interface) ---

func describe(v any) {
	fmt.Printf("type=%-20T value=%v\n", v, v)
}

// --- Type assertion ---

func processShape(s Shape) {
	// Try to get more specific type information.
	if c, ok := s.(Circle); ok {
		fmt.Printf("it's a circle with radius %.2f\n", c.Radius)
		return
	}
	if r, ok := s.(Rect); ok {
		fmt.Printf("it's a rect %gx%g\n", r.W, r.H)
		return
	}
	fmt.Println("unknown shape")
}

// --- Type switch ---

func classify(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case int:
		return fmt.Sprintf("int(%d)", x)
	case float64:
		return fmt.Sprintf("float64(%.2f)", x)
	case string:
		return fmt.Sprintf("string(%q)", x)
	case bool:
		return fmt.Sprintf("bool(%v)", x)
	case []int:
		return fmt.Sprintf("[]int(len=%d)", len(x))
	default:
		return fmt.Sprintf("unknown(%T)", x)
	}
}

func main() {
	// --- interface polymorphism ---
	shapes := []Shape{
		Circle{Radius: 3},
		Rect{W: 4, H: 5},
		Circle{Radius: 1},
	}
	for _, s := range shapes {
		printShape(s)
	}

	fmt.Println()

	// --- fmt.Stringer: the most important interface to implement ---
	// Circle already has String() — fmt uses it automatically
	fmt.Println(Circle{Radius: 7})

	fmt.Println()

	// --- io.Reader/Writer composition ---
	buf := &strings.Builder{}
	var w io.Writer = buf
	fmt.Fprintln(w, "hello via io.Writer")
	fmt.Print(buf.String())

	fmt.Println()

	// --- any ---
	describe(42)
	describe("hello")
	describe([]int{1, 2, 3})
	describe(Circle{Radius: 2})
	describe(nil)

	fmt.Println()

	// --- type assertion ---
	processShape(Circle{Radius: 5})
	processShape(Rect{W: 3, H: 4})

	fmt.Println()

	// --- type switch ---
	values := []any{nil, 42, 3.14, "go", true, []int{1, 2}}
	for _, v := range values {
		fmt.Printf("classify(%v) → %s\n", v, classify(v))
	}
}
