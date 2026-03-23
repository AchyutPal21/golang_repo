// 04_interfaces_basics.go
//
// INTERFACES — Go's primary abstraction mechanism.
//
// An interface defines a SET OF METHOD SIGNATURES. Any type that provides
// those methods automatically satisfies the interface — no "implements"
// keyword, no explicit declaration.
//
// This is called STRUCTURAL TYPING (or "duck typing" in static form):
//   "If it walks like a duck and quacks like a duck, it IS a duck."
//
// WHY IMPLICIT INTERFACES ARE POWERFUL:
//   1. Decoupling: package A defines an interface; package B satisfies it
//      WITHOUT knowing about A's interface. No import cycle needed.
//   2. Retrofitting: you can make existing types satisfy new interfaces
//      without modifying the original type (open/closed principle).
//   3. Small surfaces: interfaces tend to stay small because you only
//      list what you actually need.
//   4. Testability: swap real implementations with mocks trivially.
//
// AN INTERFACE VALUE has two components: (type, value).
//   - type: the dynamic type (the concrete type stored in the interface)
//   - value: a pointer to the concrete value
//   - A nil interface has BOTH type and value as nil.

package main

import (
	"fmt"
	"math"
	"sort"
)

// ─── 1. Defining Interfaces ───────────────────────────────────────────────────
//
// Convention: single-method interfaces get a name ending in -er (Reader, Writer,
// Stringer, Closer). This is a strong Go idiom.

type Shape interface {
	Area() float64
	Perimeter() float64
}

// Describer has one method.
type Describer interface {
	Describe() string
}

// ─── 2. Concrete Types That Satisfy Shape ─────────────────────────────────────
//
// Neither type says "implements Shape". They just have the required methods.
// The compiler checks at the point of use.

type Rect struct {
	Width, Height float64
}

func (r Rect) Area() float64      { return r.Width * r.Height }
func (r Rect) Perimeter() float64 { return 2 * (r.Width + r.Height) }
func (r Rect) Describe() string   { return fmt.Sprintf("Rectangle(%.1fx%.1f)", r.Width, r.Height) }

type Circ struct {
	Radius float64
}

func (c Circ) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circ) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circ) Describe() string   { return fmt.Sprintf("Circle(r=%.1f)", c.Radius) }

type Triangle struct {
	A, B, C float64 // side lengths
}

func (t Triangle) Area() float64 {
	// Heron's formula
	s := (t.A + t.B + t.C) / 2
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}

func (t Triangle) Perimeter() float64 { return t.A + t.B + t.C }

// Triangle does NOT implement Describer — and that's fine.
// It only needs to implement Shape to be used as a Shape.

// ─── 3. Coding to an Interface ────────────────────────────────────────────────
//
// Functions that accept interfaces work with ANY type that satisfies them.
// This is Go's polymorphism mechanism.

func printShapeInfo(s Shape) {
	// Inside here, we don't know (or care) whether s is a Rect, Circ, or Triangle.
	fmt.Printf("  Area: %.4f  |  Perimeter: %.4f\n", s.Area(), s.Perimeter())
}

func totalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

// largestShape returns the shape with the greatest area.
// Works for ANY Shape implementation — present and future.
func largestShape(shapes []Shape) Shape {
	if len(shapes) == 0 {
		return nil
	}
	largest := shapes[0]
	for _, s := range shapes[1:] {
		if s.Area() > largest.Area() {
			largest = s
		}
	}
	return largest
}

// ─── 4. The Empty Interface: any / interface{} ────────────────────────────────
//
// interface{} (aliased as "any" since Go 1.18) has NO methods.
// EVERY type satisfies the empty interface — it can hold any value.
//
// USE any when:
//   - You genuinely don't know the type at compile time.
//   - Building generic containers before generics were available.
//   - Working with JSON, reflection, or heterogeneous data.
//
// AVOID overusing any: you lose type safety and must use type assertions
// to get back to the concrete type. Prefer generics (Go 1.18+) for
// type-safe containers (see Module 09).

func printAny(v any) {
	// fmt uses the underlying type's String() method or default formatting
	fmt.Printf("  value: %v  |  type: %T\n", v, v)
}

// ─── 5. The Typed Nil Bug — the Most Famous Go Gotcha ─────────────────────────
//
// BACKGROUND: An interface value = (type, value) pair.
//
//   var s Shape = nil   → type=nil, value=nil  → interface IS nil (s == nil is true)
//
//   var r *Rect = nil   → r is a nil *Rect
//   var s Shape = r     → type=*Rect, value=nil → interface IS NOT nil!
//                         s == nil is FALSE even though the pointer inside is nil
//
// WHY: The interface has a non-nil type descriptor (*Rect), even though the
//      pointer value is nil. The interface value itself is non-nil.
//
// WHEN IT BITES YOU: The classic case is returning an interface (often error)
// from a function that returns a typed nil pointer.

// brokenNewRect returns a *Rect (typed nil) as a Shape interface.
// This is the bug pattern: the function LOOKS like it returns nil when
// the rect is nil, but the caller receives a non-nil interface.
func brokenNewRect(bad bool) Shape {
	var r *Rect // r is nil *Rect

	if bad {
		return r // BUG: returns (*Rect, nil) — interface is NOT nil!
	}

	return &Rect{Width: 3, Height: 4} // returns (*Rect, 0xADDR) — fine
}

// fixedNewRect correctly returns a nil interface when there's no value.
func fixedNewRect(bad bool) Shape {
	if bad {
		return nil // returns (nil, nil) — interface IS nil
	}
	return &Rect{Width: 3, Height: 4}
}

// ─── Sorting with sort.Interface ─────────────────────────────────────────────
//
// Real-world interface usage: sort.Interface requires Len(), Less(i,j), Swap(i,j).
// Any type that implements these three methods can be sorted by sort.Sort().

type ByArea []Shape

func (b ByArea) Len() int           { return len(b) }
func (b ByArea) Less(i, j int) bool { return b[i].Area() < b[j].Area() }
func (b ByArea) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Interfaces — Basics & Duck Typing")
	fmt.Println("========================================")

	// ── Implicit Satisfaction ────────────────────────────────────────────────
	fmt.Println("\n── Implicit Interface Satisfaction ──────────────────")

	shapes := []Shape{
		Rect{Width: 3, Height: 4},
		Circ{Radius: 5},
		Triangle{A: 3, B: 4, C: 5},
	}

	for _, s := range shapes {
		// %T prints the dynamic type of the interface value
		fmt.Printf("  %T:\n", s)
		printShapeInfo(s)
	}

	// ── Interface as Abstraction ─────────────────────────────────────────────
	fmt.Println("\n── Interface as Abstraction ─────────────────────────")

	fmt.Printf("Total area: %.4f\n", totalArea(shapes))

	largest := largestShape(shapes)
	fmt.Printf("Largest shape: %T with area %.4f\n", largest, largest.Area())

	// ── A Type Can Satisfy Multiple Interfaces ───────────────────────────────
	fmt.Println("\n── One Type, Multiple Interfaces ────────────────────")

	r := Rect{Width: 6, Height: 2}

	// Rect satisfies BOTH Shape and Describer
	var s Shape = r
	var d Describer = r

	fmt.Printf("As Shape:     Area=%.2f\n", s.Area())
	fmt.Printf("As Describer: %s\n", d.Describe())

	// Circ also satisfies both
	var s2 Shape = Circ{Radius: 3}
	var d2 Describer = Circ{Radius: 3}
	fmt.Printf("Circ as Shape:     Area=%.4f\n", s2.Area())
	fmt.Printf("Circ as Describer: %s\n", d2.Describe())

	// Triangle satisfies Shape but NOT Describer
	var s3 Shape = Triangle{A: 3, B: 4, C: 5}
	fmt.Printf("Triangle as Shape: Area=%.4f\n", s3.Area())
	// var d3 Describer = Triangle{...}  // compile error: Triangle does not implement Describer

	// ── The Empty Interface: any ─────────────────────────────────────────────
	fmt.Println("\n── Empty Interface (any) ────────────────────────────")

	// any can hold any value
	values := []any{
		42,
		"hello",
		true,
		3.14,
		Rect{Width: 2, Height: 3},
		[]int{1, 2, 3},
		nil,
	}

	for _, v := range values {
		printAny(v)
	}

	// ── any in a map (heterogeneous data) ────────────────────────────────────
	fmt.Println("\n── any Map (heterogeneous config) ───────────────────")

	config := map[string]any{
		"host":    "localhost",
		"port":    8080,
		"debug":   true,
		"timeout": 30.5,
	}

	for k, v := range config {
		fmt.Printf("  %s = %v (%T)\n", k, v, v)
	}

	// Accessing a typed value requires a type assertion (see file 08)
	if port, ok := config["port"].(int); ok {
		fmt.Printf("  Port as int: %d\n", port)
	}

	// ── Typed Nil Bug ────────────────────────────────────────────────────────
	fmt.Println("\n── The Infamous Typed Nil Bug ───────────────────────")

	broken := brokenNewRect(true) // returns (*Rect)(nil) as Shape
	fixed := fixedNewRect(true)   // returns nil interface

	fmt.Printf("broken == nil: %v  (BUG: looks like nil but isn't!)\n", broken == nil)
	fmt.Printf("fixed  == nil: %v  (CORRECT: truly nil interface)\n", fixed == nil)

	// Accessing broken.Area() would PANIC because the *Rect pointer is nil
	// and Area() is a value receiver — Go tries to copy a nil pointer.
	// If Area() were a pointer receiver, it'd be safe IF it handled nil.

	// The FIX: never return a typed nil as an interface.
	// Always return an untyped nil literal, or check before returning.

	// Demonstrating the interface internals visually:
	var iface Shape // (nil, nil) — truly nil
	fmt.Printf("\niface type:  %T\n", iface)    // <nil>
	fmt.Printf("iface value: %v\n", iface)     // <nil>
	fmt.Printf("iface == nil: %v\n", iface == nil) // true

	var ptr *Rect   // (*Rect)(nil)
	iface = ptr    // now iface = (*Rect, nil)
	fmt.Printf("\nAfter iface = (*Rect)(nil):\n")
	fmt.Printf("iface type:  %T\n", iface)    // *main.Rect
	fmt.Printf("iface value: %v\n", iface)    // <nil>
	fmt.Printf("iface == nil: %v\n", iface == nil) // FALSE — the bug!

	// ── Real-World: sort.Interface ────────────────────────────────────────────
	fmt.Println("\n── Real-World: sort.Interface ───────────────────────")

	sortable := []Shape{
		Triangle{A: 3, B: 4, C: 5},
		Circ{Radius: 1},
		Rect{Width: 10, Height: 10},
		Rect{Width: 2, Height: 3},
	}

	fmt.Println("Before sort:")
	for _, s := range sortable {
		fmt.Printf("  %T area=%.4f\n", s, s.Area())
	}

	sort.Sort(ByArea(sortable))

	fmt.Println("After sort (by area, ascending):")
	for _, s := range sortable {
		fmt.Printf("  %T area=%.4f\n", s, s.Area())
	}

	// ── Key Takeaways ────────────────────────────────────────────────────────
	fmt.Println("\n── Key Takeaways ────────────────────────────────────")
	fmt.Println(`
  1. Interfaces are satisfied implicitly — no 'implements' keyword.
  2. An interface value = (dynamic type, dynamic value).
  3. A nil interface has both type AND value as nil.
  4. A typed nil (e.g., (*T)(nil) stored in interface) is NOT nil.
  5. Use 'any' sparingly — prefer generics for type-safe containers.
  6. Small interfaces (1-2 methods) are idiomatic Go.
  7. Accept interfaces in function params → flexible and testable code.
  `)
}
