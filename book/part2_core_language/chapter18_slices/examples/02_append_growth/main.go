// FILE: book/part2_core_language/chapter18_slices/examples/02_append_growth/main.go
// CHAPTER: 18 — Slices: The Most Important Type
// TOPIC: append semantics, capacity growth, pre-allocation with make,
//        appending slices, delete patterns, insert patterns.
//
// Run (from the chapter folder):
//   go run ./examples/02_append_growth

package main

import "fmt"

// trackGrowth appends to a nil slice and prints when the backing array
// is reallocated (detectable by cap change).
func trackGrowth() {
	var s []int
	prevCap := 0
	for i := range 33 {
		s = append(s, i)
		if cap(s) != prevCap {
			fmt.Printf("  len=%-3d cap=%d (grew)\n", len(s), cap(s))
			prevCap = cap(s)
		}
	}
}

// preallocated builds the same slice with pre-allocated capacity.
// Zero reallocations.
func preallocated(n int) []int {
	s := make([]int, 0, n) // len=0, cap=n
	for i := range n {
		s = append(s, i*i)
	}
	return s
}

// appendSlice shows appending one slice to another with ...
func appendSlice() {
	a := []int{1, 2, 3}
	b := []int{4, 5, 6}
	c := append(a, b...)
	fmt.Println("append(a, b...):", c)
}

// deleteUnordered removes index i by swapping with last element and shrinking.
// O(1), does not preserve order.
func deleteUnordered(s []int, i int) []int {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// deleteOrdered removes index i while preserving order. O(n).
func deleteOrdered(s []int, i int) []int {
	return append(s[:i], s[i+1:]...)
}

// insert inserts v at index i, shifting elements right. O(n).
func insert(s []int, i, v int) []int {
	s = append(s, 0)        // grow by 1
	copy(s[i+1:], s[i:])   // shift right
	s[i] = v
	return s
}

// dedupe removes consecutive duplicates (input must be sorted).
func dedupe(s []int) []int {
	if len(s) == 0 {
		return s
	}
	out := s[:1]
	for _, v := range s[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

// filter returns a new slice containing only elements where keep(v) is true.
// Uses a[:0] to avoid allocation when the result fits in the original backing array.
func filter(s []int, keep func(int) bool) []int {
	out := s[:0]
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func main() {
	fmt.Println("=== capacity growth ===")
	trackGrowth()

	fmt.Println()
	fmt.Println("=== pre-allocated ===")
	sq := preallocated(8)
	fmt.Println("squares:", sq)

	fmt.Println()
	appendSlice()

	fmt.Println()
	fmt.Println("=== delete unordered ===")
	s := []int{10, 20, 30, 40, 50}
	s = deleteUnordered(s, 1) // remove 20
	fmt.Println("after deleteUnordered(s,1):", s)

	fmt.Println()
	fmt.Println("=== delete ordered ===")
	s = []int{10, 20, 30, 40, 50}
	s = deleteOrdered(s, 1) // remove 20
	fmt.Println("after deleteOrdered(s,1):", s)

	fmt.Println()
	fmt.Println("=== insert ===")
	s = []int{1, 2, 4, 5}
	s = insert(s, 2, 3) // insert 3 at index 2
	fmt.Println("after insert(s,2,3):", s)

	fmt.Println()
	fmt.Println("=== dedupe ===")
	sorted := []int{1, 1, 2, 3, 3, 3, 4, 5, 5}
	fmt.Println("dedupe:", dedupe(sorted))

	fmt.Println()
	fmt.Println("=== filter ===")
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	evens := filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("evens:", evens)
}
