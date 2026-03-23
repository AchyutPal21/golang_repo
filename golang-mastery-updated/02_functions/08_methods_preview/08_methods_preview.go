// FILE: 02_functions/08_methods_preview.go
// TOPIC: Methods — function vs method, receivers, method sets (preview for Module 03)
//
// Run: go run 02_functions/08_methods_preview.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   In Go, a method is just a function with a special first argument called a
//   receiver. There's no class keyword, no this/self — just a receiver.
//   Understanding that methods ARE functions unlocks powerful patterns like
//   method expressions and method values.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math"
)

type Circle struct {
	Radius float64
}

// Method on Circle — value receiver
// Syntax: func (receiver Type) MethodName(params) returnType
// 'c' is a COPY of the Circle when this method is called.
func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

// Pointer receiver — can modify the original
func (c *Circle) Scale(factor float64) {
	c.Radius *= factor // modifies the actual Circle, not a copy
}

// String method — makes Circle implement fmt.Stringer
func (c Circle) String() string {
	return fmt.Sprintf("Circle(r=%.2f)", c.Radius)
}

// The SAME thing written as a plain function (equivalent to the method above):
func CircleArea(c Circle) float64 {
	return math.Pi * c.Radius * c.Radius
}

type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Methods Preview")
	fmt.Println("════════════════════════════════════════")

	c := Circle{Radius: 5}

	fmt.Printf("\n── Calling methods ──\n")
	fmt.Printf("  %v\n", c)
	fmt.Printf("  Area:      %.4f\n", c.Area())
	fmt.Printf("  Perimeter: %.4f\n", c.Perimeter())

	// Pointer receiver — must have addressable variable or pointer
	c.Scale(2) // Go automatically takes &c for pointer receiver calls
	fmt.Printf("  After Scale(2): %v, Area=%.4f\n", c, c.Area())

	// ── METHOD EXPRESSION — convert method to a plain function ──────────
	// Method expression: Type.MethodName → becomes a function with receiver as first arg
	areaFn := Circle.Area // type: func(Circle) float64
	fmt.Printf("\n── Method expression ──\n")
	fmt.Printf("  Circle.Area as function: %.4f\n", areaFn(Circle{Radius: 3}))

	// ── METHOD VALUE — bind receiver, get a zero-arg function ───────────
	// Method value: instance.Method → bound to that specific instance
	c2 := Circle{Radius: 7}
	boundArea := c2.Area // type: func() float64, receiver already bound to c2
	fmt.Printf("\n── Method value (bound receiver) ──\n")
	fmt.Printf("  c2.Area bound: %.4f\n", boundArea())

	// ── METHODS ARE JUST FUNCTIONS ──────────────────────────────────────
	fmt.Printf("\n── Method vs plain function (identical result) ──\n")
	c3 := Circle{Radius: 4}
	fmt.Printf("  c3.Area()          = %.4f  (method call)\n", c3.Area())
	fmt.Printf("  CircleArea(c3)     = %.4f  (plain function)\n", CircleArea(c3))
	fmt.Printf("  Circle.Area(c3)    = %.4f  (method expression)\n", Circle.Area(c3))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  method = function with a receiver argument")
	fmt.Println("  value receiver  → gets a copy (use for read-only, small structs)")
	fmt.Println("  pointer receiver → modifies original (use for mutation, large structs)")
	fmt.Println("  Method expression: Type.Method → func(Type, args)")
	fmt.Println("  Method value: instance.Method → func(args)  (receiver pre-bound)")
	fmt.Println("  Full deep-dive: Module 03")
}
