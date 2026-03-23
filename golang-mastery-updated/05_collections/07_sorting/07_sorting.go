// FILE: 05_collections/07_sorting.go
// TOPIC: Sorting — sort package, custom comparators, sort.Interface
//
// Run: go run 05_collections/07_sorting.go

package main

import (
	"fmt"
	"sort"
)

type Person struct {
	Name string
	Age  int
}

// Implementing sort.Interface for a custom type:
// sort.Interface requires 3 methods: Len, Less, Swap
type ByAge []Person

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Less(i, j int) bool { return a[i].Age < a[j].Age }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Sorting")
	fmt.Println("════════════════════════════════════════")

	// ── BUILT-IN SORT FUNCTIONS ───────────────────────────────────────────
	fmt.Println("\n── Built-in sort functions ──")
	ints := []int{5, 2, 8, 1, 9, 3}
	sort.Ints(ints)
	fmt.Printf("  sort.Ints:    %v\n", ints)

	strs := []string{"banana", "apple", "cherry", "date"}
	sort.Strings(strs)
	fmt.Printf("  sort.Strings: %v\n", strs)

	floats := []float64{3.14, 1.41, 2.72, 1.73}
	sort.Float64s(floats)
	fmt.Printf("  sort.Float64s: %v\n", floats)

	// ── sort.Slice — custom comparator (most common) ───────────────────────
	// sort.Slice is the go-to for custom sorting without implementing sort.Interface.
	// The less function returns true if element i should come BEFORE element j.
	fmt.Println("\n── sort.Slice ──")
	people := []Person{
		{"Alice", 30}, {"Bob", 25}, {"Carol", 35}, {"Dave", 25},
	}
	// Sort by age ascending:
	sort.Slice(people, func(i, j int) bool {
		return people[i].Age < people[j].Age
	})
	fmt.Printf("  By age:       %v\n", people)

	// sort.SliceStable — preserves order of equal elements:
	sort.SliceStable(people, func(i, j int) bool {
		return people[i].Age < people[j].Age
	})
	fmt.Printf("  By age stable:%v\n", people)

	// Multi-key sort: first by age, then by name for ties:
	sort.SliceStable(people, func(i, j int) bool {
		if people[i].Age != people[j].Age {
			return people[i].Age < people[j].Age
		}
		return people[i].Name < people[j].Name
	})
	fmt.Printf("  By age+name:  %v\n", people)

	// ── sort.Interface — for reusable sorters ─────────────────────────────
	fmt.Println("\n── sort.Interface ──")
	people2 := []Person{{"Charlie", 40}, {"Alice", 30}, {"Bob", 25}}
	sort.Sort(ByAge(people2))
	fmt.Printf("  sort.Sort(ByAge): %v\n", people2)

	// Reverse any sort with sort.Reverse:
	sort.Sort(sort.Reverse(ByAge(people2)))
	fmt.Printf("  sort.Reverse:     %v\n", people2)

	// ── CHECKING IF SORTED ─────────────────────────────────────────────────
	fmt.Println("\n── IsSorted / Search ──")
	sorted := []int{1, 2, 3, 5, 8, 13}
	fmt.Printf("  IsSorted: %v\n", sort.IntsAreSorted(sorted))

	// sort.Search — binary search (O(log n)) — requires sorted input
	// Returns the smallest index i where f(i) is true.
	target := 5
	idx := sort.SearchInts(sorted, target)
	fmt.Printf("  SearchInts(%v, %d) → index %d\n", sorted, target, idx)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  sort.Ints / Strings / Float64s  — built-in types")
	fmt.Println("  sort.Slice(s, less)              — one-off custom sort")
	fmt.Println("  sort.SliceStable                 — preserves equal order")
	fmt.Println("  sort.Interface (Len/Less/Swap)   — reusable, reversible")
	fmt.Println("  sort.Reverse(x)                  — reverse any sort")
	fmt.Println("  sort.SearchInts / SearchStrings  — binary search")
}
