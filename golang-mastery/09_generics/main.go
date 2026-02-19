package main

// =============================================================================
// MODULE 09: GENERICS — Type parameters (Go 1.18+)
// =============================================================================
// Run: go run 09_generics/main.go
//
// Generics allow you to write functions and types that work with multiple types
// while maintaining type safety — no interface{} + type assertion needed.
// =============================================================================

import (
	"cmp"
	"fmt"
	"strings"
)

// =============================================================================
// GENERIC FUNCTIONS — basic syntax
// =============================================================================
// func FuncName[TypeParam Constraint](params) returnType
//   TypeParam = the type parameter name (T, K, V are common by convention)
//   Constraint = what types are allowed (any, comparable, custom interface)

// any = no constraint — any type is allowed
func PrintSlice[T any](s []T) {
	for i, v := range s {
		fmt.Printf("  [%d] %v\n", i, v)
	}
}

// Multiple type parameters
func Map[T, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

func Reduce[T, R any](slice []T, initial R, fn func(R, T) R) R {
	result := initial
	for _, v := range slice {
		result = fn(result, v)
	}
	return result
}

// comparable constraint — supports == and !=
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func IndexOf[T comparable](slice []T, item T) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// =============================================================================
// CONSTRAINTS — defining what types are allowed
// =============================================================================

// Type constraint using interface with type list
type Number interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

// ~ means: this type OR any type with this as underlying type
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// Using cmp.Ordered from stdlib (Go 1.21+) — use this instead of defining your own
// cmp.Ordered = all types that support < <= >= >

func Sum[T Number](nums []T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func MinSlice[T cmp.Ordered](slice []T) T {
	if len(slice) == 0 {
		panic("empty slice")
	}
	min := slice[0]
	for _, v := range slice[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// =============================================================================
// GENERIC TYPES — structs with type parameters
// =============================================================================

// Generic Stack
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
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Generic Queue
type Queue[T any] struct {
	items []T
}

func (q *Queue[T]) Enqueue(item T) {
	q.items = append(q.items, item)
}

func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	front := q.items[0]
	q.items = q.items[1:]
	return front, true
}

func (q *Queue[T]) Len() int { return len(q.items) }

// Generic Pair
type Pair[A, B any] struct {
	First  A
	Second B
}

func NewPair[A, B any](a A, b B) Pair[A, B] {
	return Pair[A, B]{First: a, Second: b}
}

// Generic Map (key-value)
type OrderedMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{values: make(map[K]V)}
}

func (m *OrderedMap[K, V]) Set(key K, val V) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = val
}

func (m *OrderedMap[K, V]) Get(key K) (V, bool) {
	v, ok := m.values[key]
	return v, ok
}

func (m *OrderedMap[K, V]) Keys() []K {
	return m.keys
}

// Generic Option/Maybe type
type Option[T any] struct {
	value    T
	hasValue bool
}

func Some[T any](v T) Option[T]    { return Option[T]{value: v, hasValue: true} }
func None[T any]() Option[T]       { return Option[T]{} }
func (o Option[T]) IsSome() bool   { return o.hasValue }
func (o Option[T]) IsNone() bool   { return !o.hasValue }
func (o Option[T]) Unwrap() T {
	if !o.hasValue {
		panic("Option is None")
	}
	return o.value
}
func (o Option[T]) UnwrapOr(def T) T {
	if !o.hasValue {
		return def
	}
	return o.value
}

// Generic Result type (better error handling)
type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](v T) Result[T]      { return Result[T]{value: v} }
func Err[T any](e error) Result[T] { return Result[T]{err: e} }
func (r Result[T]) IsOk() bool     { return r.err == nil }
func (r Result[T]) Unwrap() (T, error) { return r.value, r.err }

// =============================================================================
// REAL WORLD UTILITY FUNCTIONS
// =============================================================================

// Keys extracts keys from a map
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values extracts values from a map
func Values[K comparable, V any](m map[K]V) []V {
	vals := make([]V, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}

// Unique removes duplicates preserving order
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	var result []T
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Chunk splits a slice into chunks of given size
func Chunk[T any](slice []T, size int) [][]T {
	var chunks [][]T
	for len(slice) > 0 {
		if len(slice) < size {
			size = len(slice)
		}
		chunks = append(chunks, slice[:size])
		slice = slice[size:]
	}
	return chunks
}

// Zip combines two slices into slice of pairs
func Zip[A, B any](a []A, b []B) []Pair[A, B] {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]Pair[A, B], n)
	for i := 0; i < n; i++ {
		result[i] = NewPair(a[i], b[i])
	}
	return result
}

// Flatten flattens a 2D slice into 1D
func Flatten[T any](slices [][]T) []T {
	var result []T
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// GroupBy groups elements by a key function
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		k := keyFn(v)
		result[k] = append(result[k], v)
	}
	return result
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 09: GENERICS ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Generic functions
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Generic Functions ---")

	fmt.Println("PrintSlice[int]:")
	PrintSlice([]int{1, 2, 3})

	fmt.Println("PrintSlice[string]:")
	PrintSlice([]string{"go", "is", "great"})

	fmt.Println("PrintSlice[bool]:")
	PrintSlice([]bool{true, false, true})

	// Map, Filter, Reduce
	fmt.Println("\n--- Map / Filter / Reduce ---")

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	doubled := Map(nums, func(n int) int { return n * 2 })
	fmt.Println("doubled:", doubled)

	strs := Map(nums, func(n int) string { return fmt.Sprintf("#%d", n) })
	fmt.Println("as strings:", strs)

	evens := Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("evens:", evens)

	sum := Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Println("sum:", sum)

	product := Reduce([]int{1, 2, 3, 4, 5}, 1, func(acc, n int) int { return acc * n })
	fmt.Println("product:", product)

	// Contains and IndexOf
	fmt.Println("\nContains(nums, 7):", Contains(nums, 7))
	fmt.Println("Contains(nums, 11):", Contains(nums, 11))
	fmt.Println("IndexOf(nums, 5):", IndexOf(nums, 5))

	// -------------------------------------------------------------------------
	// SECTION 2: Number constraints
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Number Constraints ---")

	ints := []int{3, 1, 4, 1, 5, 9, 2, 6}
	floats := []float64{3.14, 1.41, 2.71, 1.73}

	fmt.Println("Sum(ints):", Sum(ints))
	fmt.Printf("Sum(floats): %.2f\n", Sum(floats))

	fmt.Println("Min(3, 5):", Min(3, 5))
	fmt.Println("Min(3.14, 2.71):", Min(3.14, 2.71))
	fmt.Println("Min('a', 'z'):", string(Min('a', 'z')))

	fmt.Println("MinSlice:", MinSlice(ints))
	fmt.Println("MinSlice strings:", MinSlice([]string{"banana", "apple", "cherry"}))

	// -------------------------------------------------------------------------
	// SECTION 3: Generic Stack
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Generic Stack ---")

	intStack := &Stack[int]{}
	intStack.Push(1)
	intStack.Push(2)
	intStack.Push(3)

	fmt.Println("Stack len:", intStack.Len())
	if top, ok := intStack.Peek(); ok {
		fmt.Println("Peek:", top)
	}

	for !intStack.IsEmpty() {
		val, _ := intStack.Pop()
		fmt.Print(val, " ")
	}
	fmt.Println()

	// String stack
	strStack := &Stack[string]{}
	strStack.Push("hello")
	strStack.Push("world")
	v, _ := strStack.Pop()
	fmt.Println("Popped:", v)

	// -------------------------------------------------------------------------
	// SECTION 4: Generic Queue
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Generic Queue ---")

	q := &Queue[string]{}
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")

	for q.Len() > 0 {
		item, _ := q.Dequeue()
		fmt.Print(item, " ")
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// SECTION 5: Pair and OrderedMap
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Pair ---")

	p := NewPair("age", 25)
	fmt.Printf("Pair: %s=%v\n", p.First, p.Second)

	p2 := NewPair(true, []int{1, 2, 3})
	fmt.Printf("Pair: %v → %v\n", p2.First, p2.Second)

	fmt.Println("\n--- OrderedMap ---")

	om := NewOrderedMap[string, int]()
	om.Set("c", 3)
	om.Set("a", 1)
	om.Set("b", 2)

	// Preserves insertion order
	for _, k := range om.Keys() {
		v, _ := om.Get(k)
		fmt.Printf("  %s: %d\n", k, v)
	}

	// -------------------------------------------------------------------------
	// SECTION 6: Option and Result types
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Option[T] ---")

	opt1 := Some(42)
	opt2 := None[int]()

	fmt.Println("Some(42).IsSome():", opt1.IsSome())
	fmt.Println("None.IsNone():", opt2.IsNone())
	fmt.Println("Some(42).Unwrap():", opt1.Unwrap())
	fmt.Println("None.UnwrapOr(99):", opt2.UnwrapOr(99))

	fmt.Println("\n--- Result[T] ---")

	r1 := Ok(100)
	r2 := Err[int](fmt.Errorf("something failed"))

	v1, e1 := r1.Unwrap()
	fmt.Println("Ok:", v1, e1)

	v2, e2 := r2.Unwrap()
	fmt.Println("Err:", v2, e2)

	// -------------------------------------------------------------------------
	// SECTION 7: Utility functions
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Utility Functions ---")

	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Println("Keys:", Keys(m))
	fmt.Println("Values:", Values(m))

	dupes := []int{1, 2, 2, 3, 3, 3, 4}
	fmt.Println("Unique:", Unique(dupes))

	chunked := Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3)
	fmt.Println("Chunk:", chunked)

	zipped := Zip([]string{"a", "b", "c"}, []int{1, 2, 3})
	for _, pair := range zipped {
		fmt.Printf("  %s → %d\n", pair.First, pair.Second)
	}

	flat := Flatten([][]int{{1, 2}, {3, 4}, {5, 6}})
	fmt.Println("Flatten:", flat)

	type Person struct{ Name, Dept string }
	people := []Person{
		{"Alice", "Eng"}, {"Bob", "HR"},
		{"Charlie", "Eng"}, {"Dave", "HR"}, {"Eve", "Eng"},
	}
	groups := GroupBy(people, func(p Person) string { return p.Dept })
	for dept, members := range groups {
		names := Map(members, func(p Person) string { return p.Name })
		fmt.Printf("  %s: %s\n", dept, strings.Join(names, ", "))
	}

	// -------------------------------------------------------------------------
	// SECTION 8: Type inference in generics
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Type Inference ---")

	// Go infers type parameters from arguments — no need to specify them manually
	fmt.Println(Contains([]int{1, 2, 3}, 2))       // T=int inferred
	fmt.Println(Contains([]string{"a", "b"}, "c")) // T=string inferred
	fmt.Println(Min(3, 7))                          // T=int inferred
	fmt.Println(Min(3.14, 2.71))                    // T=float64 inferred

	// Explicit type parameter — when inference isn't enough
	emptyNone := None[string]() // can't infer T here — must specify
	fmt.Println("None[string]:", emptyNone.IsNone())

	fmt.Println("\n=== MODULE 09 COMPLETE ===")
}
