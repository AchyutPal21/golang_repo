// 04_slice_gotchas.go
// Topic: Classic Go Slice Bugs — every pitfall explained and demonstrated
//
// These bugs are extremely common in real Go codebases. After reading this file,
// you'll be able to spot them in code review and never write them yourself.
//
// Run: go run 05_collections/04_slice_gotchas.go

package main

import (
	"encoding/json"
	"fmt"
)

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 1: Sharing the underlying array — unexpected mutation
// ─────────────────────────────────────────────────────────────────────────────
//
// The most common slice bug. When you slice an existing slice/array,
// the result SHARES the same backing array. Modifying one affects the other.
//
// This often appears when:
// - You return a sub-slice from a function
// - You store a sub-slice in a struct
// - You pass a sub-slice to another goroutine

func gotcha1SharingBackingArray() {
	fmt.Println("=== GOTCHA 1: Sharing the Underlying Array ===")
	fmt.Println()

	// BUG: Both sub-slices share backing array
	data := []int{1, 2, 3, 4, 5, 6, 7, 8}
	first := data[:4]  // [1 2 3 4]  — shares data's backing array
	second := data[4:] // [5 6 7 8]  — shares data's backing array

	fmt.Printf("data:   %v\n", data)
	fmt.Printf("first:  %v\n", first)
	fmt.Printf("second: %v\n", second)

	// Modifying first[0] also modifies data[0]
	first[0] = 999
	fmt.Printf("\nAfter first[0] = 999:\n")
	fmt.Printf("data:  %v  (CHANGED!)\n", data)
	fmt.Printf("first: %v\n", first)

	fmt.Println()

	// THE FIX: copy to create independent data
	data2 := []int{1, 2, 3, 4, 5, 6, 7, 8}
	firstSafe := make([]int, 4)
	copy(firstSafe, data2[:4]) // independent copy

	firstSafe[0] = 999
	fmt.Printf("After fix (copy):\n")
	fmt.Printf("data2:     %v  (unchanged)\n", data2)
	fmt.Printf("firstSafe: %v\n", firstSafe)

	fmt.Println()

	// REAL-WORLD SCENARIO: function returns sub-slice of its buffer
	// Dangerous: callers share the buffer
	parseHeader := func(packet []byte) []byte {
		// BUG: returns sub-slice of packet — caller's data can be modified!
		return packet[:4]
	}
	packet := []byte("HEADPAYLOAD")
	header := parseHeader(packet)
	header[0] = 'X'                    // modifies packet[0]!
	fmt.Printf("Packet after header mutation: %s\n", packet) // XEADPAYLOADs

	// Fix: return a copy
	parseHeaderSafe := func(packet []byte) []byte {
		result := make([]byte, 4)
		copy(result, packet[:4])
		return result
	}
	packet2 := []byte("HEADPAYLOAD")
	header2 := parseHeaderSafe(packet2)
	header2[0] = 'X'
	fmt.Printf("Packet after safe header: %s\n", packet2) // unchanged

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 2: The "full capacity" append bug
// ─────────────────────────────────────────────────────────────────────────────
//
// When you append to a sub-slice that has REMAINING CAPACITY, the new element
// is written to the backing array WITHIN the original slice's range.
// This silently overwrites data in the original slice.
//
// This is the trickiest slice bug — it only appears when cap > len in the sub-slice.

func gotcha2FullCapacityAppend() {
	fmt.Println("=== GOTCHA 2: The Full Capacity Append Bug ===")
	fmt.Println()

	// Create a slice with extra capacity
	original := make([]int, 5, 8)
	original[0] = 1
	original[1] = 2
	original[2] = 3
	original[3] = 4
	original[4] = 5
	fmt.Printf("original: %v  len=%d  cap=%d\n", original, len(original), cap(original))

	// Take a sub-slice
	sub := original[:3] // len=3, cap=8 (inherits full original capacity!)
	fmt.Printf("sub:      %v  len=%d  cap=%d\n", sub, len(sub), cap(sub))

	// Append to sub — doesn't trigger reallocation because cap=8 > len=3
	// The new element writes to original[3]!
	sub = append(sub, 999)
	fmt.Printf("\nAfter append(sub, 999):\n")
	fmt.Printf("sub:      %v  len=%d\n", sub, len(sub))
	fmt.Printf("original: %v  (original[3] was overwritten!)\n", original)

	fmt.Println()

	// THE FIX: use three-index slice to limit capacity
	original2 := make([]int, 5, 8)
	for i := range original2 {
		original2[i] = i + 1
	}
	fmt.Printf("original2: %v  len=%d  cap=%d\n", original2, len(original2), cap(original2))

	// sub2 has len=3, cap=3 (!) — the third index limits capacity
	sub2 := original2[:3:3] // low:high:max → cap = max-low = 3-0 = 3
	fmt.Printf("sub2:      %v  len=%d  cap=%d\n", sub2, len(sub2), cap(sub2))

	// Now append FORCES a new backing array (cap=3, must grow)
	sub2 = append(sub2, 999)
	fmt.Printf("\nAfter append(sub2, 999) with three-index fix:\n")
	fmt.Printf("sub2:      %v  (new backing array)\n", sub2)
	fmt.Printf("original2: %v  (unchanged!)\n", original2)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 3: Loop variable capture with slices of pointers
// ─────────────────────────────────────────────────────────────────────────────
//
// (Note: Go 1.22+ fixed the loop variable capture in for-range. This gotcha
// applies to Go versions < 1.22, but you'll encounter it in existing codebases.)
//
// When you take the address of a loop variable in a closure or slice,
// all pointers point to the SAME variable (which ends up as the last value).

func gotcha3LoopVariableCapture() {
	fmt.Println("=== GOTCHA 3: Loop Variable Capture ===")
	fmt.Println()

	// Building a slice of pointers to loop variable — CLASSIC BUG
	vals := []int{10, 20, 30}
	ptrs := make([]*int, len(vals))

	for i, v := range vals {
		_ = i
		// BUG (pre Go 1.22): v is a SINGLE variable reused each iteration
		// All pointers point to the SAME address (v's address)
		// After the loop, v == 30, so all pointers show 30
		localV := v // FIX: create a new variable in each iteration
		ptrs[i] = &localV
	}

	fmt.Println("Pointers to local copies (correct):")
	for i, p := range ptrs {
		fmt.Printf("  ptrs[%d] = %d\n", i, *p) // 10, 20, 30 ✓
	}

	fmt.Println()

	// The goroutine version of this bug (still relevant in Go 1.22+):
	// Launching goroutines in a loop that capture the loop variable
	// Demo without goroutines — same principle
	type Task struct {
		ID   int
		Name string
	}
	tasks := []Task{{1, "a"}, {2, "b"}, {3, "c"}}
	handlers := make([]func() string, len(tasks))

	for i, t := range tasks {
		t := t // FIX: shadow with new variable (works in all Go versions)
		handlers[i] = func() string {
			return fmt.Sprintf("task %d: %s", t.ID, t.Name)
		}
	}

	fmt.Println("Closures with proper capture:")
	for _, h := range handlers {
		fmt.Printf("  %s\n", h())
	}

	fmt.Println()

	// Alternative fix: capture by index
	handlers2 := make([]func() string, len(tasks))
	for i := range tasks {
		i := i // capture index (or use direct []index access in closure)
		handlers2[i] = func() string {
			return fmt.Sprintf("task %d: %s", tasks[i].ID, tasks[i].Name)
		}
	}
	fmt.Println("Closures via index capture:")
	for _, h := range handlers2 {
		fmt.Printf("  %s\n", h())
	}

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 4: nil vs empty slice — JSON marshaling difference
// ─────────────────────────────────────────────────────────────────────────────
//
// This causes real API bugs: returning null vs [] from JSON endpoints.
// The fix is simple once you know about it.

type Response struct {
	Items []string `json:"items"`
}

func gotcha4NilVsEmptyJSON() {
	fmt.Println("=== GOTCHA 4: nil vs Empty Slice in JSON ===")
	fmt.Println()

	// nil slice → JSON null
	nilResp := Response{Items: nil}
	nilJSON, _ := json.Marshal(nilResp)
	fmt.Printf("nil slice  → JSON: %s\n", nilJSON)

	// empty slice → JSON []
	emptyResp := Response{Items: []string{}}
	emptyJSON, _ := json.Marshal(emptyResp)
	fmt.Printf("empty slice → JSON: %s\n", emptyJSON)

	fmt.Println()

	// THE BUG: function returns nil when no items found
	findItems := func(query string) []string {
		results := []string{"apple", "banana"} // simulate DB
		var found []string                      // nil slice!
		for _, r := range results {
			if r == query {
				found = append(found, r) // only appends if match
			}
		}
		return found // returns nil if no match!
	}

	found := findItems("orange") // not found → nil
	resp := Response{Items: found}
	jsonBytes, _ := json.Marshal(resp)
	fmt.Printf("BUG: findItems('orange') JSON: %s  (null!)\n", jsonBytes)

	// THE FIX: initialize to empty slice, not nil
	findItemsSafe := func(query string) []string {
		results := []string{"apple", "banana"}
		found := []string{} // empty slice instead of nil
		for _, r := range results {
			if r == query {
				found = append(found, r)
			}
		}
		return found
	}

	foundSafe := findItemsSafe("orange")
	respSafe := Response{Items: foundSafe}
	jsonSafe, _ := json.Marshal(respSafe)
	fmt.Printf("FIX: findItemsSafe('orange') JSON: %s  ([] not null)\n", jsonSafe)

	fmt.Println()

	// Another common fix: check nil before return
	coalesce := func(s []string) []string {
		if s == nil {
			return []string{}
		}
		return s
	}
	_ = coalesce

	// Or using the fact that append(nil, s...) returns non-nil if s is non-nil
	// For guaranteed non-nil: use make([]T, 0) or []T{}

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 5: slice of slice sharing memory (unexpected mutation at distance)
// ─────────────────────────────────────────────────────────────────────────────

func gotcha5SliceOfSlice() {
	fmt.Println("=== GOTCHA 5: Slice-of-Slice Sharing Memory ===")
	fmt.Println()

	// Two sub-slices from the SAME backing array — modifying one affects both
	backing := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	a := backing[0:5]
	b := backing[3:8]
	// a and b OVERLAP at indices 3 and 4 of backing

	fmt.Printf("a (backing[0:5]): %v\n", a)
	fmt.Printf("b (backing[3:8]): %v\n", b)
	fmt.Printf("a[3]=%d, b[0]=%d — same element!\n", a[3], b[0])

	a[3] = 999
	fmt.Printf("\nAfter a[3] = 999:\n")
	fmt.Printf("a: %v\n", a)
	fmt.Printf("b: %v  (b[0] changed!)\n", b) // b[0] is now 999

	fmt.Println()

	// Practical bug: processing overlapping windows of data
	data := []byte("abcdefghij")
	window1 := data[0:5]
	window2 := data[2:7]

	fmt.Printf("window1: %s\n", window1)
	fmt.Printf("window2: %s\n", window2)

	// "Processing" window1 modifies shared bytes
	for i := range window1 {
		window1[i] = window1[i] - 32 // uppercase
	}
	fmt.Printf("\nAfter uppercasing window1:\n")
	fmt.Printf("window1: %s\n", window1) // ABCDE
	fmt.Printf("window2: %s (partial change!)\n", window2) // CDEfg (overlap uppercased!)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 6: 2D slice allocation — common wrong pattern
// ─────────────────────────────────────────────────────────────────────────────
//
// Creating a 2D slice correctly requires understanding allocation.
// A common but WRONG pattern creates all rows sharing one backing array.

func gotcha6TwoDSlice() {
	fmt.Println("=== GOTCHA 6: 2D Slice Allocation ===")
	fmt.Println()

	// WRONG WAY: allocate one big backing array, slice it into rows
	// All rows share the SAME backing array!
	rows, cols := 3, 4
	backing := make([]int, rows*cols) // one big array
	grid := make([][]int, rows)
	for i := range grid {
		grid[i] = backing[i*cols : (i+1)*cols]
	}

	// The grid looks correct:
	for i := range grid {
		for j := range grid[i] {
			grid[i][j] = i*cols + j
		}
	}
	fmt.Println("Grid (correct values):")
	for _, row := range grid {
		fmt.Printf("  %v\n", row)
	}

	// But appending to one row overwrites the next row's data!
	grid[0] = append(grid[0], 99, 88, 77)
	fmt.Printf("\nAfter appending to row 0:\n")
	fmt.Printf("row 0: %v\n", grid[0])
	fmt.Printf("row 1: %v  (CORRUPTED! 99,88,77 overwrote its start)\n", grid[1])

	fmt.Println()

	// CORRECT WAY: each row is independently allocated
	grid2 := make([][]int, rows)
	for i := range grid2 {
		grid2[i] = make([]int, cols) // separate allocation per row
	}
	for i := range grid2 {
		for j := range grid2[i] {
			grid2[i][j] = i*cols + j
		}
	}

	// Appending to row 0 now allocates new backing array for that row only
	grid2[0] = append(grid2[0], 99, 88, 77)
	fmt.Println("Grid (independent rows):")
	fmt.Printf("row 0: %v\n", grid2[0])
	fmt.Printf("row 1: %v  (unchanged!)\n", grid2[1])

	fmt.Println()

	// ALTERNATIVE: the "big array" approach IS correct if you NEVER append to rows
	// It provides better cache locality (all data is contiguous in memory)
	// The bug only appears when rows can grow via append

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// GOTCHA 7: Memory leak — large array kept alive by small slice
// ─────────────────────────────────────────────────────────────────────────────
//
// If you keep a small slice that references a large backing array,
// the ENTIRE backing array stays in memory — the GC cannot collect it.
//
// This happens when:
// - You load a large file into a []byte
// - You slice out a small portion (e.g., header)
// - You store only the small slice but forget the large array is still live

func gotcha7MemoryLeak() {
	fmt.Println("=== GOTCHA 7: Memory Leak via Large Backing Array ===")
	fmt.Println()

	// Simulate: load large data, extract small portion
	simulateLargeLoad := func() []byte {
		// Imagine this is a 10MB file read into memory
		largeData := make([]byte, 1024) // using 1024 for demo
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		return largeData
	}

	// BUG: stores only first 8 bytes but keeps 1024-byte array alive!
	bigData := simulateLargeLoad()
	leakySlice := bigData[:8] // len=8, but cap=1024!
	bigData = nil             // setting bigData nil doesn't help!
	// leakySlice still holds a pointer to the 1024-byte backing array.
	// GC cannot collect those 1024 bytes!

	fmt.Printf("leakySlice: %v  cap=%d  (holding 1024 bytes in memory!)\n",
		leakySlice, cap(leakySlice))

	// FIX: copy the small portion into its own backing array
	safeCopy := make([]byte, 8)
	copy(safeCopy, leakySlice)
	leakySlice = nil // now the large backing array can be GC'd
	fmt.Printf("safeCopy:   %v  cap=%d  (only holds 8 bytes)\n",
		safeCopy, cap(safeCopy))

	fmt.Println()
	fmt.Println("Rule: if you extract a small sub-slice from large data,")
	fmt.Println("      copy it to a new slice to allow GC of the large backing array.")
	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║       Classic Go Slice Bugs: Know Them All           ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	gotcha1SharingBackingArray()
	gotcha2FullCapacityAppend()
	gotcha3LoopVariableCapture()
	gotcha4NilVsEmptyJSON()
	gotcha5SliceOfSlice()
	gotcha6TwoDSlice()
	gotcha7MemoryLeak()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║  GOTCHA CHEAT SHEET                                  ║")
	fmt.Println("║  1. Sub-slice shares backing — copy if you mutate    ║")
	fmt.Println("║  2. Append to sub-slice can overwrite original       ║")
	fmt.Println("║     → use [lo:hi:max] to limit capacity              ║")
	fmt.Println("║  3. Loop var capture: shadow or use index            ║")
	fmt.Println("║  4. nil slice → JSON null; use []T{} for []          ║")
	fmt.Println("║  5. Overlapping sub-slices mutate each other         ║")
	fmt.Println("║  6. 2D slices: independent rows for safe append      ║")
	fmt.Println("║  7. Small slice of large array = memory leak         ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
}
