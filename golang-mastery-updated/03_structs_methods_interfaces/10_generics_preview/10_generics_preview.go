// FILE: 03_structs_methods_interfaces/10_generics_preview.go
// TOPIC: Generics Preview — how generics relate to interfaces
//
// Run: go run 03_structs_methods_interfaces/10_generics_preview.go

package main

import "fmt"

// ── BEFORE GENERICS: interface{} (the old way) ───────────────────────────────
// Before Go 1.18, the only way to write type-agnostic code was interface{}/any.
// Problem: you lose type safety and need type assertions everywhere.

func maxInterface(a, b interface{}) interface{} {
	// No way to compare without type assertion:
	switch v := a.(type) {
	case int:
		if bv := b.(int); v > bv {
			return v
		}
		return b
	case float64:
		if bv := b.(float64); v > bv {
			return v
		}
		return b
	}
	return a
}

// ── WITH GENERICS: type parameters ───────────────────────────────────────────
// Constraints in generics ARE interfaces — they define what types are allowed.
// 'comparable' is a built-in constraint meaning the type supports ==.
// You can also write your own constraint interfaces.

// Ordered is a constraint interface: any type that supports < > operators.
// (In real code use golang.org/x/exp/constraints.Ordered)
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// Generic max — type safe, no boxing, no type assertions:
func maxGeneric[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Generic Contains — works for any comparable type:
func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Generics Preview")
	fmt.Println("════════════════════════════════════════")

	fmt.Println("\n── interface{} vs generics ──")
	// Old way — loses type info:
	r := maxInterface(10, 20)
	fmt.Printf("  maxInterface(10,20) = %v (type: %T)\n", r, r)
	// Must assert to use: r.(int) — extra code, potential panic

	// Generic — type safe:
	fmt.Printf("  maxGeneric(10, 20)      = %d (int)\n", maxGeneric(10, 20))
	fmt.Printf("  maxGeneric(3.14, 2.72)  = %.2f (float64)\n", maxGeneric(3.14, 2.72))
	fmt.Printf("  maxGeneric(\"b\", \"a\")    = %q (string)\n", maxGeneric("b", "a"))

	fmt.Println("\n── Generic contains ──")
	ints := []int{1, 2, 3, 4, 5}
	strs := []string{"go", "rust", "python"}
	fmt.Printf("  contains(ints, 3)    = %v\n", contains(ints, 3))
	fmt.Printf("  contains(ints, 9)    = %v\n", contains(ints, 9))
	fmt.Printf("  contains(strs, \"go\") = %v\n", contains(strs, "go"))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Generics use interfaces as CONSTRAINTS")
	fmt.Println("  interface{}/any = runtime type check (unsafe, boxing overhead)")
	fmt.Println("  Generics = compile-time type check (safe, no boxing)")
	fmt.Println("  Constraints are interfaces: ~int | ~float64 | ...")
	fmt.Println("  Full coverage in Module 09")
}
