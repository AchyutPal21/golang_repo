// FILE: book/part2_core_language/chapter18_slices/examples/03_aliasing_traps/main.go
// CHAPTER: 18 — Slices: The Most Important Type
// TOPIC: Aliasing bugs, the "append past capacity" trap, function mutation,
//        defensive copy, slices of slices.
//
// Run (from the chapter folder):
//   go run ./examples/03_aliasing_traps

package main

import "fmt"

// --- Trap 1: two slices share backing array ---

func trap1SharedBacking() {
	original := []int{1, 2, 3, 4, 5}
	alias := original[1:4] // shares backing array

	fmt.Println("original:", original)
	fmt.Println("alias:   ", alias)

	alias[0] = 200 // modifies original[1]
	fmt.Println("after alias[0]=200:")
	fmt.Println("  original:", original) // [1 200 3 4 5]
	fmt.Println("  alias:   ", alias)    // [200 3 4]
}

// --- Trap 2: append within capacity mutates alias ---

func trap2AppendWithinCap() {
	base := make([]int, 3, 6) // len=3, cap=6
	base[0], base[1], base[2] = 1, 2, 3
	sub := base[:2] // shares backing array, cap=6

	fmt.Println("base:", base, "len:", len(base), "cap:", cap(base))
	fmt.Println("sub: ", sub, "len:", len(sub), "cap:", cap(sub))

	// append to sub: stays within cap, writes into base's backing array
	sub = append(sub, 99)
	fmt.Println("after append(sub, 99):")
	fmt.Println("  sub: ", sub)  // [1 2 99]
	fmt.Println("  base:", base) // [1 2 99] — base[2] changed!
}

// --- Fix: 3-index slice to limit cap ---

func fix3IndexSlice() {
	base := make([]int, 3, 6)
	base[0], base[1], base[2] = 1, 2, 3
	// sub has cap=2, so append will allocate a new backing array.
	sub := base[:2:2]

	sub = append(sub, 99)
	fmt.Println("3-index fix:")
	fmt.Println("  sub: ", sub)  // [1 2 99]
	fmt.Println("  base:", base) // [1 2 3] — unchanged
}

// --- Trap 3: function appends past caller's cap ---

// appendBad does not return the modified slice; the caller's slice is
// unchanged because append may have created a new backing array.
func appendBad(s []int, v int) {
	s = append(s, v) // new header; caller doesn't see it
}

// appendGood returns the new slice — the caller must reassign.
func appendGood(s []int, v int) []int {
	return append(s, v)
}

// appendViaPointer updates the caller's slice header directly.
func appendViaPointer(sp *[]int, v int) {
	*sp = append(*sp, v)
}

// --- Defensive copy ---

// safeSub returns a copy of s[lo:hi] so callers cannot alias the original.
func safeSub(s []int, lo, hi int) []int {
	cp := make([]int, hi-lo)
	copy(cp, s[lo:hi])
	return cp
}

// --- Slice of slices ---

func sliceOfSlices() {
	grid := make([][]int, 3)
	for i := range grid {
		grid[i] = make([]int, 4)
		for j := range grid[i] {
			grid[i][j] = i*4 + j
		}
	}
	fmt.Println("grid:")
	for _, row := range grid {
		fmt.Println(" ", row)
	}
	// Each row is an independent slice — no shared backing array.
}

func main() {
	fmt.Println("=== trap 1: shared backing ===")
	trap1SharedBacking()

	fmt.Println()
	fmt.Println("=== trap 2: append within cap ===")
	trap2AppendWithinCap()

	fmt.Println()
	fmt.Println("=== fix: 3-index slice ===")
	fix3IndexSlice()

	fmt.Println()
	fmt.Println("=== trap 3: function append ===")
	s := []int{1, 2, 3}
	appendBad(s, 4)
	fmt.Println("after appendBad:", s) // [1 2 3] — 4 was lost

	s = appendGood(s, 4)
	fmt.Println("after appendGood:", s) // [1 2 3 4]

	appendViaPointer(&s, 5)
	fmt.Println("after appendViaPointer:", s) // [1 2 3 4 5]

	fmt.Println()
	fmt.Println("=== defensive copy ===")
	orig := []int{10, 20, 30, 40, 50}
	sub := safeSub(orig, 1, 4)
	sub[0] = 999
	fmt.Println("orig after modifying safeSub result:", orig) // unchanged

	fmt.Println()
	fmt.Println("=== slice of slices ===")
	sliceOfSlices()
}
