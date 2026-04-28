// FILE: book/part2_core_language/chapter24_generics/examples/02_constraints/main.go
// CHAPTER: 24 — Generics: Type Parameters and Constraints
// TOPIC: Constraint interfaces, ~T (underlying type), ordered types,
//        golang.org/x/exp/constraints, when NOT to use generics.
//
// Run (from the chapter folder):
//   go run ./examples/02_constraints

package main

import "fmt"

// --- Custom constraints ---

// Integer constrains to all built-in integer types.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Float constrains to float types.
type Float interface {
	~float32 | ~float64
}

// Number combines Integer and Float.
type Number interface {
	Integer | Float
}

// Ordered constrains to types that support < > <= >=.
type Ordered interface {
	Integer | Float | ~string
}

// --- Functions using constraints ---

// Min returns the smaller of two Ordered values.
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two Ordered values.
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Sum adds all Number values.
func Sum[T Number](nums []T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

// Clamp constrains v to [lo, hi].
func Clamp[T Ordered](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// --- The ~ (tilde) operator ---

// MyInt is a named type with underlying type int.
type MyInt int

// Because Integer uses ~int, MyInt satisfies Integer.

func doubleInt[T Integer](v T) T { return v * 2 }

// --- When NOT to use generics ---
//
// Avoid generics when:
// 1. The function works on a single specific type.
// 2. Behaviour differs per type — use interfaces + type switch.
// 3. You need runtime type inspection — use reflect.
// 4. The generic version is harder to read than three simple copies.
//
// The canonical wrong use: func PrintAny[T any](v T) { fmt.Println(v) }
// Just use fmt.Println — it already accepts any.

func main() {
	// Min / Max
	fmt.Println("Min(3,5):", Min(3, 5))
	fmt.Println("Max(3.14, 2.71):", Max(3.14, 2.71))
	fmt.Println("Min(\"apple\",\"banana\"):", Min("apple", "banana"))

	fmt.Println()

	// Sum
	ints := []int{1, 2, 3, 4, 5}
	floats := []float64{1.1, 2.2, 3.3}
	fmt.Println("Sum(ints):", Sum(ints))
	fmt.Printf("Sum(floats): %.1f\n", Sum(floats))

	fmt.Println()

	// Clamp
	fmt.Println("Clamp(15, 0, 10):", Clamp(15, 0, 10))
	fmt.Println("Clamp(-5, 0, 10):", Clamp(-5, 0, 10))
	fmt.Println("Clamp(5, 0, 10):", Clamp(5, 0, 10))

	fmt.Println()

	// ~ underlying type
	var m MyInt = 21
	fmt.Println("doubleInt(MyInt(21)):", doubleInt(m))

	fmt.Println()

	// Type inference — explicit type params rarely needed
	fmt.Println("Min inferred: type params not written")
	result := Min(100, 200) // compiler infers T=int
	fmt.Println("result:", result)
}
