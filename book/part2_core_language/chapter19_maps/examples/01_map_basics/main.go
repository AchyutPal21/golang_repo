// FILE: book/part2_core_language/chapter19_maps/examples/01_map_basics/main.go
// CHAPTER: 19 — Maps: Hash Tables Built In
// TOPIC: Declaration, make, composite literals, CRUD, nil map trap,
//        comma-ok idiom, delete, iteration randomisation.
//
// Run (from the chapter folder):
//   go run ./examples/01_map_basics

package main

import (
	"fmt"
	"sort"
)

func main() {
	// --- nil map ---
	var m map[string]int
	fmt.Println("nil map:", m)
	fmt.Println("nil map == nil:", m == nil)
	fmt.Println("read from nil map:", m["key"]) // zero value, no panic

	// Writing to a nil map panics:
	// m["key"] = 1 // panic: assignment to entry in nil map

	fmt.Println()

	// --- make ---
	scores := make(map[string]int)
	scores["Alice"] = 95
	scores["Bob"] = 87
	scores["Carol"] = 92
	fmt.Println("scores:", scores) // order is randomised

	// --- composite literal ---
	config := map[string]string{
		"host":    "localhost",
		"port":    "8080",
		"timeout": "30s",
	}
	fmt.Println("config host:", config["host"])

	fmt.Println()

	// --- comma-ok idiom ---
	if v, ok := scores["Alice"]; ok {
		fmt.Println("Alice:", v)
	}
	if _, ok := scores["Dave"]; !ok {
		fmt.Println("Dave not found")
	}

	// Zero value is returned for missing keys — can be misleading:
	fmt.Println("missing key returns zero:", scores["Dave"]) // 0

	fmt.Println()

	// --- delete ---
	delete(scores, "Bob")
	fmt.Println("after delete Bob:", scores)
	delete(scores, "NonExistent") // safe — no-op

	fmt.Println()

	// --- len ---
	fmt.Println("len:", len(scores))

	fmt.Println()

	// --- iteration order is random ---
	// To iterate in sorted order, collect keys first.
	for i := range 3 {
		fmt.Printf("random iteration %d: ", i+1)
		for k := range scores {
			fmt.Printf("%s ", k)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("sorted iteration:")
	keys := make([]string, 0, len(scores))
	for k := range scores {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, scores[k])
	}

	fmt.Println()

	// --- maps are reference types ---
	original := map[string]int{"a": 1}
	ref := original   // ref and original share the same underlying map
	ref["b"] = 2
	fmt.Println("original after ref mutation:", original) // includes "b"

	// To copy: iterate and populate a new map.
	clone := make(map[string]int, len(original))
	for k, v := range original {
		clone[k] = v
	}
	clone["c"] = 3
	fmt.Println("original after clone mutation:", original) // unchanged
}
