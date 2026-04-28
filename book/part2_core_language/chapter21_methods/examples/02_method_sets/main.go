// FILE: book/part2_core_language/chapter21_methods/examples/02_method_sets/main.go
// CHAPTER: 21 — Methods: Functions With Receivers
// TOPIC: Method sets, interface satisfaction, value-in-interface vs
//        pointer-in-interface, the addressability rule.
//
// Run (from the chapter folder):
//   go run ./examples/02_method_sets

package main

import "fmt"

// Stringer is a minimal interface for illustration.
type Stringer interface {
	String() string
}

// Mover is an interface requiring a pointer receiver method.
type Mover interface {
	Move(dx, dy float64)
}

// ShapeI requires both a value-receiver method and a pointer-receiver method.
type ShapeI interface {
	Area() float64   // value receiver
	Scale(f float64) // pointer receiver
}

// Rect has both value and pointer receiver methods.
type Rect struct {
	W, H float64
}

func (r Rect) Area() float64    { return r.W * r.H }
func (r Rect) String() string   { return fmt.Sprintf("Rect(%gx%g)", r.W, r.H) }
func (r *Rect) Scale(f float64) { r.W *= f; r.H *= f }
func (r *Rect) Move(dx, dy float64) {
	fmt.Printf("moved by (%.1f, %.1f)\n", dx, dy)
}

// Method set rules:
//   T's method set  = methods with value receiver T
//  *T's method set  = methods with value receiver T + methods with pointer receiver *T

func main() {
	// --- *Rect satisfies ShapeI (has both Area and Scale) ---
	var s ShapeI = &Rect{W: 4, H: 3}
	fmt.Printf("area=%.1f\n", s.Area())
	s.Scale(2)
	fmt.Printf("after Scale(2): %v\n", s)

	// --- Rect (value) does NOT satisfy ShapeI because Scale needs *Rect ---
	// The following would be a compile error:
	// var s2 ShapeI = Rect{W: 4, H: 3} // cannot use Rect as ShapeI: Scale requires *Rect

	fmt.Println()

	// --- Value in interface: a copy is stored ---
	var str Stringer = Rect{W: 2, H: 5} // Rect has String() with value receiver — OK
	fmt.Println("value in interface:", str.String())

	// Rect stored in interface is a copy — mutations don't affect original.
	r := Rect{W: 2, H: 5}
	str = r
	// Calling Scale would require type assertion to *Rect — can't scale through Stringer

	fmt.Println()

	// --- Pointer in interface: mutations visible ---
	rp := &Rect{W: 2, H: 5}
	var m Mover = rp
	m.Move(10, 20) // printed

	_ = r   // silence unused
	_ = str // silence unused

	fmt.Println()

	// --- Addressability rule ---
	// You cannot take the address of a value stored in an interface.
	// The following does not compile:
	//   var i interface{} = Rect{1,1}
	//   i.(*Rect).Scale(2) // Rect inside interface is not addressable

	// You CAN type-assert a pointer:
	var i interface{} = &Rect{W: 3, H: 3}
	if rr, ok := i.(*Rect); ok {
		rr.Scale(3)
		fmt.Println("type-asserted and scaled:", rr)
	}

	fmt.Println()

	// --- Summary table printed at runtime ---
	fmt.Println("Method set rules:")
	fmt.Println("  T  method set: value receiver methods only")
	fmt.Println("  *T method set: value + pointer receiver methods")
	fmt.Println("  => *T satisfies interfaces that T does not")
	fmt.Println("  => pass &value, not value, when interface has pointer-receiver methods")
}
