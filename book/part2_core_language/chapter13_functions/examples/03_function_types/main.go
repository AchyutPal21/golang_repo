// FILE: book/part2_core_language/chapter13_functions/examples/03_function_types/main.go
// CHAPTER: 13 — Functions: First-Class Citizens
// TOPIC: Functions as values, function types, higher-order functions,
//        functions returned from functions (factory pattern).
//
// Run (from the chapter folder):
//   go run ./examples/03_function_types

package main

import (
	"fmt"
	"strings"
)

// Transformer is a named function type. Naming it makes signatures cleaner
// and allows methods to be defined on it (shown in Chapter 21).
type Transformer func(string) string

// Predicate is a function type that returns bool — common in filtering.
type Predicate[T any] func(T) bool

// apply runs a Transformer on each string in a slice.
// Accepting a function type as a parameter is the basis of all
// higher-order programming in Go.
func apply(ss []string, t Transformer) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = t(s)
	}
	return out
}

// filter returns elements for which pred returns true.
func filter[T any](items []T, pred Predicate[T]) []T {
	var out []T
	for _, v := range items {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// chain composes two Transformers left-to-right: chain(f,g)(s) == g(f(s)).
func chain(f, g Transformer) Transformer {
	return func(s string) string {
		return g(f(s))
	}
}

// repeat returns a Transformer that repeats s n times with sep between.
// This is a function factory: each call returns a new function value.
func repeat(n int, sep string) Transformer {
	return func(s string) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = s
		}
		return strings.Join(parts, sep)
	}
}

// adder returns a closure that adds delta to its argument.
// The returned function carries delta in its closure environment.
func adder(delta int) func(int) int {
	return func(n int) int {
		return n + delta
	}
}

// memoize wraps a func(int)int and caches results.
// The cache is part of the returned function's closure.
func memoize(f func(int) int) func(int) int {
	cache := map[int]int{}
	return func(n int) int {
		if v, ok := cache[n]; ok {
			fmt.Printf("  [cache hit  n=%d]\n", n)
			return v
		}
		fmt.Printf("  [cache miss n=%d]\n", n)
		v := f(n)
		cache[n] = v
		return v
	}
}

// --- Functions stored in maps (dispatch table) ---

type handler func(string) string

var ops = map[string]handler{
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"title": strings.Title, //nolint:staticcheck // SA1019: deprecated but still works
	"trim":  strings.TrimSpace,
}

func dispatch(op, input string) (string, bool) {
	h, ok := ops[op]
	if !ok {
		return "", false
	}
	return h(input), true
}

func main() {
	words := []string{"Go", "is", "fast", "and", "safe"}

	// apply with stdlib function as value
	fmt.Println("upper:", apply(words, strings.ToUpper))

	// apply with a lambda
	fmt.Println("exclaim:", apply(words, func(s string) string {
		return s + "!"
	}))

	fmt.Println()

	// chain: lower then exclaim
	exclaim := func(s string) string { return s + "!" }
	lowerExclaim := chain(strings.ToLower, exclaim)
	fmt.Println("lowerExclaim:", apply(words, lowerExclaim))

	fmt.Println()

	// function factory: repeat
	triple := repeat(3, "-")
	fmt.Println("triple(Go):", triple("Go"))
	fmt.Println("triple(hi):", triple("hi"))

	fmt.Println()

	// adder factory
	add10 := adder(10)
	add100 := adder(100)
	fmt.Println("add10(5)   =", add10(5))
	fmt.Println("add100(5)  =", add100(5))

	fmt.Println()

	// filter generic predicate
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	evens := filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("evens:", evens)

	longWords := filter(words, func(s string) bool { return len(s) > 2 })
	fmt.Println("longWords:", longWords)

	fmt.Println()

	// memoized square
	fmt.Println("memoized square:")
	square := memoize(func(n int) int { return n * n })
	fmt.Println(" square(4)  =", square(4))
	fmt.Println(" square(4)  =", square(4)) // cache hit
	fmt.Println(" square(9)  =", square(9))

	fmt.Println()

	// dispatch table
	for _, op := range []string{"upper", "lower", "trim", "unknown"} {
		if result, ok := dispatch(op, "  Hello World  "); ok {
			fmt.Printf("dispatch(%q) = %q\n", op, result)
		} else {
			fmt.Printf("dispatch(%q) = unknown op\n", op)
		}
	}
}
