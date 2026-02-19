package main

// =============================================================================
// MODULE 05: COLLECTIONS — Arrays, Slices, Maps — Deep Dive
// =============================================================================
// Run: go run 05_collections/main.go
// =============================================================================

import (
	"fmt"
	"sort"
	"strings"
)

func main() {
	fmt.Println("=== MODULE 05: COLLECTIONS ===")

	// =========================================================================
	// SECTION 1: ARRAYS — Fixed-size, value type
	// =========================================================================
	// Arrays in Go are VALUE types — assigning copies the entire array.
	// The length is PART OF THE TYPE: [3]int ≠ [4]int
	// Rarely used directly — slices are preferred.

	fmt.Println("\n--- Arrays ---")

	// Declaration — zero values
	var arr1 [5]int
	fmt.Println("zero array:", arr1) // [0 0 0 0 0]

	// Initialization
	arr2 := [5]int{10, 20, 30, 40, 50}
	fmt.Println("arr2:", arr2)

	// ... lets Go count the elements
	arr3 := [...]string{"apple", "banana", "cherry"}
	fmt.Println("arr3:", arr3, "len:", len(arr3))

	// Specific index initialization
	arr4 := [5]int{0: 100, 2: 200, 4: 300}
	fmt.Println("arr4:", arr4) // [100 0 200 0 300]

	// Arrays are VALUE types — copying
	arr5 := arr2          // full copy
	arr5[0] = 999
	fmt.Println("arr2:", arr2) // unchanged
	fmt.Println("arr5:", arr5) // modified copy

	// 2D array
	matrix := [3][3]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	fmt.Println("matrix:")
	for _, row := range matrix {
		fmt.Println(" ", row)
	}

	// =========================================================================
	// SECTION 2: SLICES — Dynamic, reference type — the workhorse of Go
	// =========================================================================
	// A slice has THREE components internally:
	//   - Pointer: to the underlying array
	//   - Length: number of elements currently visible
	//   - Capacity: total size of the underlying array from this pointer
	//
	// Slices are REFERENCE types — they share the underlying array.

	fmt.Println("\n--- Slices ---")

	// Creating slices
	// 1. Slice literal — creates array and slice simultaneously
	s1 := []int{1, 2, 3, 4, 5}
	fmt.Println("s1:", s1, "len:", len(s1), "cap:", cap(s1))

	// 2. make([]T, length, capacity)
	s2 := make([]int, 5)       // len=5, cap=5, all zeros
	s3 := make([]int, 3, 10)   // len=3, cap=10
	fmt.Println("s2:", s2, "len:", len(s2), "cap:", cap(s2))
	fmt.Println("s3:", s3, "len:", len(s3), "cap:", cap(s3))

	// 3. Slicing an array or another slice
	underlying := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
	s4 := underlying[2:5] // [2, 3, 4] — indices 2, 3, 4
	fmt.Printf("s4: %v len=%d cap=%d\n", s4, len(s4), cap(s4))
	// cap = from start of slice to end of underlying array = 8-2 = 6

	// Slice default bounds — can omit low or high
	s5 := underlying[:3]  // same as underlying[0:3]
	s6 := underlying[5:]  // same as underlying[5:8]
	s7 := underlying[:]   // same as underlying[0:8] — full slice
	fmt.Println("s5:", s5)
	fmt.Println("s6:", s6)
	fmt.Println("s7:", s7)

	// 4. nil slice — zero value of a slice
	var nilSlice []int
	fmt.Println("nil slice:", nilSlice, "len:", len(nilSlice), "nil?", nilSlice == nil)
	// nil slices are valid — len=0, cap=0, append works on them

	// --- SHARED UNDERLYING ARRAY ---
	// Modifying a slice element modifies the underlying array
	a := []int{1, 2, 3, 4, 5}
	b := a[1:4] // shares same underlying array
	b[0] = 99
	fmt.Println("a after b[0]=99:", a) // [1 99 3 4 5] — a is affected!

	// --- APPEND ---
	// append returns a new slice (may point to a new underlying array if capacity exceeded)
	fmt.Println("\n--- append ---")

	var s []int // nil slice
	s = append(s, 1)
	s = append(s, 2, 3, 4) // can append multiple values
	fmt.Printf("s: %v len=%d cap=%d\n", s, len(s), cap(s))

	// Append grows capacity automatically — usually doubles
	s8 := make([]int, 0, 2)
	for i := 1; i <= 6; i++ {
		s8 = append(s8, i)
		fmt.Printf("after append(%d): len=%d cap=%d\n", i, len(s8), cap(s8))
	}

	// Append a slice to a slice using ...
	s9 := []int{1, 2, 3}
	s10 := []int{4, 5, 6}
	s11 := append(s9, s10...) // spread s10
	fmt.Println("combined:", s11)

	// --- COPY ---
	// copy(dst, src) — copies elements from src to dst
	// Returns number of elements copied = min(len(dst), len(src))
	fmt.Println("\n--- copy ---")

	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3) // only 3 spaces
	n := copy(dst, src)
	fmt.Println("copied:", n, "dst:", dst) // copies 3 elements

	// copy for safe independent slice
	original := []int{10, 20, 30}
	clone := make([]int, len(original))
	copy(clone, original)
	clone[0] = 999
	fmt.Println("original:", original) // unchanged
	fmt.Println("clone:", clone)       // modified

	// --- DELETING from a slice ---
	fmt.Println("\n--- Slice operations ---")

	nums := []int{1, 2, 3, 4, 5, 6, 7}

	// Delete element at index 2 (value 3)
	i := 2
	nums = append(nums[:i], nums[i+1:]...)
	fmt.Println("after delete index 2:", nums) // [1 2 4 5 6 7]

	// Insert element at index 2
	nums = append(nums[:2], append([]int{99}, nums[2:]...)...)
	fmt.Println("after insert 99 at index 2:", nums) // [1 2 99 4 5 6 7]

	// Reverse a slice
	rev := []int{1, 2, 3, 4, 5}
	for l, r := 0, len(rev)-1; l < r; l, r = l+1, r-1 {
		rev[l], rev[r] = rev[r], rev[l] // swap
	}
	fmt.Println("reversed:", rev)

	// Filter — create new slice with elements matching condition
	all := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	evens := all[:0] // len=0, shares underlying array
	for _, v := range all {
		if v%2 == 0 {
			evens = append(evens, v)
		}
	}
	// Or better — don't share underlying array:
	var evens2 []int
	for _, v := range all {
		if v%2 == 0 {
			evens2 = append(evens2, v)
		}
	}
	fmt.Println("evens:", evens2)

	// Map over slice
	doubled := make([]int, len(all))
	for i, v := range all {
		doubled[i] = v * 2
	}
	fmt.Println("doubled:", doubled)

	// Reduce
	sum := 0
	for _, v := range all {
		sum += v
	}
	fmt.Println("sum:", sum)

	// --- 2D SLICE (slice of slices) ---
	fmt.Println("\n--- 2D Slice ---")

	rows, cols := 3, 4
	grid := make([][]int, rows)
	for i := range grid {
		grid[i] = make([]int, cols)
		for j := range grid[i] {
			grid[i][j] = i*cols + j
		}
	}
	for _, row := range grid {
		fmt.Println(row)
	}

	// --- SORTING ---
	fmt.Println("\n--- Sorting ---")

	ints := []int{5, 2, 8, 1, 9, 3}
	sort.Ints(ints)
	fmt.Println("sorted ints:", ints)

	strs := []string{"banana", "apple", "cherry", "date"}
	sort.Strings(strs)
	fmt.Println("sorted strings:", strs)

	// sort.Slice — sort by custom comparator
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Charlie", 30},
		{"Alice", 25},
		{"Bob", 35},
	}
	// Sort by age ascending
	sort.Slice(people, func(i, j int) bool {
		return people[i].Age < people[j].Age
	})
	fmt.Println("sorted by age:", people)

	// Sort by name
	sort.Slice(people, func(i, j int) bool {
		return people[i].Name < people[j].Name
	})
	fmt.Println("sorted by name:", people)

	// Check if sorted
	fmt.Println("is sorted:", sort.IntsAreSorted(ints))

	// Binary search
	idx, found := sort.Find(len(ints), func(i int) int {
		return ints[i] - 5
	})
	fmt.Printf("binary search for 5: idx=%d found=%v\n", idx, found)

	// =========================================================================
	// SECTION 3: MAPS — Hash tables / dictionaries
	// =========================================================================
	// Maps are REFERENCE types.
	// Key must be a comparable type (==, !=) — no slices, maps, or funcs as keys.
	// Reading a missing key returns the zero value — NEVER panics.
	// Writing to a nil map PANICS.

	fmt.Println("\n--- Maps ---")

	// Creating maps
	// 1. make
	m1 := make(map[string]int)
	m1["apple"] = 5
	m1["banana"] = 3
	fmt.Println("m1:", m1)

	// 2. Map literal
	m2 := map[string]int{
		"apple":  5,
		"banana": 3,
		"cherry": 8,
	}
	fmt.Println("m2:", m2)

	// 3. nil map — reading is safe, writing panics!
	var nilMap map[string]int
	fmt.Println("nil map:", nilMap)             // map[]
	fmt.Println("nil map[x]:", nilMap["x"])     // 0 — safe read
	fmt.Println("nil map is nil:", nilMap == nil) // true
	// nilMap["x"] = 1 // PANIC: assignment to entry in nil map

	// --- CRUD operations ---
	fmt.Println("\n--- Map CRUD ---")

	inventory := make(map[string]int)

	// Create / Update
	inventory["apple"] = 10
	inventory["banana"] = 5
	inventory["cherry"] = 8
	inventory["apple"] = 15 // update (overwrite)

	// Read
	fmt.Println("apple:", inventory["apple"])

	// Check existence — two-value form
	val, exists := inventory["banana"]
	fmt.Printf("banana: %d, exists: %v\n", val, exists)

	val2, exists2 := inventory["grape"] // missing key
	fmt.Printf("grape: %d, exists: %v\n", val2, exists2) // 0, false

	// Delete
	delete(inventory, "cherry")
	fmt.Println("after delete cherry:", inventory)

	// Iterate over map — ORDER IS RANDOM (by design)
	fmt.Println("iterating:")
	for key, val := range inventory {
		fmt.Printf("  %s: %d\n", key, val)
	}

	// Iterate keys only
	for key := range inventory {
		fmt.Print(key, " ")
	}
	fmt.Println()

	// --- MAP LENGTH ---
	fmt.Println("map len:", len(inventory))

	// --- SORTED MAP KEYS (common pattern) ---
	fmt.Println("\n--- Sorted Map Iteration ---")

	m3 := map[string]int{"c": 3, "a": 1, "b": 2, "e": 5, "d": 4}
	keys := make([]string, 0, len(m3))
	for k := range m3 {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s: %d\n", k, m3[k])
	}

	// --- MAP AS SET ---
	fmt.Println("\n--- Map as Set ---")

	set := make(map[string]struct{}) // struct{} uses ZERO bytes
	words := []string{"go", "is", "awesome", "go", "is", "great"}
	for _, w := range words {
		set[w] = struct{}{}
	}
	fmt.Println("unique words:", set)

	// Check membership
	if _, ok := set["go"]; ok {
		fmt.Println("'go' is in the set")
	}

	// --- NESTED MAPS ---
	fmt.Println("\n--- Nested Maps ---")

	nested := map[string]map[string]int{
		"Alice": {"math": 95, "science": 88},
		"Bob":   {"math": 72, "science": 91},
	}

	for student, scores := range nested {
		fmt.Printf("%s: math=%d, science=%d\n", student, scores["math"], scores["science"])
	}

	// Safe nested map access
	if scores, ok := nested["Alice"]; ok {
		if mathScore, ok := scores["math"]; ok {
			fmt.Println("Alice's math:", mathScore)
		}
	}

	// --- MAP OF SLICES ---
	fmt.Println("\n--- Map of Slices ---")

	groups := make(map[string][]string)
	data := []struct{ name, dept string }{
		{"Alice", "Engineering"},
		{"Bob", "Marketing"},
		{"Charlie", "Engineering"},
		{"Dave", "Marketing"},
		{"Eve", "Engineering"},
	}
	for _, d := range data {
		groups[d.dept] = append(groups[d.dept], d.name)
	}
	for dept, members := range groups {
		fmt.Printf("%s: %s\n", dept, strings.Join(members, ", "))
	}

	// =========================================================================
	// SECTION 4: STRING OPERATIONS (bonus — strings act like byte slices)
	// =========================================================================
	fmt.Println("\n--- Strings ---")

	str := "Hello, 世界" // Unicode string
	fmt.Println("len (bytes):", len(str))

	// Iterate as bytes
	for i := 0; i < len(str); i++ {
		fmt.Printf("%d:%x ", i, str[i])
	}
	fmt.Println()

	// Iterate as runes (Unicode code points) — correct for Unicode
	for i, ch := range str {
		fmt.Printf("%d:%c ", i, ch)
	}
	fmt.Println()

	// Convert to rune slice for indexing by character
	runes := []rune(str)
	fmt.Println("rune len:", len(runes))
	fmt.Println("first rune:", string(runes[0]))  // H
	fmt.Println("last rune:", string(runes[len(runes)-1])) // 界

	// String to byte slice — for modification
	bs := []byte("hello")
	bs[0] = 'H'
	fmt.Println(string(bs)) // Hello

	fmt.Println("\n=== MODULE 05 COMPLETE ===")
}
