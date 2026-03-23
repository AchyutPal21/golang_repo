// 03_constraints.go
//
// CONSTRAINTS — Defining What Type Parameters Can Do
// ==================================================
// A constraint is an INTERFACE that describes the SET OF TYPES
// a type parameter may be instantiated with. This file covers:
//
//   - The "any" constraint (no restriction)
//   - The "comparable" constraint (supports ==, !=)
//   - Union types in constraints (int | float64 | string)
//   - The ~ (tilde/underlying-type) operator
//   - Defining your own constraints
//   - Using constraints from golang.org/x/exp/constraints (or stdlib)
//   - Constraints as documentation and safety guarantees
//   - The "interface as constraint" mental model
//
// KEY INSIGHT: In Go, every constraint IS an interface.
// The type parameter system extends interfaces to describe not just
// method sets, but also TYPE SETS (the set of types that satisfy the interface).

package main

import (
	"fmt"
	"strings"
)

// =============================================================================
// PART 1: THE "any" CONSTRAINT
// =============================================================================
//
// "any" = interface{} = no restriction at all.
// A type parameter constrained by "any" can be instantiated with ANY type:
// int, string, []byte, chan struct{}, your custom struct, anything.
//
// WHEN TO USE "any":
// When your function doesn't need to perform any operations on T —
// it just stores, passes, or returns T. The moment you do anything
// with T (compare, add, call a method), you need a tighter constraint.

func Store[T any](v T) *T {
	// We can store T (assign to pointer), but we can't compare it,
	// do arithmetic on it, or call any methods on it.
	// "any" is appropriate here.
	return &v
}

// Observe: the following would NOT compile with "any":
//
//   func Bad[T any](a, b T) bool { return a == b }  // COMPILE ERROR
//   // T is not constrained to be comparable — might be a slice or map.
//
// Slices and maps are NOT comparable in Go (can't use == between them).
// The compiler enforces this through constraints.

// =============================================================================
// PART 2: THE "comparable" CONSTRAINT
// =============================================================================
//
// "comparable" is a built-in constraint that allows == and != operations.
// Types satisfying comparable: all basic types (int, string, bool, pointers,
// structs with all-comparable fields, arrays of comparable types).
// Types NOT satisfying comparable: slices, maps, functions.
//
// Use comparable when your algorithm needs equality checks.

func Equal[T comparable](a, b T) bool {
	return a == b
}

// Contains checks if a slice contains a target value.
// comparable constraint needed for the == check.
func Contains[T comparable](slice []T, target T) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of target, or -1.
func IndexOf[T comparable](slice []T, target T) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}

// Unique returns a new slice with duplicate elements removed, preserving order.
// Uses a map for O(n) lookup — requires comparable keys.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// =============================================================================
// PART 3: UNION TYPES IN CONSTRAINTS
// =============================================================================
//
// You can define a constraint that allows a SPECIFIC SET of types.
// This uses the union operator | in the interface body.
//
// Syntax:
//   type MyConstraint interface {
//       int | float64 | string
//   }
//
// This means: T may be int, OR float64, OR string — nothing else.
// With such a constraint, the compiler knows WHICH operations are valid:
// operations that are valid on ALL members of the union.
//
// int | float64 → arithmetic (+, -, *, /), comparison (<, >, <=, >=) are valid
// int | string  → only == and != are valid (you can't add int and string)

// Integer is a constraint for all signed and unsigned integer types.
// This is similar to what's provided in golang.org/x/exp/constraints.
type Integer interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 | uintptr
}

// Float is a constraint for float32 and float64.
type Float interface {
	float32 | float64
}

// Ordered is a constraint for types that support <, >, <=, >=.
// This is the most important constraint for sorting, min/max algorithms.
// string supports < and > via lexicographic comparison.
type Ordered interface {
	Integer | Float | ~string
}

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

// Clamp restricts v to the range [lo, hi].
func Clamp[T Ordered](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// =============================================================================
// PART 4: THE ~ (TILDE) OPERATOR — UNDERLYING TYPE CONSTRAINT
// =============================================================================
//
// Problem: What if someone defines:
//
//   type MyInt int
//
// MyInt's underlying type is int. But without ~, the constraint "int" does NOT
// match MyInt — it only matches the EXACT type "int".
//
//   Min[MyInt](a, b)  // COMPILE ERROR if constraint is "int" not "~int"
//
// The ~ operator means "any type whose UNDERLYING TYPE is T."
//
//   ~int = matches int, MyInt, type Score int, type Count int, ...
//
// In our Ordered constraint above, ~string means it matches not just string
// but also type UserID string, type Color string, etc.
//
// RULE OF THUMB:
// Almost always use ~ when listing types in a constraint.
// The exceptions are when you SPECIFICALLY want to exclude named types —
// which is very rare.
//
// Why does this matter?
// Go encourages creating named types for semantic clarity:
//   type UserID string
//   type Score float64
//   type Celsius float32
//
// Without ~, none of these would work with your generic functions.
// With ~, they all do — and operations like < are still valid because
// the underlying type (string, float64, float32) supports them.

// Let's demonstrate with a named type:
type Score float64
type Temperature float32

// BestOf returns the higher of two Scores.
// Works because Score's underlying type is float64, which is in ~Float.
func BestOf[T Ordered](a, b T) T {
	return Max(a, b)
}

// =============================================================================
// PART 5: CONSTRAINTS WITH METHOD REQUIREMENTS
// =============================================================================
//
// Constraints can require BOTH a specific type set AND specific methods.
// This is the full power of the "interface as constraint" model.
//
// Syntax:
//   type Stringer interface {
//       String() string
//   }
//
// When used as a constraint, T must have a String() method.
// This is a METHOD constraint — any type with that method qualifies.
//
// You can combine method requirements with type unions:
//   type Printable interface {
//       ~int | ~string
//       String() string
//   }
// But this is unusual — such an intersection is often empty (int has no String()).
// More commonly, method-only constraints are used for algorithm flexibility.

// Stringer is the classic interface for types that can describe themselves.
// When used as a constraint, any type with String() string works.
type Stringer interface {
	String() string
}

// PrintAll prints a slice of any Stringer — works at compile time.
func PrintAll[T Stringer](items []T) {
	for _, item := range items {
		fmt.Println(item.String())
	}
}

// A type that satisfies Stringer:
type Point struct {
	X, Y float64
}

func (p Point) String() string {
	return fmt.Sprintf("(%.2f, %.2f)", p.X, p.Y)
}

// =============================================================================
// PART 6: COMBINING CONSTRAINTS — TYPE SETS AND INTERFACES
// =============================================================================
//
// Constraints can embed other interfaces (constraint or regular):

// Number is a constraint for types that support arithmetic.
// We use ~ to support named types with numeric underlying types.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Summable adds the Number constraint to a function.
func Sum[T Number](nums []T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

// Average computes the mean. Returns float64 always.
// We need T to be a Number so we can sum it, and we convert to float64
// using the float64() conversion (valid because T's underlying type is numeric).
func Average[T Number](nums []T) float64 {
	if len(nums) == 0 {
		return 0
	}
	var total T
	for _, n := range nums {
		total += n
	}
	return float64(total) / float64(len(nums))
}

// =============================================================================
// PART 7: INLINE CONSTRAINTS (anonymous interface in type param list)
// =============================================================================
//
// You don't always need to name a constraint. You can inline it:
//
//   func Add[T interface{ ~int | ~float64 }](a, b T) T { return a + b }
//
// Named constraints are better for reuse and readability.
// Inline constraints are convenient for one-off functions.

func AbsDiff[T interface{ ~int | ~float64 }](a, b T) T {
	if a > b {
		return a - b
	}
	return b - a
}

// =============================================================================
// PART 8: THE "comparable" GOTCHA — Interfaces at Runtime
// =============================================================================
//
// comparable allows == and != at compile time.
// However, there is a RUNTIME GOTCHA:
//
// If T is constrained by comparable, and you pass an INTERFACE VALUE as T,
// the == check succeeds at compile time but may PANIC at runtime if the
// underlying concrete type of the interface is not comparable (e.g., a slice).
//
//   type Doer interface{ Do() }
//   type SliceDoer []int
//   func (s SliceDoer) Do() {}
//
//   // SliceDoer implements Doer, but is NOT comparable (it's a slice).
//   // var d Doer = SliceDoer{1,2,3}
//   // Equal[Doer](d, d)  // compiles, but PANICS at runtime!
//
// Why? Interface values themselves are comparable (you can == two interface values),
// so "Doer" satisfies comparable. But comparing two interface values uses the
// dynamic type's equality — and if the dynamic type is a slice, that panics.
//
// MITIGATION: For user-facing generic APIs, document that T must be concretely
// comparable (not an interface type wrapping a non-comparable value).
// This is a known Go generics wart with no compile-time fix yet.
//
// In practice: when T is a concrete type (int, string, struct), this never bites you.
// Only problematic when T is instantiated with an interface type.

// SafeEqual shows the comparison — works fine for concrete types.
func SafeEqual[T comparable](a, b T) bool {
	// For concrete T: fully safe.
	// For interface T wrapping non-comparable: runtime panic possible.
	return a == b
}

// =============================================================================
// PART 9: CONSTRAINT TAXONOMY SUMMARY
// =============================================================================
//
// BUILT-IN CONSTRAINTS:
//   any        — no restriction; only store/pass/return T
//   comparable — supports == and !=
//
// COMMON CUSTOM CONSTRAINTS:
//   Integer  — all signed and unsigned integer types
//   Float    — float32 | float64
//   Ordered  — Integer | Float | ~string (supports < > <= >=)
//   Number   — Integer | Float (supports arithmetic)
//   Stringer — has String() string method
//
// OPERATOR REFERENCE:
//   |   type union: T can be int OR string OR float64 ...
//   ~T  underlying type: any type whose underlying type is T
//   methods in interface body: T must have those methods
//   embedded interfaces: T must satisfy all embedded interfaces

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("CONSTRAINTS — Defining What Type Parameters Can Do")
	fmt.Println(strings.Repeat("=", 60))

	// --- Part 1: any ---
	fmt.Println("\n--- any constraint ---")
	p1 := Store(42)
	p2 := Store("gopher")
	p3 := Store([]int{1, 2, 3})
	fmt.Printf("Store(42):        *int   → %d\n", *p1)
	fmt.Printf("Store(\"gopher\"): *string → %q\n", *p2)
	fmt.Printf("Store([]int{...}): *[]int → %v\n", *p3)

	// --- Part 2: comparable ---
	fmt.Println("\n--- comparable constraint ---")
	fmt.Println("Equal(1, 1):           ", Equal(1, 1))
	fmt.Println("Equal(\"a\", \"b\"):     ", Equal("a", "b"))
	fmt.Println("Contains([1,2,3], 2):  ", Contains([]int{1, 2, 3}, 2))
	fmt.Println("Contains([a,b,c], d):  ", Contains([]string{"a", "b", "c"}, "d"))
	fmt.Println("IndexOf([10,20,30],20):", IndexOf([]int{10, 20, 30}, 20))

	dupes := []string{"go", "rust", "go", "c", "rust", "go"}
	fmt.Printf("Unique(%v): %v\n", dupes, Unique(dupes))
	fmt.Printf("Unique(%v): %v\n", []int{1, 2, 1, 3, 2}, Unique([]int{1, 2, 1, 3, 2}))

	// --- Part 3 & 4: Union types and ~ ---
	fmt.Println("\n--- Ordered constraint (union + ~) ---")
	fmt.Println("Min(3, 7):         ", Min(3, 7))
	fmt.Println("Max(3.14, 2.71):   ", Max(3.14, 2.71))
	fmt.Println("Min(\"apple\",\"go\"):", Min("apple", "go"))
	fmt.Println("Clamp(15, 0, 10):  ", Clamp(15, 0, 10))
	fmt.Println("Clamp(-5, 0, 10):  ", Clamp(-5, 0, 10))
	fmt.Println("Clamp(7, 0, 10):   ", Clamp(7, 0, 10))

	// Named types with underlying numeric type
	var s1, s2 Score = 95.5, 87.3
	var t1, t2 Temperature = 100.0, 98.6
	fmt.Printf("BestOf Score(%.1f, %.1f) = %.1f\n", s1, s2, BestOf(s1, s2))
	fmt.Printf("BestOf Temperature(%.1f, %.1f) = %.1f\n", t1, t2, BestOf(t1, t2))

	// --- Part 5: Method constraints ---
	fmt.Println("\n--- Method constraint (Stringer) ---")
	points := []Point{{1, 2}, {3, 4}, {5, 6}}
	PrintAll(points) // Point satisfies Stringer

	// --- Part 6: Number constraint ---
	fmt.Println("\n--- Number constraint (arithmetic) ---")
	ints := []int{1, 2, 3, 4, 5}
	floats := []float64{1.5, 2.5, 3.5}
	fmt.Printf("Sum(%v)            = %d\n", ints, Sum(ints))
	fmt.Printf("Sum(%v)        = %.1f\n", floats, Sum(floats))
	fmt.Printf("Average(%v)        = %.2f\n", ints, Average(ints))
	fmt.Printf("Average(%v) = %.2f\n", floats, Average(floats))

	// Custom named type
	type Celsius float64
	temps := []Celsius{36.6, 37.0, 38.5, 36.9}
	fmt.Printf("Sum(Celsius temps) = %.1f°C\n", Sum(temps))
	fmt.Printf("Average(Celsius)   = %.2f°C\n", Average(temps))

	// --- Part 7: Inline constraint ---
	fmt.Println("\n--- Inline constraint ---")
	fmt.Println("AbsDiff(10, 3):    ", AbsDiff(10, 3))
	fmt.Println("AbsDiff(3.14, 2.0):", AbsDiff(3.14, 2.0))

	// --- Summary ---
	fmt.Println("\n--- Constraint Quick Reference ---")
	rows := []struct{ constraint, meaning string }{
		{"any", "no restriction — only store/pass/return T"},
		{"comparable", "supports == and != (not slices, maps, funcs)"},
		{"~int", "int OR any named type with underlying type int"},
		{"int | float64", "exactly int OR float64 (not ~, so no named types)"},
		{"~int | ~float64", "int/named-int OR float64/named-float64"},
		{"Integer | Float", "embed sub-constraints via union"},
		{"interface{Method()}", "T must have Method()"},
	}
	for _, r := range rows {
		fmt.Printf("  %-30s → %s\n", r.constraint, r.meaning)
	}
}
