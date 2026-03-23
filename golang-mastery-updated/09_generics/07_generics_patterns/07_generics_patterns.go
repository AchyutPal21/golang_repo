// FILE: 09_generics/07_generics_patterns.go
// TOPIC: Generics Patterns — Result[T], Option[T], generic cache, when NOT to use
//
// Run: go run 09_generics/07_generics_patterns.go

package main

import (
	"fmt"
	"sync"
)

// ── RESULT[T] — typed error result ───────────────────────────────────────────
// Like Rust's Result<T, E> or Haskell's Either.
// Avoids returning (T, error) tuples when building functional pipelines.

type Result[T any] struct {
	value T
	err   error
}

func OK[T any](v T) Result[T]       { return Result[T]{value: v} }
func Err[T any](e error) Result[T]  { return Result[T]{err: e} }

func (r Result[T]) IsOK() bool      { return r.err == nil }
func (r Result[T]) Value() T        { return r.value }
func (r Result[T]) Error() error    { return r.err }
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Map applies a transform only if result is OK:
func ResultMap[T, U any](r Result[T], f func(T) U) Result[U] {
	if !r.IsOK() {
		return Err[U](r.err)
	}
	return OK(f(r.value))
}

// ── OPTION[T] — explicitly optional values ────────────────────────────────────
// Replaces the nil pointer pattern with a typed optional.

type Option[T any] struct {
	value    T
	hasValue bool
}

func Some[T any](v T) Option[T] { return Option[T]{value: v, hasValue: true} }
func None[T any]() Option[T]    { return Option[T]{} }

func (o Option[T]) IsSome() bool       { return o.hasValue }
func (o Option[T]) IsNone() bool       { return !o.hasValue }
func (o Option[T]) ValueOr(def T) T {
	if o.hasValue { return o.value }
	return def
}
func (o Option[T]) Unwrap() T {
	if !o.hasValue { panic("unwrap on None") }
	return o.value
}

// ── GENERIC CACHE ─────────────────────────────────────────────────────────────
type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{items: make(map[K]V)}
}

func (c *Cache[K, V]) Set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = v
}

func (c *Cache[K, V]) Get(k K) Option[V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.items[k]; ok {
		return Some(v)
	}
	return None[V]()
}

func (c *Cache[K, V]) GetOrSet(k K, compute func() V) V {
	if opt := c.Get(k); opt.IsSome() {
		return opt.Unwrap()
	}
	v := compute()
	c.Set(k, v)
	return v
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Generics Patterns")
	fmt.Println("════════════════════════════════════════")

	// ── Result[T] ─────────────────────────────────────────────────────────
	fmt.Println("\n── Result[T] ──")
	r1 := OK(42)
	r2 := Err[int](fmt.Errorf("something failed"))
	fmt.Printf("  OK(42): isOK=%v, value=%d\n", r1.IsOK(), r1.Value())
	fmt.Printf("  Err:    isOK=%v, err=%v\n", r2.IsOK(), r2.Error())

	// Chain with ResultMap:
	r3 := ResultMap(r1, func(n int) string { return fmt.Sprintf("result=%d", n) })
	fmt.Printf("  ResultMap(OK(42), toString): %q\n", r3.Value())

	r4 := ResultMap(r2, func(n int) string { return "never called" })
	fmt.Printf("  ResultMap(Err, toString): isOK=%v, err=%v\n", r4.IsOK(), r4.Error())

	// ── Option[T] ─────────────────────────────────────────────────────────
	fmt.Println("\n── Option[T] ──")
	some := Some("hello")
	none := None[string]()
	fmt.Printf("  Some(\"hello\"): isSome=%v, value=%q\n", some.IsSome(), some.Unwrap())
	fmt.Printf("  None: isNone=%v, ValueOr=\"default\"=%q\n", none.IsNone(), none.ValueOr("default"))

	// Database lookup returning Option instead of (*T, error):
	userDB := map[int]string{1: "Alice", 2: "Bob"}
	findUser := func(id int) Option[string] {
		if u, ok := userDB[id]; ok {
			return Some(u)
		}
		return None[string]()
	}
	fmt.Printf("  findUser(1): %q\n", findUser(1).ValueOr("unknown"))
	fmt.Printf("  findUser(99): %q\n", findUser(99).ValueOr("unknown"))

	// ── Generic Cache ─────────────────────────────────────────────────────
	fmt.Println("\n── Generic Cache[K,V] ──")
	cache := NewCache[string, int]()
	cache.Set("count", 100)
	cache.Set("score", 95)

	fmt.Printf("  Get(\"count\"): %d\n", cache.Get("count").ValueOr(0))
	fmt.Printf("  Get(\"missing\"): %d\n", cache.Get("missing").ValueOr(-1))

	calls := 0
	for i := 0; i < 3; i++ {
		v := cache.GetOrSet("computed", func() int {
			calls++
			return 42
		})
		fmt.Printf("  GetOrSet attempt %d: %d\n", i+1, v)
	}
	fmt.Printf("  compute() called %d time(s) (memoized)\n", calls)

	// ── When NOT to use generics ──────────────────────────────────────────
	fmt.Println("\n── When NOT to use generics ──")
	fmt.Println(`
  1. Only 1-2 call sites → just write it out (two concrete functions)
     Bad: generic for only int and string → just write two functions

  2. Interface already covers it → use interface
     If you need runtime polymorphism (different behavior per type), use interface.
     If you need compile-time safety + no boxing, use generics.

  3. The function logic uses reflect anyway → no benefit

  4. Complex constraints hurt readability → simplify or use interface

  Rule of thumb:
    Data structures (Stack, Queue, Map, Set) → generics ✓
    Algorithms on homogenous data (Map, Filter, Sort) → generics ✓
    Behavior that varies by type (method dispatch) → interface ✓
    One-off functions → just use concrete types ✓
`)

	fmt.Println("─── SUMMARY ────────────────────────────────")
	fmt.Println("  Result[T]: typed success/failure, chainable with ResultMap")
	fmt.Println("  Option[T]: explicit optional (vs nil pointer)")
	fmt.Println("  Cache[K,V]: type-safe concurrent cache")
	fmt.Println("  Generics shine for: containers, algorithms, utilities")
	fmt.Println("  Use interface when behavior differs per type at runtime")
}
