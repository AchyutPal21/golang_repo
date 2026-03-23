// 04_generic_types.go
//
// GENERIC TYPES — Data Structures with Type Parameters
// ====================================================
// Generic TYPES are structs (or other type definitions) that have type
// parameters. They're the correct tool when you need a DATA STRUCTURE
// that STORES values of type T and provides operations on them over time.
//
// This file covers:
//   - Generic struct types: Stack[T], Queue[T], Pair[K, V]
//   - Generic type instantiation (when the type is "frozen")
//   - Methods on generic types (and the no-method-type-params rule)
//   - Generic type aliases
//   - Set[T comparable] — a real-world useful type
//   - Option[T] — modeling optional values without nil pointers
//   - When generic types beat interface{} data structures
//   - When to use generic types vs generic functions

package main

import (
	"fmt"
	"strings"
)

// =============================================================================
// PART 1: Stack[T] — The Classic Generic Data Structure
// =============================================================================
//
// A stack is LIFO (Last In, First Out).
// Without generics, you'd write IntStack, StringStack, etc.
// With generics: one implementation, compile-time type safety for all.
//
// The type parameter T is declared on the struct, not on methods.
// All methods that use T simply reference it as an already-declared parameter.

type Stack[T any] struct {
	items []T
	// Why []T and not []interface{}?
	// 1. No boxing: values stored directly, no heap allocation per item
	// 2. Type safety: Push(42) on Stack[string] is a compile error
	// 3. Pop() returns T directly — no type assertion needed by caller
}

// Push adds an item to the top of the stack.
// Note: receiver is (s *Stack[T]) — we include [T] to specify which
// instantiation's receiver this is, but T is NOT re-declared here.
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop removes and returns the top item.
// Returns (zero value of T, false) on empty stack — avoids panic.
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T // zero value of T (0 for int, "" for string, nil for pointer, etc.)
		return zero, false
	}
	n := len(s.items)
	top := s.items[n-1]
	s.items = s.items[:n-1]
	return top, true
}

// Peek returns the top item WITHOUT removing it.
func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Len returns the number of items in the stack.
func (s *Stack[T]) Len() int { return len(s.items) }

// IsEmpty is a convenience method.
func (s *Stack[T]) IsEmpty() bool { return len(s.items) == 0 }

// Clear removes all items.
func (s *Stack[T]) Clear() { s.items = s.items[:0] }

// =============================================================================
// PART 2: Queue[T] — FIFO Generic Data Structure
// =============================================================================
//
// A queue is FIFO (First In, First Out).
// A naive implementation uses a slice with append+slice-off,
// but that's O(n) for Dequeue (shifting elements). We use a
// head pointer for O(1) amortized operations (with occasional garbage).
// For simplicity, we use the basic append approach here.

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
	item := q.items[0]
	q.items = q.items[1:] // O(n) but simple; use ring buffer for performance
	return item, true
}

func (q *Queue[T]) Front() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	return q.items[0], true
}

func (q *Queue[T]) Len() int     { return len(q.items) }
func (q *Queue[T]) IsEmpty() bool { return len(q.items) == 0 }

// =============================================================================
// PART 3: Pair[K, V] — Two Different Type Parameters
// =============================================================================
//
// Pair holds two values of potentially different types.
// This is useful for functions that return two conceptually related values
// as a single unit (avoiding unnamed multiple returns).
//
// It's also the building block for ordered maps, priority queues, etc.

type Pair[K, V any] struct {
	Key   K
	Value V
}

// NewPair is a constructor generic function.
func NewPair[K, V any](key K, value V) Pair[K, V] {
	return Pair[K, V]{Key: key, Value: value}
}

func (p Pair[K, V]) String() string {
	return fmt.Sprintf("(%v, %v)", p.Key, p.Value)
}

// Swap returns a new Pair with Key and Value exchanged.
// We can write this as a PACKAGE FUNCTION (workaround for no-method-type-params):
func SwapPair[K, V any](p Pair[K, V]) Pair[V, K] {
	return Pair[V, K]{Key: p.Value, Value: p.Key}
}

// Zip two slices into a slice of Pairs.
func ZipToPairs[K, V any](keys []K, values []V) []Pair[K, V] {
	n := len(keys)
	if len(values) < n {
		n = len(values)
	}
	result := make([]Pair[K, V], n)
	for i := 0; i < n; i++ {
		result[i] = NewPair(keys[i], values[i])
	}
	return result
}

// =============================================================================
// PART 4: Set[T comparable] — A Real-World Useful Generic Type
// =============================================================================
//
// A Set stores unique values and supports membership testing,
// union, intersection, and difference.
//
// The constraint must be "comparable" (not just "any") because we need
// T to be usable as a map key. Slices and maps cannot be map keys,
// so they cannot be stored in a Set. The compiler enforces this.
//
// COMMON MISTAKE: Using "any" as the constraint for a Set.
// This compiles but fails at runtime when you try to use a non-comparable
// type as a map key. Always use "comparable" for map-key type parameters.

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](items ...T) Set[T] {
	s := Set[T]{m: make(map[T]struct{})}
	for _, item := range items {
		s.m[item] = struct{}{}
	}
	return s
}

func (s *Set[T]) Add(item T) {
	s.m[item] = struct{}{}
}

func (s *Set[T]) Remove(item T) {
	delete(s.m, item)
}

func (s *Set[T]) Contains(item T) bool {
	_, ok := s.m[item]
	return ok
}

func (s *Set[T]) Len() int { return len(s.m) }

// ToSlice returns the elements as a slice (order is random — map iteration).
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.m))
	for k := range s.m {
		result = append(result, k)
	}
	return result
}

// Union returns a new Set with all elements from both sets.
// Package-level function (because methods can't introduce new type params —
// and in this case both sets have the same T, so it works as a method too):
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := NewSet[T]()
	for k := range s.m {
		result.Add(k)
	}
	for k := range other.m {
		result.Add(k)
	}
	return result
}

// Intersection returns elements present in both sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := NewSet[T]()
	for k := range s.m {
		if other.Contains(k) {
			result.Add(k)
		}
	}
	return result
}

// Difference returns elements in s but NOT in other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := NewSet[T]()
	for k := range s.m {
		if !other.Contains(k) {
			result.Add(k)
		}
	}
	return result
}

// =============================================================================
// PART 5: Option[T] — Modeling Optional Values
// =============================================================================
//
// Go uses multiple returns (value, bool) or (value, error) to signal
// absence of a value. This is idiomatic Go and works well.
//
// However, in APIs where you return a value that MIGHT be absent,
// an Option[T] type makes the optionality explicit in the return type.
// The caller cannot forget to check — the type system enforces it.
//
// Comparison with other languages:
//   Haskell: Maybe a  (Just a | Nothing)
//   Rust:    Option<T> (Some(T) | None)
//   Java:    Optional<T>
//   Go:      Option[T] (when you want this pattern)
//
// In Go, *T (nil-able pointer) is sometimes used for optionality,
// but pointers have overhead and semantic implications.
// Option[T] makes the intent explicit.

type Option[T any] struct {
	value   T
	present bool
}

// Some wraps a value in an Option that IS present.
func Some[T any](v T) Option[T] {
	return Option[T]{value: v, present: true}
}

// None returns an Option that is NOT present (the zero/absent value).
func None[T any]() Option[T] {
	return Option[T]{} // present = false, value = zero value of T
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool { return o.present }

// IsNone returns true if the Option is absent.
func (o Option[T]) IsNone() bool { return !o.present }

// Unwrap returns the value, panicking if absent.
// Use when you've already checked IsSome() or when absence is a programming error.
func (o Option[T]) Unwrap() T {
	if !o.present {
		panic("option: Unwrap called on None")
	}
	return o.value
}

// UnwrapOr returns the value if present, otherwise returns the default.
func (o Option[T]) UnwrapOr(defaultVal T) T {
	if o.present {
		return o.value
	}
	return defaultVal
}

// Get returns (value, bool) — the idiomatic Go style as a fallback.
func (o Option[T]) Get() (T, bool) {
	return o.value, o.present
}

func (o Option[T]) String() string {
	if o.present {
		return fmt.Sprintf("Some(%v)", o.value)
	}
	return "None"
}

// Real-world usage: finding an element in a map, returning Option.
func FindInMap[K comparable, V any](m map[K]V, key K) Option[V] {
	if v, ok := m[key]; ok {
		return Some(v)
	}
	return None[V]()
}

// =============================================================================
// PART 6: GENERIC TYPE INSTANTIATION
// =============================================================================
//
// When you write Stack[int]{} or Set[string]{}, you INSTANTIATE the generic type.
// The type parameter T is "frozen" to a specific type at that point.
//
// You can also create TYPE ALIASES for common instantiations:
//
//   type IntStack = Stack[int]
//   type StringSet = Set[string]
//
// This is useful when a particular instantiation is used frequently —
// avoids repeating the type argument everywhere.

type IntStack = Stack[int]     // type alias
type StringSet = Set[string]   // type alias
type StringIntPair = Pair[string, int]

// =============================================================================
// PART 7: WHEN GENERIC TYPES > interface{} DATA STRUCTURES
// =============================================================================
//
// Pre-generics, many Go libraries used interface{} for generic containers:
//
//   type Stack struct { items []interface{} }
//   func (s *Stack) Push(v interface{}) { s.items = append(s.items, v) }
//   func (s *Stack) Pop() interface{} { ... }
//
// Problems:
// 1. TYPE SAFETY LOST: You can push any type to any stack. No compile error.
//    A Stack meant for ints can accidentally receive a string.
// 2. TYPE ASSERTION ON POP: Caller must do pop().(int) — can panic.
// 3. BOXING OVERHEAD: Every value stored as interface = heap allocation per item.
//
// Generic types solve all three:
// 1. Stack[int] only accepts int — compile-time verified.
// 2. Pop() returns int directly — no assertion.
// 3. Values stored as T (concrete type) — no boxing, no heap allocation per item.

// =============================================================================
// PART 8: WHEN TO USE GENERIC TYPES vs GENERIC FUNCTIONS
// =============================================================================
//
// GENERIC TYPE when:
//   - You need a DATA STRUCTURE that STORES T over time
//   - Multiple operations on the same stored data (Push, Pop, Peek for Stack)
//   - State persists between method calls
//   - The structure IS the abstraction (Stack[T] IS a stack)
//
// GENERIC FUNCTION when:
//   - You process data and return a result (no persistent state)
//   - The function is the abstraction (Filter, Map, Reduce)
//   - Single operation on data
//
// The test: "Does T need to be stored in a struct field?"
//   YES → Generic type
//   NO  → Generic function

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("GENERIC TYPES — Data Structures with Type Parameters")
	fmt.Println(strings.Repeat("=", 60))

	// --- Stack[T] ---
	fmt.Println("\n--- Stack[T] ---")
	var istack IntStack // using type alias
	istack.Push(10)
	istack.Push(20)
	istack.Push(30)
	fmt.Printf("Stack size: %d\n", istack.Len())
	if top, ok := istack.Peek(); ok {
		fmt.Printf("Peek: %d\n", top)
	}
	for !istack.IsEmpty() {
		v, _ := istack.Pop()
		fmt.Printf("  Popped: %d\n", v)
	}

	// Stack of strings
	sstack := Stack[string]{}
	for _, word := range []string{"go", "is", "awesome"} {
		sstack.Push(word)
	}
	var words []string
	for !sstack.IsEmpty() {
		w, _ := sstack.Pop()
		words = append(words, w)
	}
	fmt.Printf("String stack (reversed): %v\n", words)

	// --- Queue[T] ---
	fmt.Println("\n--- Queue[T] ---")
	q := Queue[int]{}
	for _, n := range []int{1, 2, 3, 4, 5} {
		q.Enqueue(n)
	}
	fmt.Printf("Queue size: %d\n", q.Len())
	for !q.IsEmpty() {
		v, _ := q.Dequeue()
		fmt.Printf("  Dequeued: %d\n", v)
	}

	// --- Pair[K, V] ---
	fmt.Println("\n--- Pair[K, V] ---")
	p := NewPair("age", 30)
	fmt.Println("Pair:", p)
	swapped := SwapPair(p)
	fmt.Println("Swapped:", swapped)

	names := []string{"Alice", "Bob", "Carol"}
	scores := []int{95, 87, 92}
	pairs := ZipToPairs(names, scores)
	for _, pair := range pairs {
		fmt.Printf("  %s scored %d\n", pair.Key, pair.Value)
	}

	// Type alias usage
	var sip StringIntPair = NewPair("count", 42)
	fmt.Println("StringIntPair:", sip)

	// --- Set[T comparable] ---
	fmt.Println("\n--- Set[T comparable] ---")
	a := NewSet(1, 2, 3, 4, 5)
	b := NewSet(3, 4, 5, 6, 7)

	fmt.Printf("Set A: %v\n", a.ToSlice())
	fmt.Printf("Set B: %v\n", b.ToSlice())
	fmt.Printf("A.Contains(3): %v\n", a.Contains(3))
	fmt.Printf("A.Contains(9): %v\n", a.Contains(9))

	union := a.Union(b)
	inter := a.Intersection(b)
	diff := a.Difference(b)
	// Sets don't preserve order — sort for deterministic output
	fmt.Printf("Union size: %d\n", union.Len())
	fmt.Printf("Intersection size: %d\n", inter.Len())
	fmt.Printf("Difference(A-B) size: %d\n", diff.Len())

	// String set
	var strSet StringSet = NewSet("apple", "banana", "cherry")
	strSet.Add("date")
	strSet.Remove("banana")
	fmt.Printf("StringSet: contains 'apple'=%v, 'banana'=%v\n",
		strSet.Contains("apple"), strSet.Contains("banana"))

	// --- Option[T] ---
	fmt.Println("\n--- Option[T] ---")
	db := map[string]int{
		"alice": 30,
		"bob":   25,
	}

	for _, name := range []string{"alice", "charlie", "bob"} {
		opt := FindInMap(db, name)
		fmt.Printf("  FindInMap(%q) → %v\n", name, opt)
		if opt.IsSome() {
			fmt.Printf("    Age: %d\n", opt.Unwrap())
		} else {
			fmt.Printf("    Not found, default: %d\n", opt.UnwrapOr(0))
		}
	}

	// Option usage patterns
	present := Some(42)
	absent := None[int]()
	fmt.Printf("Some(42).UnwrapOr(0) = %d\n", present.UnwrapOr(0))
	fmt.Printf("None.UnwrapOr(99)    = %d\n", absent.UnwrapOr(99))
	v, ok := absent.Get()
	fmt.Printf("None.Get() = (%d, %v)\n", v, ok)

	// --- Summary ---
	fmt.Println("\n--- Generic Type Summary ---")
	fmt.Println("Stack[T]:    LIFO, O(1) Push/Pop, type-safe")
	fmt.Println("Queue[T]:    FIFO, O(1) Enqueue, O(n) Dequeue")
	fmt.Println("Pair[K,V]:   Two typed values as a unit")
	fmt.Println("Set[T]:      Unique elements, requires comparable")
	fmt.Println("Option[T]:   Explicit optional value (Some/None)")
	fmt.Println()
	fmt.Println("Alias examples:")
	fmt.Println("  type IntStack = Stack[int]")
	fmt.Println("  type StringSet = Set[string]")
	fmt.Println()
	fmt.Println("Rule: If T is stored in a struct field → Generic type")
	fmt.Println("      If T is processed and returned  → Generic function")
}
