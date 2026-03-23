// FILE: 09_generics/05_generic_functions.go
// TOPIC: Generic Functions — Map, Filter, Reduce, Contains, Keys, Values, Ptr
//
// Run: go run 09_generics/05_generic_functions.go

package main

import "fmt"

// ── CONSTRAINTS ──────────────────────────────────────────────────────────────
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// ── THE FUNCTIONAL TRIO ───────────────────────────────────────────────────────

// Map transforms each element: []T → []R
func Map[T, R any](s []T, f func(T) R) []R {
	result := make([]R, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

// Filter keeps elements where predicate is true: []T → []T
func Filter[T any](s []T, f func(T) bool) []T {
	var result []T
	for _, v := range s {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

// Reduce folds a slice into a single value
func Reduce[T, Acc any](s []T, initial Acc, f func(Acc, T) Acc) Acc {
	acc := initial
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// ── UTILITY FUNCTIONS ─────────────────────────────────────────────────────────

func Contains[T comparable](s []T, v T) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

func Find[T any](s []T, f func(T) bool) (T, bool) {
	for _, v := range s {
		if f(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func Values[K comparable, V any](m map[K]V) []V {
	vals := make([]V, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}

// Ptr returns a pointer to the given value.
// Useful when you need a *T literal: Ptr("hello") → *string
func Ptr[T any](v T) *T { return &v }

// Must panics if err is non-nil — like regexp.MustCompile but generic
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Sum adds all numbers in a slice
func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

// Min/Max
func Min[T Number | ~string](a, b T) T {
	if a < b { return a }
	return b
}

func Max[T Number | ~string](a, b T) T {
	if a > b { return a }
	return b
}

// Chunk splits a slice into chunks of size n
func Chunk[T any](s []T, n int) [][]T {
	var result [][]T
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) { end = len(s) }
		result = append(result, s[i:end])
	}
	return result
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Generic Functions")
	fmt.Println("════════════════════════════════════════")

	// ── Map ──────────────────────────────────────────────────────────────
	fmt.Println("\n── Map ──")
	ints := []int{1, 2, 3, 4, 5}
	doubled := Map(ints, func(n int) int { return n * 2 })
	fmt.Printf("  double ints: %v\n", doubled)

	strs := Map(ints, func(n int) string { return fmt.Sprintf("item%d", n) })
	fmt.Printf("  to strings: %v\n", strs)

	// ── Filter ───────────────────────────────────────────────────────────
	fmt.Println("\n── Filter ──")
	evens := Filter(ints, func(n int) bool { return n%2 == 0 })
	fmt.Printf("  evens: %v\n", evens)
	long := Filter([]string{"hi", "hello", "go", "golang"}, func(s string) bool { return len(s) > 2 })
	fmt.Printf("  long strings: %v\n", long)

	// ── Reduce ───────────────────────────────────────────────────────────
	fmt.Println("\n── Reduce ──")
	sum := Reduce(ints, 0, func(acc, n int) int { return acc + n })
	fmt.Printf("  sum: %d\n", sum)
	product := Reduce(ints, 1, func(acc, n int) int { return acc * n })
	fmt.Printf("  product: %d\n", product)
	concat := Reduce([]string{"a", "b", "c"}, "", func(acc, s string) string { return acc + s })
	fmt.Printf("  concat: %q\n", concat)

	// ── Chaining ──────────────────────────────────────────────────────────
	fmt.Println("\n── Chaining Map+Filter+Reduce ──")
	result := Reduce(
		Filter(
			Map([]int{1, 2, 3, 4, 5, 6}, func(n int) int { return n * n }),
			func(n int) bool { return n > 5 },
		),
		0,
		func(acc, n int) int { return acc + n },
	)
	fmt.Printf("  sum of squares > 5: %d\n", result)

	// ── Utility functions ─────────────────────────────────────────────────
	fmt.Println("\n── Contains / Find ──")
	fmt.Printf("  Contains([1,2,3], 2): %v\n", Contains([]int{1, 2, 3}, 2))
	fmt.Printf("  Contains([1,2,3], 9): %v\n", Contains([]int{1, 2, 3}, 9))

	v, ok := Find([]int{1, 2, 3, 4}, func(n int) bool { return n > 2 })
	fmt.Printf("  Find(>2): %d, found=%v\n", v, ok)

	// ── Map keys/values ───────────────────────────────────────────────────
	fmt.Println("\n── Keys / Values ──")
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Printf("  Keys:   %v\n", Keys(m))
	fmt.Printf("  Values: %v\n", Values(m))

	// ── Ptr — pointer to literal ─────────────────────────────────────────
	fmt.Println("\n── Ptr ──")
	s := Ptr("hello")
	n := Ptr(42)
	fmt.Printf("  Ptr(\"hello\"): %p → %q\n", s, *s)
	fmt.Printf("  Ptr(42):      %p → %d\n", n, *n)

	// ── Sum / Min / Max ──────────────────────────────────────────────────
	fmt.Println("\n── Sum / Min / Max ──")
	fmt.Printf("  Sum([1..5]): %d\n", Sum(ints))
	fmt.Printf("  Min(3,7): %d\n", Min(3, 7))
	fmt.Printf("  Max(\"apple\",\"banana\"): %q\n", Max("apple", "banana"))

	// ── Chunk ─────────────────────────────────────────────────────────────
	fmt.Println("\n── Chunk ──")
	chunks := Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3)
	fmt.Printf("  Chunk([1..7], 3): %v\n", chunks)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Map[T,R] / Filter[T] / Reduce[T,Acc] — functional trio")
	fmt.Println("  Contains[T comparable] / Find[T any]")
	fmt.Println("  Keys[K,V] / Values[K,V] — map utilities")
	fmt.Println("  Ptr[T] — pointer to value (useful for optional fields)")
	fmt.Println("  Sum[T Number] — typed generic arithmetic")
	fmt.Println("  Type inference works for most calls — no explicit [T] needed")
}
