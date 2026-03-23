// FILE: 09_generics/06_constraints_advanced.go
// TOPIC: Advanced Constraints — interface as constraint, union types, ~T, comparable gotchas
//
// Run: go run 09_generics/06_constraints_advanced.go

package main

import "fmt"

// ── CONSTRAINT WITH METHODS ───────────────────────────────────────────────────
// A constraint can require methods, not just types.
// This is an interface used as a TYPE CONSTRAINT, not a value type.

type Stringer interface {
	String() string
}

// PrintAll prints any slice of Stringer-implementing elements
func PrintAll[T Stringer](items []T) {
	for _, item := range items {
		fmt.Printf("  %s\n", item.String())
	}
}

type Color int

const (
	Red Color = iota
	Green
	Blue
)

func (c Color) String() string {
	return [...]string{"Red", "Green", "Blue"}[c]
}

type Point struct{ X, Y int }

func (p Point) String() string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

// ── UNION CONSTRAINTS WITH ~ ──────────────────────────────────────────────────
// ~T means "any type whose UNDERLYING type is T".
// This includes custom types based on T.
// Without ~: only the exact type T matches.

type MyInt int  // underlying type is int

// Without ~int: MyInt would NOT satisfy this:
type JustInt interface{ ~int }

// With ~int: MyInt satisfies this:
type IntLike interface{ ~int }

func Double[T IntLike](v T) T { return v + v }

// ── INTERSECTION CONSTRAINTS ──────────────────────────────────────────────────
// Multiple embedded interfaces in a constraint = ALL must be satisfied.

type Ordered interface {
	~int | ~float64 | ~string
}

type Printable interface {
	String() string
}

// A type satisfying both Ordered and Printable:
type Score int

func (s Score) String() string { return fmt.Sprintf("Score(%d)", s) }

// Constraint requiring both comparable + Stringer:
type ComparableStringer interface {
	comparable
	Stringer
}

func FindDuplicate[T ComparableStringer](items []T) (T, bool) {
	seen := make(map[T]bool)
	for _, item := range items {
		if seen[item] {
			return item, true
		}
		seen[item] = true
	}
	var zero T
	return zero, false
}

// ── COMPARABLE GOTCHA ─────────────────────────────────────────────────────────
// comparable means the type supports ==.
// BUT: interface types are comparable at compile time, may PANIC at runtime
// if the underlying concrete type is not comparable (e.g., slice in interface).

func Equal[T comparable](a, b T) bool { return a == b }

// ── TYPE PARAMETER IN RETURN TYPE ────────────────────────────────────────────

func Zero[T any]() T {
	var z T
	return z  // returns the zero value of T
}

func MakeSlice[T any](n int, fill T) []T {
	s := make([]T, n)
	for i := range s {
		s[i] = fill
	}
	return s
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Advanced Constraints")
	fmt.Println("════════════════════════════════════════")

	// ── Method constraint ─────────────────────────────────────────────────
	fmt.Println("\n── Method constraint (Stringer) ──")
	colors := []Color{Red, Green, Blue}
	PrintAll(colors)
	points := []Point{{1, 2}, {3, 4}}
	PrintAll(points)

	// ── ~T underlying type ────────────────────────────────────────────────
	fmt.Println("\n── ~int (underlying type) ──")
	fmt.Printf("  Double(int(5)):   %d\n", Double(5))
	fmt.Printf("  Double(MyInt(7)): %d\n", Double(MyInt(7)))  // works because ~int

	// ── ComparableStringer constraint ─────────────────────────────────────
	fmt.Println("\n── Intersection constraint (comparable + Stringer) ──")
	scores := []Score{Score(90), Score(85), Score(90), Score(70)}
	dup, found := FindDuplicate(scores)
	fmt.Printf("  FindDuplicate: %v, found=%v\n", dup, found)

	// ── Zero value via generics ───────────────────────────────────────────
	fmt.Println("\n── Zero[T]() and MakeSlice[T]() ──")
	fmt.Printf("  Zero[int]():    %d\n", Zero[int]())
	fmt.Printf("  Zero[string](): %q\n", Zero[string]())
	fmt.Printf("  Zero[bool]():   %v\n", Zero[bool]())
	fmt.Printf("  MakeSlice[int](5, 42): %v\n", MakeSlice(5, 42))
	fmt.Printf("  MakeSlice[string](3, \"go\"): %v\n", MakeSlice(3, "go"))

	// ── comparable at compile vs runtime ─────────────────────────────────
	fmt.Println("\n── comparable ──")
	fmt.Printf("  Equal(1, 1):         %v\n", Equal(1, 1))
	fmt.Printf("  Equal(\"a\", \"b\"):     %v\n", Equal("a", "b"))
	fmt.Printf("  Equal(Point{1,2}, Point{1,2}): %v\n", Equal(Point{1, 2}, Point{1, 2}))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Method constraints: interface with methods, not just types")
	fmt.Println("  ~T: matches exact T AND all types with underlying type T")
	fmt.Println("  Intersection: embed multiple interfaces in one constraint")
	fmt.Println("  comparable: supports ==, but interface values may panic at runtime")
	fmt.Println("  Zero[T](): returns zero value of T — useful in generic containers")
}
