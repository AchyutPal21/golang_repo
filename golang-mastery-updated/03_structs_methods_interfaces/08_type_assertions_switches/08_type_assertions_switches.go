// FILE: 03_structs_methods_interfaces/08_type_assertions_switches.go
// TOPIC: Type Assertions & Type Switches — safe dynamic dispatch
//
// Run: go run 03_structs_methods_interfaces/08_type_assertions_switches.go

package main

import "fmt"

type Animal interface {
	Sound() string
}

type Dog struct{ Name string }
type Cat struct{ Name string }
type Bird struct{ Name string }

func (d Dog) Sound() string  { return "Woof" }
func (c Cat) Sound() string  { return "Meow" }
func (b Bird) Sound() string { return "Tweet" }
func (b Bird) Fly() string   { return b.Name + " is flying" }

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Type Assertions & Switches")
	fmt.Println("════════════════════════════════════════")

	// ── TYPE ASSERTION ───────────────────────────────────────────────────
	// Syntax: value.(ConcreteType)
	// An interface variable holds a (type, value) pair internally.
	// Type assertion extracts the concrete value.
	//
	// TWO FORMS:
	//   v := i.(T)          → panics if i is not T
	//   v, ok := i.(T)      → ok=false if not T, never panics (ALWAYS use this)

	var a Animal = Dog{Name: "Rex"}

	fmt.Println("\n── Type assertion ──")
	// Safe form (always prefer this):
	if dog, ok := a.(Dog); ok {
		fmt.Printf("  It's a Dog! Name: %s, Sound: %s\n", dog.Name, dog.Sound())
	}

	// Asserting to an interface the concrete type also satisfies:
	type Flyer interface{ Fly() string }
	var a2 Animal = Bird{Name: "Tweety"}
	if flyer, ok := a2.(Flyer); ok {
		fmt.Printf("  Bird implements Flyer: %s\n", flyer.Fly())
	}

	// ── TYPE SWITCH ──────────────────────────────────────────────────────
	// Type switch checks multiple types in one statement.
	// Each case tests whether the interface value holds that type.
	// The variable in each case has the concrete type (not the interface).

	fmt.Println("\n── Type switch ──")
	animals := []Animal{Dog{"Buddy"}, Cat{"Whiskers"}, Bird{"Tweety"}}
	for _, animal := range animals {
		switch v := animal.(type) {
		case Dog:
			fmt.Printf("  Dog %q says %s\n", v.Name, v.Sound())
		case Cat:
			fmt.Printf("  Cat %q says %s\n", v.Name, v.Sound())
		case Bird:
			fmt.Printf("  Bird %q says %s and can fly: %s\n", v.Name, v.Sound(), v.Fly())
		default:
			fmt.Printf("  Unknown animal: %T\n", v)
		}
	}

	// ── THE TYPED NIL GOTCHA ─────────────────────────────────────────────
	// This is one of Go's most famous subtle bugs.
	// An interface is nil only if BOTH its type AND value are nil.
	// A *Dog that is nil, stored in an Animal interface, is NOT nil!
	fmt.Println("\n── The typed nil interface gotcha ──")
	var d *Dog = nil
	var iface Animal = d   // iface holds (type=*Dog, value=nil)
	fmt.Printf("  d == nil:      %v\n", d == nil)
	fmt.Printf("  iface == nil:  %v  ← NOT nil! has type *Dog\n", iface == nil)
	// FIX: only assign to interface if the pointer is non-nil, or check type:
	if d2, ok := iface.(*Dog); ok && d2 == nil {
		fmt.Println("  Correctly detected: interface holds a nil *Dog")
	}

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  v, ok := i.(T)  → always use two-value form (no panic)")
	fmt.Println("  switch v := i.(type) { case T: ... }  → multi-type dispatch")
	fmt.Println("  Interface internals: (type, value) pair")
	fmt.Println("  Typed nil gotcha: interface ≠ nil even if concrete value is nil")
}
