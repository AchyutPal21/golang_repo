// 02_slice_internals.go
// Topic: Slice Internals — THE most important collection topic in Go
//
// Slices look simple on the surface but have deep mechanics that trip up
// nearly every Go programmer at least once. This file builds a MENTAL MODEL
// of exactly what happens in memory, enabling you to predict behavior
// precisely — not just know the rules, but understand WHY they exist.
//
// Run: go run 05_collections/02_slice_internals.go

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: The Slice Header — what a slice ACTUALLY is
// ─────────────────────────────────────────────────────────────────────────────
//
// A slice is NOT a dynamic array. It is a DESCRIPTOR — a small struct with
// exactly 3 fields:
//
//   type sliceHeader struct {
//       Data unsafe.Pointer  // pointer to first element in backing array
//       Len  int             // number of elements accessible
//       Cap  int             // total capacity of backing array from Data
//   }
//
// On a 64-bit system: 8 + 8 + 8 = 24 bytes. That's ALL a slice variable stores.
//
// The ACTUAL data lives in a separate backing array, allocated on the heap.
//
// Visual:
//
//   slice variable (24 bytes on stack):
//   ┌──────────────────┬─────┬─────┐
//   │ Data ptr         │ Len │ Cap │
//   │ → heap address   │  3  │  5  │
//   └──────────────────┴─────┴─────┘
//                │
//                ▼ (heap)
//   ┌─────┬─────┬─────┬─────┬─────┐
//   │  1  │  2  │  3  │     │     │
//   └─────┴─────┴─────┴─────┴─────┘
//    [0]   [1]   [2]   [3]   [4]    ← backing array indices
//    accessible ──────┘  not yet accessible (within cap)

// printSliceHeader shows the underlying pointer, len, and cap
// using the reflect package — safe way to inspect slice internals
func printSliceHeader(name string, s []int) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	fmt.Printf("%-10s → ptr=0x%x  len=%-3d  cap=%d\n",
		name, header.Data, len(s), cap(s))
}

func section1SliceHeader() {
	fmt.Println("=== SECTION 1: The Slice Header ===")
	fmt.Println()

	// The slice header size is always 24 bytes (on 64-bit)
	var s []int
	fmt.Printf("Size of slice header: %d bytes\n", unsafe.Sizeof(s))
	// For comparison: size of the int element itself
	fmt.Printf("Size of int:          %d bytes\n", unsafe.Sizeof(int(0)))
	fmt.Println()

	// Create a slice with make([]T, len, cap)
	// This allocates a backing array of capacity 5, len 3
	s = make([]int, 3, 5)
	s[0], s[1], s[2] = 10, 20, 30

	printSliceHeader("s", s)
	// The pointer tells us WHERE in memory the data lives

	// Create another slice pointing to the SAME backing array
	// s2 shares memory with s — they have the SAME Data pointer
	s2 := s[:2] // sub-slice: same start, len=2, cap=5
	printSliceHeader("s2=s[:2]", s2)

	// Modify through s2: affects s too (shared backing array)
	s2[0] = 999
	fmt.Printf("After s2[0]=999:  s=%v, s2=%v\n", s, s2)
	// Both show 999 at index 0!

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: nil slice vs empty slice
// ─────────────────────────────────────────────────────────────────────────────
//
// This distinction is subtle but CRITICAL, especially when dealing with JSON.
//
// nil slice:   var s []int         → Data=nil, Len=0, Cap=0
// empty slice: s := []int{}        → Data=<non-nil>, Len=0, Cap=0
//              s := make([]int, 0) → Data=<non-nil>, Len=0, Cap=0
//
// Both behave identically for: len(), cap(), range, append(), indexing
// They differ in: == nil check, json.Marshal output

func section2NilVsEmpty() {
	fmt.Println("=== SECTION 2: nil Slice vs Empty Slice ===")
	fmt.Println()

	// Nil slice: declared but not initialized
	var nilSlice []int
	fmt.Printf("nil slice:      value=%v  len=%d  cap=%d  isNil=%v\n",
		nilSlice, len(nilSlice), cap(nilSlice), nilSlice == nil)

	// Empty slice: initialized but has no elements
	emptySlice1 := []int{}
	fmt.Printf("empty []int{}:  value=%v  len=%d  cap=%d  isNil=%v\n",
		emptySlice1, len(emptySlice1), cap(emptySlice1), emptySlice1 == nil)

	emptySlice2 := make([]int, 0)
	fmt.Printf("make([]int,0):  value=%v  len=%d  cap=%d  isNil=%v\n",
		emptySlice2, len(emptySlice2), cap(emptySlice2), emptySlice2 == nil)

	fmt.Println()

	// THE CRITICAL DIFFERENCE: JSON marshaling
	nilJSON, _ := json.Marshal(nilSlice)
	emptyJSON, _ := json.Marshal(emptySlice1)
	fmt.Printf("JSON(nil slice):   %s\n", nilJSON)    // null
	fmt.Printf("JSON(empty slice): %s\n", emptyJSON)  // []
	// If your API returns null vs [] differently, this is the root cause!

	fmt.Println()

	// Safe to use nil slice with append, len, cap, range
	fmt.Printf("append to nil: %v\n", append(nilSlice, 1, 2, 3))
	fmt.Printf("len(nil):      %d\n", len(nilSlice))
	for i, v := range nilSlice {
		_ = i
		_ = v
		fmt.Println("This never prints — ranging over nil is fine, 0 iterations")
	}

	// READING from nil slice: OK (no panic)
	// WRITING to nil slice: nil[0] = 1 would PANIC
	// This is a common mistake: forgot to initialize the slice before indexing

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Slice literals and their backing arrays
// ─────────────────────────────────────────────────────────────────────────────
//
// A slice literal creates a backing array AND a slice header pointing to it.
// The capacity equals the length (no extra room).

func section3SliceLiterals() {
	fmt.Println("=== SECTION 3: Slice Literals ===")
	fmt.Println()

	// Basic literal: len=cap=4
	s := []int{10, 20, 30, 40}
	printSliceHeader("literal", s)

	// Sparse literal (like array but creates a slice)
	sparse := []int{0: 100, 5: 500}
	printSliceHeader("sparse", sparse) // len=6, cap=6

	fmt.Printf("sparse: %v\n", sparse)

	// Slice of structs
	type Point struct{ X, Y int }
	points := []Point{{1, 2}, {3, 4}, {5, 6}}
	fmt.Printf("points: %v\n", points)

	// Slice of slices (2D slice) — each inner slice is independent
	matrix := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	fmt.Printf("matrix[1][2] = %d\n", matrix[1][2]) // 6

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: make([]T, len, cap) — explicit allocation
// ─────────────────────────────────────────────────────────────────────────────
//
// make is a built-in function that allocates a new backing array and
// returns a slice header pointing to it. All elements are zero-initialized.
//
// Signature: make([]T, length, capacity)
//            make([]T, length)  → capacity == length
//
// WHEN TO USE make:
// 1. You know the final size upfront → allocate once, no re-allocation
// 2. You want to pre-allocate capacity to avoid appends causing re-alloc
// 3. Building a result set with known size

func section4Make() {
	fmt.Println("=== SECTION 4: make([]T, len, cap) ===")
	fmt.Println()

	// make with length only: len=cap=5
	s1 := make([]int, 5)
	printSliceHeader("make(5)", s1)
	fmt.Printf("  elements: %v (zero-initialized)\n", s1)

	// make with explicit capacity: len=3, cap=10
	// Use this when you'll be appending elements and want to avoid re-allocation
	s2 := make([]int, 3, 10)
	printSliceHeader("make(3,10)", s2)
	fmt.Printf("  elements: %v\n", s2)
	fmt.Printf("  cap-len = %d slots available before re-allocation\n", cap(s2)-len(s2))

	fmt.Println()

	// PATTERN: Pre-allocate for performance
	// If you know you'll append N items, pre-allocate cap=N
	n := 1000
	withoutHint := make([]int, 0)
	withHint := make([]int, 0, n)

	// Both produce the same result, but withHint avoids ~10 re-allocations
	for i := 0; i < n; i++ {
		withoutHint = append(withoutHint, i)
		withHint = append(withHint, i)
	}
	fmt.Printf("Without hint: len=%d, cap=%d\n", len(withoutHint), cap(withoutHint))
	fmt.Printf("With hint:    len=%d, cap=%d\n", len(withHint), cap(withHint))
	// withHint.cap == 1000 (no extra allocation)
	// withoutHint.cap == 1024 (grown through doubling: 0→1→2→4→8...→1024)

	fmt.Println()

	// PATTERN: Pre-fill vs pre-allocate
	// make([]T, n)   → fills n elements with zero, use for index-based writes
	// make([]T, 0, n) → empty slice, use for append-based accumulation
	preFilled := make([]int, 5)   // [0 0 0 0 0] — write with preFilled[i]=x
	preAlloc := make([]int, 0, 5) // []          — grow with append

	preFilled[2] = 42
	preAlloc = append(preAlloc, 42)
	fmt.Printf("preFilled: %v\n", preFilled)
	fmt.Printf("preAlloc:  %v\n", preAlloc)

	// Common mistake: using make([]T, n) and then appending
	// This adds AFTER the n zero elements, not AT them!
	mistakeSlice := make([]int, 3)
	mistakeSlice = append(mistakeSlice, 99)
	fmt.Printf("Mistake (make+append): %v (99 is at index 3!)\n", mistakeSlice)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Visualizing memory layout with pointer addresses
// ─────────────────────────────────────────────────────────────────────────────
//
// Let's prove the shared backing array model with actual addresses.

func section5MemoryLayout() {
	fmt.Println("=== SECTION 5: Memory Layout Visualization ===")
	fmt.Println()

	// Create a backing array manually
	backing := [10]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Create slices referencing different parts of the same backing array
	all := backing[:]       // ptr=&backing[0], len=10, cap=10
	first := backing[:5]    // ptr=&backing[0], len=5,  cap=10
	second := backing[5:]   // ptr=&backing[5], len=5,  cap=5
	middle := backing[3:7]  // ptr=&backing[3], len=4,  cap=7

	fmt.Printf("backing array address: %p\n", &backing[0])
	fmt.Println()
	printSliceHeader("all", all)
	printSliceHeader("first", first)
	printSliceHeader("second", second)
	printSliceHeader("middle", middle)

	fmt.Println()
	fmt.Println("Proving shared memory by modifying backing[3]:")
	backing[3] = 999
	fmt.Printf("  all[3]    = %d\n", all[3])    // 999
	fmt.Printf("  first[3]  = %d\n", first[3])  // 999
	fmt.Printf("  middle[0] = %d\n", middle[0]) // 999
	fmt.Printf("  second[0] = %d\n", second[0]) // 5 (not affected)

	fmt.Println()

	// Address arithmetic: prove second starts 5 ints (40 bytes) after first
	firstPtr := (*reflect.SliceHeader)(unsafe.Pointer(&first)).Data
	secondPtr := (*reflect.SliceHeader)(unsafe.Pointer(&second)).Data
	diff := secondPtr - firstPtr
	fmt.Printf("Pointer difference: second - first = %d bytes = %d ints\n",
		diff, diff/8) // 40 bytes = 5 × 8-byte ints

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Why modifying a slice element affects the original
// ─────────────────────────────────────────────────────────────────────────────
//
// Slices are REFERENCE types in the sense that the slice header contains
// a POINTER to the backing array. When you pass a slice to a function:
// - The slice HEADER is copied (pointer+len+cap, 24 bytes)
// - The backing array DATA is NOT copied
//
// So the function has its own len/cap but points to the SAME data.
//
// This means element modifications PROPAGATE but len/cap changes do NOT.

// modifyElement: the function gets its own slice header copy
// but the pointer inside still points to the caller's backing array
func modifyElement(s []int) {
	s[0] = 9999 // ← writes to shared backing array
	fmt.Printf("  Inside function: s=%v (ptr=%p)\n", s, &s[0])
}

// modifyViaAppend: appending to a slice with available cap mutates
// the CALLER'S backing array but does NOT change caller's len!
func modifyViaAppend(s []int) {
	// This append uses existing capacity, writing to backing array[len]
	// The caller's slice does NOT see the new element (len unchanged in caller)
	// but their backing array IS overwritten at that position!
	s = append(s, 777)
	fmt.Printf("  Inside append func: s=%v len=%d cap=%d\n", s, len(s), cap(s))
}

func section6WhyModificationsPropagate() {
	fmt.Println("=== SECTION 6: Why Modifications Propagate ===")
	fmt.Println()

	// Demonstrate element modification propagation
	original := []int{1, 2, 3, 4, 5}
	fmt.Printf("Before call:      original=%v (ptr=%p)\n", original, &original[0])
	modifyElement(original)
	fmt.Printf("After modifyElement: original=%v\n", original)
	// original[0] is now 9999 — function DID modify it!
	fmt.Println()

	// Demonstrate the sneaky append scenario
	// Create a slice with spare capacity
	danger := make([]int, 3, 6)
	danger[0], danger[1], danger[2] = 10, 20, 30
	printSliceHeader("danger before", danger)

	// Take a sub-slice (shares same backing array, same cap start)
	sub := danger[:3] // identical to danger here
	printSliceHeader("sub=danger[:3]", sub)

	fmt.Printf("Before modifyViaAppend: danger=%v\n", danger)
	modifyViaAppend(sub)
	// The append in the function wrote 777 to backing[3]
	// danger's len is still 3, so danger cannot SEE index 3
	// BUT if we extend danger to cap, we'll see it!
	dangerExtended := danger[:cap(danger)]
	fmt.Printf("After modifyViaAppend: danger=%v\n", danger)
	fmt.Printf("Extended to cap: %v (777 is there!)\n", dangerExtended)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: When does Go allocate a NEW backing array?
// ─────────────────────────────────────────────────────────────────────────────
//
// A new backing array is allocated ONLY when:
// 1. append() is called and len == cap (no more room)
// 2. You explicitly create a new slice/make
// 3. copy() is used (always creates independent data)
//
// If len < cap, append REUSES the existing backing array.
// This is the source of the "full capacity" bug (see 04_slice_gotchas.go).

func section7WhenNewArrayAllocated() {
	fmt.Println("=== SECTION 7: When Does Go Allocate a New Backing Array? ===")
	fmt.Println()

	s := make([]int, 3, 5) // len=3, cap=5 (2 extra slots)
	s[0], s[1], s[2] = 1, 2, 3
	printSliceHeader("s (before append)", s)

	// Append 1 element: len < cap → reuses backing array
	s2 := append(s, 4)
	printSliceHeader("s (after s2=append)", s)
	printSliceHeader("s2", s2)
	// s and s2 have the SAME Data pointer — same backing array!
	fmt.Printf("Same backing array: %v\n",
		(*reflect.SliceHeader)(unsafe.Pointer(&s)).Data ==
			(*reflect.SliceHeader)(unsafe.Pointer(&s2)).Data)

	fmt.Println()

	// Append until cap is exceeded: NEW backing array allocated
	s3 := append(s2, 5) // now at len=5, cap=5 — FULL
	s4 := append(s3, 6) // len=6 > cap=5 → NEW array!
	printSliceHeader("s3 (full)", s3)
	printSliceHeader("s4 (grew)", s4)
	fmt.Printf("Same backing array s3/s4: %v\n",
		(*reflect.SliceHeader)(unsafe.Pointer(&s3)).Data ==
			(*reflect.SliceHeader)(unsafe.Pointer(&s4)).Data)
	// false! s4 has a new backing array

	fmt.Println()
	fmt.Println("Rule: append returns a new slice — ALWAYS assign it back!")
	fmt.Println("  s = append(s, x)  ← correct")
	fmt.Println("  append(s, x)      ← WRONG: return value discarded")
	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║       Go Slice Internals: Deep Mental Model          ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	section1SliceHeader()
	section2NilVsEmpty()
	section3SliceLiterals()
	section4Make()
	section5MemoryLayout()
	section6WhyModificationsPropagate()
	section7WhenNewArrayAllocated()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║  MENTAL MODEL SUMMARY                                ║")
	fmt.Println("║  1. Slice = 24-byte header (ptr + len + cap)         ║")
	fmt.Println("║  2. Data lives in a separate heap-allocated array     ║")
	fmt.Println("║  3. Multiple slices can share one backing array       ║")
	fmt.Println("║  4. nil ≠ empty: JSON difference (null vs [])        ║")
	fmt.Println("║  5. Element mod propagates; len/cap change does not   ║")
	fmt.Println("║  6. New backing array only when len exceeds cap       ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
}
