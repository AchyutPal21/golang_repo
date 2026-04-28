// FILE: book/part2_core_language/chapter10_conversion_assertion_switch/examples/02_type_assertions/main.go
// CHAPTER: 10 — Type Conversion, Assertion, Switch
// TOPIC: Type assertion (x.(T)) — runtime, interface→concrete.
//
// Run (from the chapter folder):
//   go run ./examples/02_type_assertions

package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// notFoundErr is a custom error type. We'll demonstrate how it round-trips
// through the `error` interface and back via assertion.
type notFoundErr struct {
	resource string
}

func (e *notFoundErr) Error() string {
	return "not found: " + e.resource
}

// findUser returns either a *notFoundErr (which is also an `error` via the
// Error() method) or nil.
func findUser(id int) error {
	if id == 0 {
		return &notFoundErr{resource: "user"}
	}
	return nil
}

// findUserBuggy demonstrates the typed-nil-interface gotcha.
func findUserBuggy(id int) error {
	var e *notFoundErr // typed nil pointer
	if id == 0 {
		e = &notFoundErr{resource: "user"}
	}
	return e // BUG: returns a non-nil interface even when e is nil
}

func main() {
	// ─── Safe assertion (comma-ok) ────────────────────────────────────────
	err := findUser(0)
	if nfErr, ok := err.(*notFoundErr); ok {
		fmt.Printf("got a notFoundErr; resource = %q\n", nfErr.resource)
	} else {
		fmt.Println("error was not a notFoundErr")
	}

	// ─── Modern equivalent: errors.As ─────────────────────────────────────
	//
	// Since Go 1.13 you should prefer errors.As over manual assertion;
	// it walks wrapped errors transparently. We'll cover this in Chapter 36.
	var target *notFoundErr
	if errors.As(err, &target) {
		fmt.Printf("(via errors.As) resource = %q\n", target.resource)
	}

	// ─── Asserting an interface (not just a concrete type) ────────────────
	var r io.Reader = strings.NewReader("data")
	if _, ok := r.(io.Closer); ok {
		fmt.Println("strings.Reader implements io.Closer")
	} else {
		fmt.Println("strings.Reader does NOT implement io.Closer (correct)")
	}

	// ─── The typed-nil-interface gotcha ───────────────────────────────────
	fmt.Println("\n--- typed nil interface ---")
	good := findUser(1)      // returns literal nil interface
	bad := findUserBuggy(1) // returns interface(*notFoundErr, nil)
	fmt.Printf("findUser(1)      == nil? %v   (expected true)\n", good == nil)
	fmt.Printf("findUserBuggy(1) == nil? %v   (TRAP: false even though the pointer is nil)\n", bad == nil)
	fmt.Println("\nThe interface holds a (type, value) pair. (*notFoundErr, nil) is")
	fmt.Println("NOT the same as (nil, nil). Always return literal nil from a")
	fmt.Println("function that returns an interface, not a typed nil pointer.")

	// ─── Panicky assertion — intentionally trigger to show the form ─────
	fmt.Println("\n--- panicky form caught with recover ---")
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic: %v\n", r)
		}
	}()
	var i any = "not an int"
	n := i.(int) // boom
	fmt.Println("unreachable:", n)
}
