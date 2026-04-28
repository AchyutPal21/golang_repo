// FILE: book/part2_core_language/chapter13_functions/examples/02_variadic/main.go
// CHAPTER: 13 — Functions: First-Class Citizens
// TOPIC: Variadic functions, spreading a slice, init functions.
//
// Run (from the chapter folder):
//   go run ./examples/02_variadic

package main

import "fmt"

// sum is the textbook variadic: nums is []int inside the function body.
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// max extends variadic with a required first argument — the signature
// forces at least one value, making max(0 values) a compile error.
func max(first int, rest ...int) int {
	m := first
	for _, n := range rest {
		if n > m {
			m = n
		}
	}
	return m
}

// joinWith demonstrates variadic mixed with non-variadic parameters.
func joinWith(sep string, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}

// appendUnique appends elements to a slice, skipping duplicates.
// Passing the existing slice with ... spreads it into the variadic.
func appendUnique(existing []int, candidates ...int) []int {
	seen := make(map[int]bool, len(existing))
	for _, v := range existing {
		seen[v] = true
	}
	for _, c := range candidates {
		if !seen[c] {
			existing = append(existing, c)
			seen[c] = true
		}
	}
	return existing
}

// --- init function ---
//
// init runs automatically before main, after all variable initialisations.
// A package can have multiple init functions (across one or many files);
// they run in source order within a file, file order within a package.
// You cannot call init() manually — the compiler forbids it.

var greeting string

func init() {
	greeting = "Hello from init"
	fmt.Println("[init] package initialised")
}

func main() {
	fmt.Println(greeting)
	fmt.Println()

	// --- sum ---
	fmt.Println("sum()       =", sum())
	fmt.Println("sum(1,2,3)  =", sum(1, 2, 3))

	// Spread a slice with the ... operator.
	nums := []int{10, 20, 30, 40}
	fmt.Println("sum(spread) =", sum(nums...))

	fmt.Println()

	// --- max ---
	fmt.Println("max(5)         =", max(5))
	fmt.Println("max(3,1,4,1,5) =", max(3, 1, 4, 1, 5))

	fmt.Println()

	// --- joinWith ---
	fmt.Println(joinWith(", ", "Alice", "Bob", "Carol"))
	fmt.Println(joinWith("-", "2026", "04", "27"))
	fmt.Println(joinWith("/", "home"))

	fmt.Println()

	// --- appendUnique (spread into variadic) ---
	base := []int{1, 2, 3}
	extra := []int{3, 4, 5, 1}
	result := appendUnique(base, extra...)
	fmt.Println("appendUnique:", result)
}
