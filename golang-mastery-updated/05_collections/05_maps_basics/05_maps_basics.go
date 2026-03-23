// FILE: 05_collections/05_maps_basics.go
// TOPIC: Maps — declaration, operations, nil vs empty, iteration, comma-ok
//
// Run: go run 05_collections/05_maps_basics.go

package main

import "fmt"

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Maps Basics")
	fmt.Println("════════════════════════════════════════")

	// ── WHAT IS A MAP? ───────────────────────────────────────────────────
	// A map is a hash table — unordered key→value store.
	// Keys must be COMPARABLE (==): string, int, bool, pointer, struct with comparable fields.
	// NOT valid as keys: slice, map, function.
	// Maps are REFERENCE TYPES: assigning a map to a new variable doesn't copy the data.

	// ── DECLARATION & INITIALIZATION ─────────────────────────────────────

	// nil map (zero value) — READING returns zero value, WRITING PANICS
	var nilMap map[string]int
	fmt.Printf("\n── nil vs empty map ──\n")
	fmt.Printf("  nilMap == nil: %v\n", nilMap == nil)
	fmt.Printf("  nilMap[\"x\"]:   %d  (reading nil map is safe, returns zero)\n", nilMap["x"])
	// nilMap["x"] = 1  ← PANIC: assignment to entry in nil map

	// Empty map via make — safe to read AND write
	emptyMap := make(map[string]int)
	fmt.Printf("  emptyMap == nil: %v\n", emptyMap == nil)
	emptyMap["x"] = 1  // safe

	// Map literal
	scores := map[string]int{
		"Alice": 95,
		"Bob":   82,
		"Carol": 91,
	}
	fmt.Printf("\n── Map literal ──\n")
	fmt.Printf("  scores: %v\n", scores)

	// make with capacity hint (optimization — not a hard limit)
	bigMap := make(map[string]int, 1000)
	_ = bigMap

	// ── CRUD OPERATIONS ──────────────────────────────────────────────────

	fmt.Println("\n── CRUD operations ──")

	// CREATE / UPDATE — same syntax
	scores["Dave"] = 88
	scores["Alice"] = 98  // update existing
	fmt.Printf("  After insert Dave and update Alice: %v\n", scores)

	// READ — always returns a value (zero if key absent)
	fmt.Printf("  scores[\"Alice\"] = %d\n", scores["Alice"])
	fmt.Printf("  scores[\"Zara\"]  = %d  (not present → zero value)\n", scores["Zara"])

	// DELETE
	delete(scores, "Dave")
	fmt.Printf("  After delete Dave: %v\n", scores)
	delete(scores, "Nobody")  // deleting non-existent key is a no-op (no panic)

	// ── COMMA-OK: check if key exists ────────────────────────────────────
	// ALWAYS use comma-ok to distinguish "key absent" from "value is zero"
	fmt.Println("\n── Comma-ok idiom ──")
	if v, ok := scores["Alice"]; ok {
		fmt.Printf("  Alice found: %d\n", v)
	}
	if _, ok := scores["Zara"]; !ok {
		fmt.Println("  Zara not found")
	}

	// ── len() on a map ────────────────────────────────────────────────────
	fmt.Printf("\n  len(scores) = %d\n", len(scores))

	// ── ITERATION — ORDER IS RANDOM ───────────────────────────────────────
	// Go deliberately randomizes map iteration order (security: hash flooding prevention).
	// Never rely on map iteration order. Sort keys if you need deterministic output.
	fmt.Println("\n── Map iteration (random order) ──")
	for k, v := range scores {
		fmt.Printf("  %s: %d\n", k, v)
	}

	// ── MAP AS REFERENCE TYPE ─────────────────────────────────────────────
	fmt.Println("\n── Maps are reference types ──")
	a := map[string]int{"x": 1}
	b := a           // b points to the SAME underlying map
	b["x"] = 99
	fmt.Printf("  After b[\"x\"]=99: a[\"x\"]=%d  (shared!)\n", a["x"])

	// To copy a map, you must iterate:
	c := make(map[string]int, len(a))
	for k, v := range a {
		c[k] = v
	}
	c["x"] = 1  // doesn't affect a
	fmt.Printf("  After copy+modify: a[\"x\"]=%d, c[\"x\"]=%d\n", a["x"], c["x"])

	// ── COUNTING WITH MAPS ────────────────────────────────────────────────
	fmt.Println("\n── Counting pattern ──")
	words := []string{"go", "is", "fast", "go", "is", "simple", "go"}
	freq := make(map[string]int)
	for _, w := range words {
		freq[w]++  // zero value (0) + 1 on first occurrence — elegant!
	}
	fmt.Printf("  word frequencies: %v\n", freq)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  nil map: reading OK (returns zero), writing PANICS")
	fmt.Println("  make(map[K]V) or literal to initialize")
	fmt.Println("  v, ok := m[k]  — always use comma-ok for existence check")
	fmt.Println("  delete(m, k)   — no-op if key absent")
	fmt.Println("  Iteration order is RANDOM — don't rely on it")
	fmt.Println("  Maps are reference types — assignment shares the data")
}
