// FILE: book/part2_core_language/chapter22_interfaces/examples/02_interface_internals/main.go
// CHAPTER: 22 — Interfaces: Go's Killer Feature
// TOPIC: The itab (type, value) pair, nil interface vs typed nil,
//        interface comparison, interface overhead.
//
// Run (from the chapter folder):
//   go run ./examples/02_interface_internals

package main

import (
	"errors"
	"fmt"
)

// An interface value is two words:
//   word 1: *itab — pointer to (interface type, concrete type, method table)
//   word 2: data  — pointer to (or small value of) the concrete value
//
// A nil interface has BOTH words nil.
// A typed nil has a non-nil itab and a nil data pointer.

// --- Nil interface ---

func nilInterface() error {
	return nil // both words are nil
}

// --- Typed nil: the famous gotcha ---

type myError struct{ msg string }

func (e *myError) Error() string { return e.msg }

// buggy: returns a non-nil interface wrapping a nil *myError pointer.
func findBuggy(fail bool) error {
	var err *myError // nil pointer of type *myError
	if fail {
		err = &myError{"something failed"}
	}
	return err // WRONG: wraps nil pointer in error interface
	// itab = (*myError, error), data = nil → interface is NOT nil
}

// fixed: only set the interface when there's an actual error.
func findFixed(fail bool) error {
	if fail {
		return &myError{"something failed"}
	}
	return nil // true nil interface
}

// --- Interface comparison ---

type Animal interface{ Sound() string }

type Dog struct{}
type Cat struct{}

func (d Dog) Sound() string { return "woof" }
func (c Cat) Sound() string { return "meow" }

func interfaceComparison() {
	var a1, a2 Animal = Dog{}, Dog{}
	var a3 Animal = Cat{}

	fmt.Println("Dog{} == Dog{}:", a1 == a2) // true: same type, same value
	fmt.Println("Dog{} == Cat{}:", a1 == a3) // false: different types

	var nilA Animal
	fmt.Println("nil == nil:", nilA == nil) // true
}

// --- errors.As and errors.Is with typed nil ---

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s — %s", e.Field, e.Message)
}

func validate(name string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "required"}
	}
	return nil
}

func main() {
	// --- nil interface ---
	err := nilInterface()
	fmt.Println("nilInterface() == nil:", err == nil) // true

	fmt.Println()

	// --- typed nil gotcha ---
	err1 := findBuggy(false)
	err2 := findFixed(false)
	fmt.Println("findBuggy(false) == nil:", err1 == nil) // FALSE — typed nil trap
	fmt.Println("findFixed(false) == nil:", err2 == nil) // true

	err3 := findBuggy(true)
	fmt.Println("findBuggy(true) error:", err3) // "something failed"

	fmt.Println()
	fmt.Println("LESSON: never return a typed nil through an interface.")
	fmt.Println("  Return the interface type directly, or use a bool/error guard.")

	fmt.Println()

	// --- interface comparison ---
	interfaceComparison()

	fmt.Println()

	// --- errors.Is / errors.As ---
	err4 := validate("")
	var ve *ValidationError
	if errors.As(err4, &ve) {
		fmt.Printf("validation error: field=%s msg=%s\n", ve.Field, ve.Message)
	}
	fmt.Println("validate('alice'):", validate("alice"))

	fmt.Println()

	// --- interface overhead: dynamic dispatch ---
	// Interface method calls go through the itab's method table pointer.
	// This prevents inlining in most cases and adds ~1ns vs direct call.
	// For hot loops, consider type switches or generics instead of interfaces.
	fmt.Println("Interface overhead: ~1ns per call (dynamic dispatch via itab)")
	fmt.Println("  Acceptable for I/O, HTTP handlers, plugins, test fakes.")
	fmt.Println("  Profile before removing interfaces for performance reasons.")
}
