// FILE: 05_collections/06_maps_advanced.go
// TOPIC: Maps Advanced — sets, grouping, inversion, concurrent access
//
// Run: go run 05_collections/06_maps_advanced.go

package main

import (
	"fmt"
	"sort"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Maps Advanced")
	fmt.Println("════════════════════════════════════════")

	// ── MAP AS A SET ─────────────────────────────────────────────────────
	// Go has no built-in set type. Use map[T]struct{} as a set.
	// struct{} uses ZERO bytes — it's the empty struct, a memory-free value.
	// This is more efficient than map[T]bool (bool uses 1 byte per entry).

	fmt.Println("\n── Set using map[T]struct{} ──")
	set := make(map[string]struct{})
	for _, v := range []string{"go", "rust", "python", "go", "rust"} {
		set[v] = struct{}{}  // add to set (duplicates ignored)
	}
	fmt.Printf("  set: ")
	// Print sorted for readability:
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println(keys)

	// Check membership:
	_, inSet := set["go"]
	fmt.Printf("  \"go\" in set: %v\n", inSet)
	_, inSet = set["java"]
	fmt.Printf("  \"java\" in set: %v\n", inSet)

	// ── GROUPING (map[K][]V) ──────────────────────────────────────────────
	fmt.Println("\n── Grouping by key ──")
	type Person struct{ Name, City string }
	people := []Person{
		{"Alice", "NYC"}, {"Bob", "LA"}, {"Carol", "NYC"},
		{"Dave", "LA"}, {"Eve", "NYC"},
	}
	byCity := make(map[string][]string)
	for _, p := range people {
		byCity[p.City] = append(byCity[p.City], p.Name)
	}
	for _, city := range []string{"NYC", "LA"} {
		fmt.Printf("  %s: %v\n", city, byCity[city])
	}

	// ── INVERTING A MAP ───────────────────────────────────────────────────
	fmt.Println("\n── Inverting a map ──")
	codeToName := map[int]string{1: "Alice", 2: "Bob", 3: "Carol"}
	nameToCode := make(map[string]int, len(codeToName))
	for k, v := range codeToName {
		nameToCode[v] = k
	}
	fmt.Printf("  nameToCode: %v\n", nameToCode)

	// ── SORTED MAP ITERATION ──────────────────────────────────────────────
	fmt.Println("\n── Sorted iteration ──")
	scores := map[string]int{"Charlie": 70, "Alice": 95, "Bob": 82}
	sortedKeys := make([]string, 0, len(scores))
	for k := range scores {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		fmt.Printf("  %s: %d\n", k, scores[k])
	}

	// ── DELETING DURING RANGE — SAFE IN GO ────────────────────────────────
	fmt.Println("\n── Delete during range (safe) ──")
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	for k, v := range m {
		if v%2 == 0 {
			delete(m, k)  // safe: Go allows delete during range
		}
	}
	fmt.Printf("  After deleting evens: %v\n", m)

	// ── MAPS ARE NOT SAFE FOR CONCURRENT USE ─────────────────────────────
	// Concurrent reads are fine, but concurrent read+write causes a race.
	// Solutions:
	//   1. sync.Mutex around map access
	//   2. sync.RWMutex (multiple readers, exclusive writers)
	//   3. sync.Map (built-in concurrent map, for specific use cases)
	// See Module 06 for details.
	fmt.Println("\n── Concurrent map safety ──")
	fmt.Println("  Maps are NOT safe for concurrent access!")
	fmt.Println("  Use sync.Mutex, sync.RWMutex, or sync.Map (Module 06)")

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Set: map[T]struct{}  — zero memory overhead for values")
	fmt.Println("  Grouping: map[K][]V  — append to slice per key")
	fmt.Println("  Sort keys for deterministic output")
	fmt.Println("  delete() during range is safe")
	fmt.Println("  Concurrent access needs sync.Mutex or sync.Map")
}
