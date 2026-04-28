// FILE: book/part2_core_language/chapter24_generics/examples/01_type_params/main.go
// CHAPTER: 24 — Generics: Type Parameters and Constraints
// TOPIC: Type parameters, any constraint, comparable, generic functions,
//        generic types (Stack, Map, Pair), type inference.
//
// Run (from the chapter folder):
//   go run ./examples/01_type_params

package main

import "fmt"

// --- Generic functions ---

// Map transforms a slice, applying f to each element.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Filter returns elements for which keep returns true.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce folds a slice into a single value.
func Reduce[T, U any](s []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// Contains reports whether v is in s.
func Contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Keys returns the keys of a map.
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// --- Generic types ---

// Pair holds two values of possibly different types.
type Pair[A, B any] struct {
	First  A
	Second B
}

func NewPair[A, B any](a A, b B) Pair[A, B] {
	return Pair[A, B]{First: a, Second: b}
}

// Optional holds a value that may be absent.
type Optional[T any] struct {
	value T
	ok    bool
}

func Some[T any](v T) Optional[T] { return Optional[T]{value: v, ok: true} }
func None[T any]() Optional[T]    { return Optional[T]{} }

func (o Optional[T]) Get() (T, bool) { return o.value, o.ok }

func (o Optional[T]) OrElse(def T) T {
	if o.ok {
		return o.value
	}
	return def
}

func main() {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Map: int → string
	strs := Map(nums, func(n int) string { return fmt.Sprintf("%d²=%d", n, n*n) })
	fmt.Println("Map:", strs[:3], "...")

	// Filter
	evens := Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("Filter evens:", evens)

	// Reduce: sum
	sum := Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Println("Reduce sum:", sum)

	// Contains
	fmt.Println("Contains 5:", Contains(nums, 5))
	fmt.Println("Contains 11:", Contains(nums, 11))
	fmt.Println("Contains 'go':", Contains([]string{"go", "rust", "python"}, "go"))

	fmt.Println()

	// Keys
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Println("Keys len:", len(Keys(m)))

	fmt.Println()

	// Pair — type inferred
	p := NewPair("hello", 42)
	fmt.Printf("Pair: (%s, %d)\n", p.First, p.Second)

	p2 := NewPair(true, []int{1, 2, 3})
	fmt.Printf("Pair: (%v, %v)\n", p2.First, p2.Second)

	fmt.Println()

	// Optional
	opt := Some(42)
	if v, ok := opt.Get(); ok {
		fmt.Println("Some:", v)
	}
	none := None[int]()
	fmt.Println("None OrElse:", none.OrElse(99))
}
