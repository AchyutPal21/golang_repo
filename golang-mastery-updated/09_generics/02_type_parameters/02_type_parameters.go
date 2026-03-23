// 02_type_parameters.go
//
// TYPE PARAMETERS — The Syntax and Mechanics of Go Generics
// =========================================================
// This file covers every angle of type parameter syntax:
//   - Declaring type parameters on functions and types
//   - Explicit vs inferred type arguments at call sites
//   - Multiple type parameters
//   - Type parameters in return positions
//   - WHY methods cannot have their own type parameters (only the receiver type can)
//   - Generic functions vs generic types — when to use which

package main

import (
	"fmt"
	"strings"
)

// =============================================================================
// PART 1: BASIC SYNTAX — FUNCTION WITH A TYPE PARAMETER
// =============================================================================
//
// The general form is:
//
//   func FunctionName[TypeParam Constraint](params) returnType { ... }
//
// The square brackets [ ] introduce the type parameter list.
// TypeParam is the name you use inside the function body (like a variable name for types).
// Constraint is an interface that limits which types are valid for TypeParam.
//
// "any" means "no restriction" — any type is accepted.
// (any = interface{} — it's an alias since Go 1.18)

// Identity returns whatever value it receives, unchanged.
// This is the simplest possible generic function.
//
// Why is this useful?
// It proves that the function is generic without any computation.
// It's also useful in certain higher-order function patterns where
// a typed identity function is needed.
func Identity[T any](v T) T {
	return v
}

// Print prints any value with its type.
// Notice: we use %T to display the concrete type at runtime.
func PrintWithType[T any](v T) {
	fmt.Printf("value: %v  (type: %T)\n", v, v)
}

// =============================================================================
// PART 2: CALLING GENERIC FUNCTIONS — EXPLICIT vs INFERRED TYPE ARGUMENTS
// =============================================================================
//
// When calling a generic function, you CAN provide the type explicitly:
//
//   Identity[int](42)
//   Identity[string]("hello")
//
// OR you can let the compiler INFER the type from the argument:
//
//   Identity(42)      // compiler infers T = int
//   Identity("hello") // compiler infers T = string
//
// Type inference works when the type parameter appears in the function's
// parameter list. If the type parameter ONLY appears in the return type,
// inference is impossible and you MUST provide it explicitly.

// MakeZero returns the zero value of type T.
// T does NOT appear in the parameter list — only in the return type.
// Therefore: MakeZero[int]() works, but MakeZero() does NOT compile.
// The compiler has no information to infer T from.
func MakeZero[T any]() T {
	var zero T // var declaration gives us the zero value of type T
	return zero
}

// =============================================================================
// PART 3: MULTIPLE TYPE PARAMETERS
// =============================================================================
//
// You can declare multiple type parameters, separated by commas.
// Each has its own constraint.

// Pair holds two values of potentially different types.
// K and V can be any types, including the same type.
func MakePair[K, V any](key K, value V) (K, V) {
	return key, value
}

// Map applies a function to convert []T → []U
// This requires TWO type parameters because the input and output types differ.
//
// T = type of input elements (any type)
// U = type of output elements (any type)
//
// Why can't we use one type parameter here?
// If we wrote Map[T any](slice []T, f func(T) T) []T
// then f MUST return the same type as input. We couldn't do:
//   Map([]int{1,2,3}, func(n int) string { return fmt.Sprint(n) })
// because that converts int → string. We need T and U to be independent.
func MapSlice[T, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}

// ZipWith combines two slices element-by-element using a combiner function.
// Three type parameters: A (first slice), B (second slice), C (result).
func ZipWith[A, B, C any](as []A, bs []B, combine func(A, B) C) []C {
	// Use the shorter slice's length to avoid index out of bounds.
	n := len(as)
	if len(bs) < n {
		n = len(bs)
	}
	result := make([]C, n)
	for i := 0; i < n; i++ {
		result[i] = combine(as[i], bs[i])
	}
	return result
}

// =============================================================================
// PART 4: TYPE PARAMETER IN RETURN TYPE
// =============================================================================

// First returns the first element of a slice, plus a boolean indicating success.
// T appears in both parameter and return — inference works.
func First[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[0], true
}

// Convert demonstrates type conversion within a generic function.
// Here T is the input, U is the output, and the conversion must be
// expressed via the constraint (see 03_constraints.go for details).
// For now we show the signature pattern.
func Ptr[T any](v T) *T {
	// This is genuinely useful! Returns a pointer to a value.
	// Common need: struct field is *string but you have a string literal.
	//   s := Ptr("hello")  // s is *string
	// Without generics you write helper functions for each type.
	return &v
}

// =============================================================================
// PART 5: TYPE PARAMETERS ON TYPES (GENERIC TYPES)
// =============================================================================
//
// Not just functions — TYPES can also have type parameters.
// Generic types are covered in depth in 04_generic_types.go,
// but we introduce the syntax here for completeness.

// Box wraps a single value of any type.
// The type parameter T is declared on the type, not on individual methods.
type Box[T any] struct {
	value T
	label string
}

// NewBox creates a Box[T] with the given value and label.
// Notice: this is a GENERIC FUNCTION (has [T any] on the function),
// not a method. Methods on generic types do NOT have their own type params.
func NewBox[T any](value T, label string) Box[T] {
	return Box[T]{value: value, label: label}
}

// Unbox retrieves the value from a Box.
// This is a METHOD on Box[T]. Notice: no [T any] here.
// T is already declared on the Box type itself — the method inherits it.
//
// CRITICAL RULE: Methods in Go CANNOT have their own type parameters.
// Only the RECEIVER TYPE can have type parameters.
//
// Why? This decision was made for simplicity and to avoid complexity:
//   1. Interface satisfaction would become ambiguous.
//      If Box[T].Transform[U]() existed, what interface does Box[T] satisfy?
//      You'd need interfaces with method type parameters too — huge complexity.
//   2. Method sets must be concrete for interface satisfaction.
//      A method with its own type param would make Box[T] implement
//      an infinite family of interfaces — unmanageable.
//
// The workaround: use STANDALONE GENERIC FUNCTIONS instead of generic methods.
//   Instead of box.Transform[U](), write Transform[T, U](box, ...)
func (b Box[T]) Unbox() T {
	return b.value
}

func (b Box[T]) Label() string {
	return b.label
}

// String allows Box to satisfy the fmt.Stringer interface.
// Again: no extra type param on the method — only T from the receiver type.
func (b Box[T]) String() string {
	return fmt.Sprintf("Box(%q: %v)", b.label, b.value)
}

// =============================================================================
// PART 6: THE "METHODS CAN'T HAVE THEIR OWN TYPE PARAMS" RULE IN PRACTICE
// =============================================================================
//
// Suppose we want a method Transform[U any]() on Box[T] that converts the box.
// This is ILLEGAL in Go:
//
//   func (b Box[T]) Transform[U any](f func(T) U) Box[U] {  // COMPILE ERROR
//       return Box[U]{value: f(b.value), label: b.label}
//   }
//
// The solution: make it a package-level generic function.

// TransformBox converts Box[T] → Box[U] by applying f.
// Works around the "no method type params" restriction.
func TransformBox[T, U any](b Box[T], f func(T) U) Box[U] {
	return Box[U]{value: f(b.value), label: b.label}
}

// =============================================================================
// PART 7: GENERIC FUNCTIONS vs GENERIC TYPES — WHEN TO USE WHICH
// =============================================================================
//
// GENERIC FUNCTIONS:
//   - Algorithms that operate on data and return results
//   - No persistent state needed
//   - Examples: Map, Filter, Reduce, Sort, Min, Max, Contains
//
// GENERIC TYPES:
//   - Data structures that HOLD values and provide operations over time
//   - Persistent state (the stored elements)
//   - Examples: Stack[T], Queue[T], Set[T], Cache[K,V]
//
// The distinction mirrors the difference between:
//   - A sorting algorithm (function — takes data, returns result)
//   - A priority queue (type — holds data, provides Push/Pop over time)
//
// When in doubt: if you need to store T in a struct field → generic type.
//               If you just process T and return → generic function.

// Demonstrate a simple generic type: Stack

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	n := len(s.items)
	item := s.items[n-1]
	s.items = s.items[:n-1]
	return item, true
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

// =============================================================================
// PART 8: TYPE PARAMETER NAMING CONVENTIONS
// =============================================================================
//
// Single-letter names are conventional (like K, V, T, E):
//   T — general "Type" (the classic choice)
//   K — Key (in maps, caches)
//   V — Value (in maps, caches)
//   E — Element (in collections)
//   S — String-like type
//
// For multiple params, use descriptive names if single letters are ambiguous:
//   [Input, Output any]  — clearer than [T, U any] for transformation functions
//
// Avoid names that shadow built-in types:
//   Don't use [string any], [int any], [error any] — confusing and misleading.

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("TYPE PARAMETERS — Syntax and Mechanics")
	fmt.Println(strings.Repeat("=", 60))

	// --- Part 1: Basic generic function ---
	fmt.Println("\n--- Identity and PrintWithType ---")
	fmt.Println(Identity(42))         // T inferred as int
	fmt.Println(Identity("hello"))    // T inferred as string
	fmt.Println(Identity(3.14))       // T inferred as float64
	fmt.Println(Identity(true))       // T inferred as bool
	PrintWithType(42)
	PrintWithType("hello")
	PrintWithType([]int{1, 2, 3})

	// --- Part 2: Explicit vs inferred type arguments ---
	fmt.Println("\n--- Explicit vs Inferred type arguments ---")
	fmt.Println(Identity[int](42))    // explicit — same result
	fmt.Println(Identity[string]("hello")) // explicit

	// MakeZero: cannot infer T — must be explicit
	fmt.Printf("MakeZero[int]()    = %v\n", MakeZero[int]())
	fmt.Printf("MakeZero[string]() = %q\n", MakeZero[string]())
	fmt.Printf("MakeZero[bool]()   = %v\n", MakeZero[bool]())
	fmt.Printf("MakeZero[float64]()= %v\n", MakeZero[float64]())

	// --- Part 3: Multiple type parameters ---
	fmt.Println("\n--- Multiple type parameters ---")
	k, v := MakePair("name", 42)
	fmt.Printf("MakePair(\"name\", 42) → key=%q, value=%d\n", k, v)

	// MapSlice: convert []int → []string
	nums := []int{1, 2, 3, 4, 5}
	strs := MapSlice(nums, func(n int) string {
		return fmt.Sprintf("#%d", n)
	})
	fmt.Printf("MapSlice int→string: %v\n", strs)

	// MapSlice: square each int
	squared := MapSlice(nums, func(n int) int { return n * n })
	fmt.Printf("MapSlice square:     %v\n", squared)

	// ZipWith: add corresponding elements
	as := []int{1, 2, 3}
	bs := []int{10, 20, 30}
	sums := ZipWith(as, bs, func(a, b int) int { return a + b })
	fmt.Printf("ZipWith add:         %v\n", sums)

	// ZipWith: combine int + string → string
	labels := []string{"a", "b", "c"}
	combined := ZipWith(as, labels, func(n int, s string) string {
		return fmt.Sprintf("%s=%d", s, n)
	})
	fmt.Printf("ZipWith int+str:     %v\n", combined)

	// --- Part 4: Return type type parameters ---
	fmt.Println("\n--- Type parameter in return type ---")
	if v, ok := First([]int{10, 20, 30}); ok {
		fmt.Printf("First([10,20,30]) = %d\n", v)
	}
	if _, ok := First([]int{}); !ok {
		fmt.Println("First([]) = (zero, false) — empty slice")
	}

	// Ptr: extremely useful for pointer-to-literal pattern
	sp := Ptr("hello")
	np := Ptr(42)
	bp := Ptr(true)
	fmt.Printf("Ptr(\"hello\") = %p → %q\n", sp, *sp)
	fmt.Printf("Ptr(42)      = %p → %d\n", np, *np)
	fmt.Printf("Ptr(true)    = %p → %v\n", bp, *bp)

	// --- Part 5 & 6: Generic type Box ---
	fmt.Println("\n--- Generic type Box[T] ---")
	intBox := NewBox(42, "the answer")
	strBox := NewBox("gopher", "mascot")
	fmt.Println(intBox)
	fmt.Println(strBox)
	fmt.Printf("intBox.Unbox() = %d\n", intBox.Unbox())
	fmt.Printf("strBox.Unbox() = %q\n", strBox.Unbox())

	// TransformBox: Box[int] → Box[string]
	strFromInt := TransformBox(intBox, func(n int) string {
		return fmt.Sprintf("value is %d", n)
	})
	fmt.Println("TransformBox:", strFromInt)

	// --- Part 7: Stack as generic type ---
	fmt.Println("\n--- Stack[T] — generic data structure ---")
	s := &Stack[string]{}
	s.Push("first")
	s.Push("second")
	s.Push("third")
	fmt.Printf("Stack has %d items\n", s.Len())
	for {
		item, ok := s.Pop()
		if !ok {
			break
		}
		fmt.Printf("  Popped: %q\n", item)
	}

	// Stack of ints — same type, different instantiation
	ns := &Stack[int]{}
	for _, n := range []int{100, 200, 300} {
		ns.Push(n)
	}
	top, _ := ns.Pop()
	fmt.Printf("int Stack top: %d\n", top)

	// --- Summary ---
	fmt.Println("\n--- Summary ---")
	fmt.Println("1. Type params use [] syntax: func F[T Constraint](v T) T")
	fmt.Println("2. Type inference works when T appears in parameter list")
	fmt.Println("3. MakeZero[T]() needs explicit T — return-only type param")
	fmt.Println("4. Multiple type params: [K, V any], [T, U any]")
	fmt.Println("5. Methods CANNOT have their own type params — use package functions")
	fmt.Println("6. Generic functions: stateless algorithms")
	fmt.Println("7. Generic types: stateful data structures that store T")
}
