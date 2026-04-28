// FILE: book/part2_core_language/chapter08_variables_constants_zero/examples/01_declarations/main.go
// CHAPTER: 08 — Variables, Constants, and the Zero Value
// TOPIC: All four declaration forms, plus multiple assignment and shadowing.
//
// Run (from the chapter folder):
//   go run ./examples/01_declarations
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Senior Go engineers reflexively pick the right declaration form for the
//   context. This file walks through all four side by side, plus multiple
//   assignment, plus shadowing — and shows why `go vet` is the safety net.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// ─── Package-level declarations ──────────────────────────────────────────────
//
// At package level you must use `var` (and `const`); `:=` is illegal here.
// Grouped `var ()` blocks are idiomatic when several related variables share
// a purpose.
var (
	// Form 1: type + initializer (rare; the type is implied by the literal,
	// so the explicit type is usually redundant).
	pkgVersion string = "1.0.0"

	// Form 2: type, no initializer — gets the zero value.
	startupErrors []error // zero is nil

	// Form 3: initializer, type inferred — most common at package level.
	maxRetries = 3
)

func main() {
	fmt.Println("=== Form 1: var with type and initializer ===")
	var x int = 42 // explicit; rarely useful unless you need a non-default type
	fmt.Printf("x = %d (type %T)\n", x, x)

	fmt.Println("\n=== Form 2: var with type, no initializer (zero value) ===")
	var s string
	var p *int
	var m map[string]int
	fmt.Printf("s = %q (type %T)\n", s, s)
	fmt.Printf("p = %v (type %T)\n", p, p)
	fmt.Printf("m == nil → %v\n", m == nil)

	fmt.Println("\n=== Form 3: var with initializer, type inferred ===")
	var y = 42 // y is int
	fmt.Printf("y = %d (type %T)\n", y, y)

	fmt.Println("\n=== Form 4: short declaration (functions only) ===")
	z := 42 // z is int — the workhorse inside functions
	fmt.Printf("z = %d (type %T)\n", z, z)

	// ─── Multiple declaration ────────────────────────────────────────────────
	fmt.Println("\n=== Multiple declaration ===")
	a, b, c := 1, "two", 3.14
	fmt.Printf("a=%v (%T), b=%v (%T), c=%v (%T)\n", a, a, b, b, c, c)

	// ─── Multiple assignment (and the swap idiom) ────────────────────────────
	fmt.Println("\n=== The swap idiom ===")
	one, two := 1, 2
	fmt.Printf("before: one=%d two=%d\n", one, two)
	one, two = two, one // no temp variable required
	fmt.Printf("after:  one=%d two=%d\n", one, two)

	// ─── The blank identifier ────────────────────────────────────────────────
	fmt.Println("\n=== Blank identifier ===")
	_, ok := lookup(map[string]int{"a": 1}, "a")
	fmt.Printf("ok=%v\n", ok)

	// ─── := with mixed declare/assign ────────────────────────────────────────
	//
	// Rule: in `a, b := ..., ...`, at least one of a or b must be NEW for
	// the line to compile. Existing names get assigned, new names get
	// declared. This is the only form of "redeclaration" Go allows.
	fmt.Println("\n=== Mixed declare/assign with := ===")
	existing := 1
	existing, fresh := 2, 3 // existing is reassigned, fresh is new
	fmt.Printf("existing=%d (assigned), fresh=%d (declared)\n", existing, fresh)

	// ─── Shadowing: the silent bug ───────────────────────────────────────────
	//
	// Inside an inner block, `:=` creates a NEW variable hiding the outer
	// one. Inside the same block, `=` updates the outer.
	//
	// We won't actually demonstrate the bug here (it would require
	// disabling -vet), but the comment shows the shape.
	fmt.Println("\n=== Shadowing demo ===")
	err := outer()
	fmt.Printf("outer() returned err=%v\n", err)

	fmt.Printf("\n(package-level: pkgVersion=%s, maxRetries=%d, startupErrors=%v)\n",
		pkgVersion, maxRetries, startupErrors)
}

// outer demonstrates the *correct* pattern for updating an outer error.
// We use `=`, not `:=`, when we want to overwrite the outer `err`.
func outer() error {
	var err error
	if true {
		// Use = to assign to the OUTER err. If you wrote `err := nil` here
		// you would shadow it (and `go vet` would warn).
		err = nil
	}
	return err
}

// lookup is here only to give the blank-identifier example something to call.
func lookup(m map[string]int, k string) (int, bool) {
	v, ok := m[k]
	return v, ok
}
