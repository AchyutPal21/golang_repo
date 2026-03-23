// 03_methods.go
//
// METHODS — functions with a receiver.
//
// In Go, a method is just a function with a special first parameter called the
// receiver. There is no "this" or "self" keyword. The receiver is explicitly
// declared and can be any name (conventionally the first letter of the type).
//
// Syntax:
//   func (receiver ReceiverType) MethodName(params) returnType { ... }
//
// Methods can only be defined in the SAME PACKAGE as the type. You cannot
// add methods to types defined in other packages (e.g., you can't add a method
// to string or int directly — you must define a new named type).

package main

import (
	"fmt"
	"math"
)

// ─── Types We'll Use ──────────────────────────────────────────────────────────

type Rectangle struct {
	Width  float64
	Height float64
}

type Counter struct {
	value int
}

// ─── 1. Value Receiver ────────────────────────────────────────────────────────
//
// A value receiver receives a COPY of the struct.
// Modifications to the receiver inside the method do NOT affect the original.
//
// USE value receiver when:
//   - The method does not need to modify the receiver.
//   - The receiver is a small struct (copying is cheap).
//   - You want the method to work on an immutable view of the data.
//   - The type is a basic type (int, float64, etc.).
//
// Value receiver methods are in the method set of BOTH T and *T.

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

func (r Rectangle) IsSquare() bool {
	return r.Width == r.Height
}

// Attempting to modify with a value receiver — has NO effect on caller's struct.
// This is a common mistake beginners make.
func (r Rectangle) ScaleWrong(factor float64) {
	// r is a copy — this modifies the local copy, not the original
	r.Width *= factor  // lost when function returns
	r.Height *= factor // lost when function returns
}

// ─── 2. Pointer Receiver ──────────────────────────────────────────────────────
//
// A pointer receiver receives a POINTER to the struct.
// Modifications AFFECT the original struct.
//
// USE pointer receiver when:
//   - The method needs to MODIFY the receiver.
//   - The struct is large (avoid expensive copies).
//   - Consistency: if any method on the type uses a pointer receiver,
//     ALL methods should use pointer receivers for consistency and correctness
//     (important for satisfying interfaces — see below).
//
// Pointer receiver methods are ONLY in the method set of *T (not T).
// Exception: if you have an ADDRESSABLE value of type T, Go auto-takes its address.

func (r *Rectangle) Scale(factor float64) {
	r.Width *= factor  // modifies the ORIGINAL through the pointer
	r.Height *= factor
}

func (r *Rectangle) SetDimensions(w, h float64) {
	r.Width = w
	r.Height = h
}

// ─── 3. Methods on Non-Struct Types ───────────────────────────────────────────
//
// You can define methods on ANY named type in the same package,
// not just structs. This includes named types based on primitives.

type Celsius float64
type Fahrenheit float64

func (c Celsius) ToFahrenheit() Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

func (f Fahrenheit) ToCelsius() Celsius {
	return Celsius((f - 32) * 5 / 9)
}

func (c Celsius) String() string {
	return fmt.Sprintf("%.2f°C", float64(c))
}

// ─── 4. Method Sets — The Rule ────────────────────────────────────────────────
//
// The METHOD SET of a type determines which interface it satisfies.
//
//   Type T:  method set = { all value receiver methods }
//   Type *T: method set = { all value receiver methods } ∪ { all pointer receiver methods }
//
// This means:
//   - A value T can call value methods directly.
//   - A value T can call pointer methods ONLY if it is addressable (Go auto-takes &t).
//   - An interface variable holding T can only call value methods.
//   - An interface variable holding *T can call both.
//
// PRACTICAL IMPLICATION:
//   If you implement an interface with a pointer receiver method, you MUST
//   store a *T in the interface, not a T.

type Shaper interface {
	Area() float64
}

type Resizer interface {
	Scale(factor float64)
}

// Circle uses value receivers — Circle (not *Circle) satisfies Shaper.
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

// ─── 5. Calling Methods on Nil ────────────────────────────────────────────────
//
// With a POINTER receiver, the method can be called even if the pointer is nil.
// The method must handle the nil case explicitly (check if receiver is nil).
// This is occasionally useful for building "null object" patterns.
//
// With a VALUE receiver, calling on nil panics (can't dereference nil to copy).

type Node struct {
	Value int
	Next  *Node
}

// Sum traverses the linked list. Safe to call on nil (empty list).
func (n *Node) Sum() int {
	if n == nil {
		return 0 // base case: nil node contributes 0
	}
	return n.Value + n.Next.Sum() // recursive traversal
}

// ─── 6. Method Expressions ────────────────────────────────────────────────────
//
// A METHOD EXPRESSION turns a method into a plain function.
// The receiver becomes the first parameter.
//
// Syntax: TypeName.MethodName → becomes func(TypeName, ...params) returnType
//
// WHY: Useful when you want to store a method in a variable, pass it as a
// callback, or work with it as a first-class function.

// ─── 7. Method Values ─────────────────────────────────────────────────────────
//
// A METHOD VALUE binds a specific receiver to a method.
// The result is a function that, when called, uses the bound receiver.
//
// Syntax: instance.MethodName → becomes func(...params) returnType
//
// WHY: Convenient for callbacks, goroutines, event handlers, etc.

// ─── 8. Consistency Rules (Important!) ───────────────────────────────────────
//
// The Go specification and vet tools expect consistency:
//
//   RULE 1: Don't mix value and pointer receivers on the same type if
//           the type will implement interfaces. Use ALL pointer receivers
//           or ALL value receivers.
//
//   RULE 2: If ANY method mutates state → use pointer receivers for ALL methods.
//
//   RULE 3: Large structs → always pointer receivers to avoid copies.
//
//   RULE 4: Small immutable value types (like Point, Color) → value receivers are fine.

// Counter uses ALL pointer receivers because Increment mutates state.
func (c *Counter) Increment() {
	c.value++
}

func (c *Counter) Add(n int) {
	c.value += n
}

func (c *Counter) Reset() {
	c.value = 0
}

func (c *Counter) Value() int {
	return c.value // read-only, but pointer receiver for consistency
}

func (c *Counter) String() string {
	return fmt.Sprintf("Counter(%d)", c.value)
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Go Methods — Deep Dive")
	fmt.Println("========================================")

	// ── Value vs Pointer Receiver ────────────────────────────────────────────
	fmt.Println("\n── Value vs Pointer Receiver ────────────────────────")

	r := Rectangle{Width: 10, Height: 5}
	fmt.Printf("rect: %+v\n", r)
	fmt.Printf("Area():      %.2f\n", r.Area())
	fmt.Printf("Perimeter(): %.2f\n", r.Perimeter())
	fmt.Printf("IsSquare():  %v\n", r.IsSquare())

	// Demonstrate that value receiver doesn't modify the original
	r.ScaleWrong(2.0) // returns, but r is unchanged
	fmt.Printf("After ScaleWrong(2.0): %+v (unchanged!)\n", r)

	// Pointer receiver DOES modify the original
	r.Scale(2.0) // Go auto-takes &r because r is addressable
	// This is equivalent to: (&r).Scale(2.0)
	fmt.Printf("After Scale(2.0):      %+v (changed!)\n", r)

	// ── Method on Non-Struct Type ────────────────────────────────────────────
	fmt.Println("\n── Methods on Named Primitive Types ─────────────────")

	boiling := Celsius(100)
	fmt.Printf("Boiling: %s = %.2f°F\n", boiling, boiling.ToFahrenheit())

	bodyTemp := Fahrenheit(98.6)
	fmt.Printf("Body temp: %.1f°F = %s\n", bodyTemp, bodyTemp.ToCelsius())

	// ── Method Sets & Interfaces ─────────────────────────────────────────────
	fmt.Println("\n── Method Sets & Interface Satisfaction ─────────────")

	c := Circle{Radius: 5}

	// Circle (value type) satisfies Shaper (Area is a value receiver method)
	var s Shaper = c // OK: T satisfies interface with value receiver methods
	fmt.Printf("Circle area via Shaper: %.4f\n", s.Area())

	// *Circle also satisfies Shaper (pointer's method set is a superset)
	var s2 Shaper = &c
	fmt.Printf("*Circle area via Shaper: %.4f\n", s2.Area())

	// Rectangle has Scale with POINTER receiver.
	// So *Rectangle satisfies Resizer, but Rectangle does NOT.
	rect := Rectangle{Width: 4, Height: 6}
	var resizer Resizer = &rect // must use pointer
	resizer.Scale(1.5)
	fmt.Printf("After resizer.Scale(1.5): %+v\n", rect)

	// This would be a compile error:
	// var resizer2 Resizer = rect  // ERROR: Rectangle does not implement Resizer
	//                               // (Scale has pointer receiver)

	// ── Calling Methods on Nil ───────────────────────────────────────────────
	fmt.Println("\n── Methods on Nil Pointer (safe with pointer receiver) ──")

	// Build a linked list: 1 -> 2 -> 3 -> nil
	list := &Node{Value: 1, Next: &Node{Value: 2, Next: &Node{Value: 3}}}
	fmt.Printf("List sum: %d\n", list.Sum()) // 6

	// Empty list (nil) — Sum() handles nil gracefully
	var emptyList *Node
	fmt.Printf("Empty list sum: %d\n", emptyList.Sum()) // 0

	// ── Method Expressions ───────────────────────────────────────────────────
	fmt.Println("\n── Method Expressions ───────────────────────────────")

	// Method expression: Rectangle.Area has signature func(Rectangle) float64
	// The receiver becomes the first argument.
	areaFn := Rectangle.Area // type: func(Rectangle) float64
	r2 := Rectangle{Width: 3, Height: 4}
	fmt.Printf("areaFn(r2) = %.2f\n", areaFn(r2))

	// Useful for storing a list of operations to apply:
	operations := []func(Rectangle) float64{
		Rectangle.Area,
		Rectangle.Perimeter,
	}
	names := []string{"Area", "Perimeter"}
	for i, op := range operations {
		fmt.Printf("  %s: %.2f\n", names[i], op(r2))
	}

	// Pointer method expression: (*Rectangle).Scale
	// Signature: func(*Rectangle, float64)
	scaleFn := (*Rectangle).Scale
	r3 := Rectangle{Width: 2, Height: 3}
	scaleFn(&r3, 3.0) // pass &r3 explicitly
	fmt.Printf("After scaleFn(&r3, 3.0): %+v\n", r3)

	// ── Method Values ────────────────────────────────────────────────────────
	fmt.Println("\n── Method Values ────────────────────────────────────")

	r4 := Rectangle{Width: 5, Height: 5}

	// Method value: bind r4 to Area. Result: func() float64 (no receiver param)
	boundArea := r4.Area // type: func() float64, bound to r4
	fmt.Printf("boundArea() = %.2f\n", boundArea())

	// Useful for goroutines and callbacks:
	ctr := &Counter{}
	ctr.Add(10)

	// Bind Increment to this specific counter
	doIncrement := ctr.Increment // type: func()
	doIncrement()
	doIncrement()
	doIncrement()
	fmt.Printf("Counter after 3 doIncrement() calls: %s\n", ctr)

	// Pass as callback to a function
	applyNTimes(ctr.Increment, 5)
	fmt.Printf("Counter after applyNTimes(5): %s\n", ctr)

	// ── Consistency Demo with Counter ────────────────────────────────────────
	fmt.Println("\n── Counter (all pointer receivers for consistency) ──")

	ctr2 := &Counter{}
	ctr2.Increment()
	ctr2.Increment()
	ctr2.Add(8)
	fmt.Printf("Counter: %s\n", ctr2)
	ctr2.Reset()
	fmt.Printf("After Reset: %s\n", ctr2)

	// ── Summary ──────────────────────────────────────────────────────────────
	fmt.Println("\n── Summary: Which Receiver to Use ───────────────────")
	fmt.Println(`
  VALUE receiver  (r Rectangle):
    + No side effects on original
    + Safe for concurrent reads
    + Works directly on non-addressable values
    - Cannot mutate the original
    - Copying cost for large structs

  POINTER receiver (*r Rectangle):
    + Can mutate the original
    + Efficient for large structs (no copy)
    + Consistent with other mutating methods
    - Requires addressable value (or auto-take-address)
    - Must be careful with nil

  DECISION FLOWCHART:
    1. Does any method mutate state?   → ALL pointer receivers
    2. Is the struct large (> ~64B)?   → pointer receiver
    3. Are you implementing an interface? → match receiver type to method set
    4. None of the above?              → value receiver is fine
  `)
}

// applyNTimes applies a function n times. Accepts a method value.
func applyNTimes(fn func(), n int) {
	for i := 0; i < n; i++ {
		fn()
	}
}
