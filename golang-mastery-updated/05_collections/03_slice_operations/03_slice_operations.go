// 03_slice_operations.go
// Topic: Slice Operations — append, copy, slicing expressions, and mutations
//
// This file covers every practical operation you'll perform on slices:
// how they work mechanically, edge cases, and idiomatic Go patterns.
//
// Run: go run 05_collections/03_slice_operations.go

package main

import (
	"fmt"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: append — how it grows the slice
// ─────────────────────────────────────────────────────────────────────────────
//
// append(s []T, elems ...T) []T
//
// The growth algorithm (as of Go 1.18+):
//   - If new len <= 2 × old cap: double the capacity
//   - If new len > 2 × old cap: use new len as cap (rare: bulk append)
//   - Once cap >= 256: grow by 25% + smoothing factor (not exactly 1.25x)
//   - The exact sizes are rounded to memory allocator size classes
//
// KEY RULE: append ALWAYS returns a new slice header.
// The new slice might point to the old backing array (if cap was sufficient)
// OR to a newly allocated one. You cannot know which — always reassign!
//
//   s = append(s, x)  ← ALWAYS do this

func section1AppendGrowth() {
	fmt.Println("=== SECTION 1: How append Grows ===")
	fmt.Println()

	// Watch capacity growth as we repeatedly append
	var s []int
	fmt.Printf("%-3s  %-4s  %-4s\n", "len", "cap", "grew?")
	fmt.Println("---  ----  -----")

	prevCap := 0
	for i := 0; i < 20; i++ {
		s = append(s, i)
		grew := ""
		if cap(s) != prevCap {
			grew = fmt.Sprintf("← cap was %d, now %d", prevCap, cap(s))
			prevCap = cap(s)
		}
		fmt.Printf("%-3d  %-4d  %s\n", len(s), cap(s), grew)
	}

	fmt.Println()

	// Growth pattern for small slices: 0→1→2→4→8→16
	// Once past 256 elements, growth slows to ~25%
	// Let's observe the 256 threshold:
	fmt.Println("Around the 256 threshold:")
	s2 := make([]int, 0, 240)
	prevCap2 := cap(s2)
	for i := 0; i < 50; i++ {
		s2 = append(s2, i)
		if cap(s2) != prevCap2 {
			fmt.Printf("  len=%-4d  cap grew: %d → %d (ratio %.2f)\n",
				len(s2), prevCap2, cap(s2), float64(cap(s2))/float64(prevCap2))
			prevCap2 = cap(s2)
		}
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: append variants
// ─────────────────────────────────────────────────────────────────────────────

func section2AppendVariants() {
	fmt.Println("=== SECTION 2: append Variants ===")
	fmt.Println()

	// 1. Append to nil slice — perfectly valid, allocates new backing array
	var nilSlice []string
	nilSlice = append(nilSlice, "first")
	nilSlice = append(nilSlice, "second")
	fmt.Printf("Append to nil: %v\n", nilSlice)

	// 2. Append multiple elements at once
	s := []int{1, 2, 3}
	s = append(s, 4, 5, 6) // variadic: multiple elements in one call
	fmt.Printf("Append multiple: %v\n", s)

	// 3. Append one slice to another using spread operator (...)
	// append(s, other...) unpacks other into the variadic parameter
	a := []int{1, 2, 3}
	b := []int{4, 5, 6}
	a = append(a, b...) // ... is the "spread" or "unpack" operator
	fmt.Printf("Append slice to slice: %v\n", a)

	// Equivalent to: append(a, b[0], b[1], b[2])

	// 4. Common pattern: build a slice by conditional appending
	result := make([]int, 0, 10)
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			result = append(result, i)
		}
	}
	fmt.Printf("Conditional append: %v\n", result)

	// 5. Append bytes to a string-based slice
	words := []string{"hello", "world"}
	moreWords := []string{"foo", "bar", "baz"}
	words = append(words, moreWords...)
	fmt.Printf("Append strings: %v\n", words)

	// 6. Append string bytes to []byte
	// append([]byte, string...) is a special Go case — no need to convert
	buf := []byte("Hello")
	buf = append(buf, ", World"...)
	fmt.Printf("Append string to []byte: %s\n", buf)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: copy() — creating INDEPENDENT slices
// ─────────────────────────────────────────────────────────────────────────────
//
// copy(dst, src []T) int
//
// Copies min(len(dst), len(src)) elements from src into dst.
// Returns the number of elements copied.
// dst and src may overlap — copy handles it correctly.
//
// KEY: copy always creates INDEPENDENT data in dst.
// After copy, dst and src do NOT share a backing array.

func section3Copy() {
	fmt.Println("=== SECTION 3: copy() ===")
	fmt.Println()

	// Basic copy
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, len(src))
	n := copy(dst, src)
	fmt.Printf("Copied %d elements\n", n)
	fmt.Printf("src: %v\n", src)
	fmt.Printf("dst: %v\n", dst)

	// Prove independence: modifying dst does NOT affect src
	dst[0] = 9999
	fmt.Printf("After dst[0]=9999: src=%v, dst=%v\n", src, dst)
	fmt.Println()

	// copy only copies min(len(dst), len(src)) elements
	small := make([]int, 3)       // dst has room for 3
	n2 := copy(small, src)        // src has 5 — only 3 copied
	fmt.Printf("Copy into smaller: copied=%d, small=%v\n", n2, small)

	large := make([]int, 10)      // dst has room for 10
	n3 := copy(large, src)        // src has 5 — only 5 copied
	fmt.Printf("Copy into larger: copied=%d, large=%v\n", n3, large)
	// large[5:] remains zero-initialized

	fmt.Println()

	// Copy subslice
	middle := make([]int, 3)
	copy(middle, src[1:4]) // copy src[1], src[2], src[3]
	fmt.Printf("Copy subslice src[1:4]: %v\n", middle)

	// Correct pattern to clone a slice
	original := []int{10, 20, 30}
	// WRONG: clone := original  → shares backing array
	// RIGHT: allocate + copy
	clone := make([]int, len(original))
	copy(clone, original)
	clone[0] = 999
	fmt.Printf("original: %v (unchanged)\n", original)
	fmt.Printf("clone:    %v\n", clone)

	// Shorthand clone using append:
	// clone2 := append([]int(nil), original...)
	// This works but allocates a new backing array via append growth.
	// The make+copy pattern is clearer and avoids over-allocation.
	clone2 := append([]int(nil), original...)
	fmt.Printf("clone2 via append: %v\n", clone2)

	// copy works with []byte and string (special case):
	// copy([]byte, string) without explicit conversion
	bs := make([]byte, 5)
	copy(bs, "Hello")
	fmt.Printf("copy string → []byte: %s\n", bs)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Three-index slicing [low:high:max]
// ─────────────────────────────────────────────────────────────────────────────
//
// Normal slice expression:  s[low:high]        → cap = cap(s) - low
// Full slice expression:    s[low:high:max]    → cap = max - low
//
// The third index (max) limits the capacity of the resulting slice.
// This is crucial for SAFETY: it prevents the sub-slice from accidentally
// growing into (and overwriting) elements beyond 'high' in the backing array.
//
// Use case: hand a sub-slice to untrusted code; limit what it can see.
//
// Constraints: low <= high <= max <= cap(s)

func section4ThreeIndexSlice() {
	fmt.Println("=== SECTION 4: Three-Index Slicing [low:high:max] ===")
	fmt.Println()

	backing := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	fmt.Printf("backing: %v  cap=%d\n", backing, cap(backing))
	fmt.Println()

	// Normal two-index slice:
	// cap extends to end of backing array
	twoIndex := backing[2:5]
	fmt.Printf("twoIndex=backing[2:5]:     %v  len=%d  cap=%d\n",
		twoIndex, len(twoIndex), cap(twoIndex))
	// cap = 10-2 = 8 (can "see" elements 2-9 via cap)

	// Three-index slice:
	// cap is explicitly limited to max-low = 5-2 = 3
	threeIndex := backing[2:5:5]
	fmt.Printf("threeIndex=backing[2:5:5]: %v  len=%d  cap=%d\n",
		threeIndex, len(threeIndex), cap(threeIndex))
	// cap = 5-2 = 3 (CANNOT see beyond index 4 in backing)

	fmt.Println()

	// SAFETY: appending to threeIndex beyond cap forces NEW backing array
	// It cannot overwrite backing[5]
	extended := append(threeIndex, 99)
	fmt.Printf("append(threeIndex, 99): %v\n", extended)
	fmt.Printf("backing after append:   %v (unchanged!)\n", backing)
	// With twoIndex, the append WOULD have overwritten backing[5]!

	fmt.Println()

	// Demonstrating the danger WITHOUT three-index slicing:
	backup := make([]int, len(backing))
	copy(backup, backing)

	dangerSlice := backing[2:5] // cap=8, can reach backing[9]
	dangerSlice = append(dangerSlice, 99) // writes to backing[5]!
	fmt.Printf("Danger: backing[5] after append to twoIndex sub-slice: %d\n",
		backing[5]) // was 5, now 99!

	// Restore
	copy(backing, backup)

	fmt.Println()

	// Another use: when passing sub-slice to a function,
	// limit its capacity so it can't grow into adjacent data
	data := make([]byte, 0, 100)
	data = append(data, []byte("Header|Payload|Footer")...)

	// Give function only the Payload portion, with cap limited to just that
	// so function can't overwrite Footer
	payload := data[7:14:14] // [7:14] with cap capped at 14
	fmt.Printf("Payload: %s (cap=%d)\n", payload, cap(payload))

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Prepend to a slice
// ─────────────────────────────────────────────────────────────────────────────
//
// Go has no built-in prepend. The pattern is:
//   s = append([]T{value}, s...)
//   OR
//   s = append(s[:1], s...)  // trick using copy
//
// Prepend is O(n) — every element must be shifted right.
// If you frequently prepend, consider a different data structure.

func section5Prepend() {
	fmt.Println("=== SECTION 5: Prepend ===")
	fmt.Println()

	s := []int{2, 3, 4, 5}

	// Method 1: create new 1-element slice, append rest
	s = append([]int{1}, s...)
	fmt.Printf("After prepend 1: %v\n", s)

	// Method 2: grow by 1, shift right, set first element
	// More efficient when you want to avoid the intermediate slice allocation
	s = append(s, 0) // grow by 1
	copy(s[1:], s)   // shift everything right
	s[0] = 0         // set first element
	fmt.Printf("After prepend 0: %v\n", s)

	// Prepend multiple elements
	prefix := []int{-2, -1}
	s = append(prefix, s...)
	fmt.Printf("After prepend {-2,-1}: %v\n", s)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Delete from a slice (by index)
// ─────────────────────────────────────────────────────────────────────────────
//
// Go has no built-in delete. Common patterns:
//
// 1. Order-preserving delete: copy elements left (O(n))
// 2. Order-NON-preserving delete: swap with last, truncate (O(1))
//
// Go 1.21 added slices.Delete from the "slices" package.

func section6Delete() {
	fmt.Println("=== SECTION 6: Delete from Slice ===")
	fmt.Println()

	// ORDER-PRESERVING delete: append the parts before and after the element
	s := []int{10, 20, 30, 40, 50}
	i := 2 // delete index 2 (value 30)

	// Method: append s[:i] with s[i+1:]
	// This shifts s[i+1:] left by 1 (writes over the deleted element)
	result := append(s[:i], s[i+1:]...)
	fmt.Printf("Order-preserving delete [2]: %v\n", result)
	// WARNING: s is now also affected (they share backing array)!
	// After delete, s[:i] is s[:2] = [10,20]
	// The append shifts [40,50] to positions 2 and 3
	// s's backing array is now [10,20,40,50,50] but s reports [10,20,40,50,50]
	// This is the "slice sharing" gotcha — see 04_slice_gotchas.go
	// SAFE approach: make a copy first

	s2 := []int{10, 20, 30, 40, 50}
	i = 2
	// Safe order-preserving delete (avoids mutating original)
	safe := make([]int, 0, len(s2)-1)
	safe = append(safe, s2[:i]...)
	safe = append(safe, s2[i+1:]...)
	fmt.Printf("Safe order-preserving delete: %v\n", safe)
	fmt.Printf("s2 unchanged: %v\n", s2)

	fmt.Println()

	// ORDER-SWAPPING delete: swap index i with last, truncate by 1
	// O(1) but doesn't preserve order
	s3 := []int{10, 20, 30, 40, 50}
	j := 2 // delete index 2 (value 30)
	s3[j] = s3[len(s3)-1]   // overwrite with last element
	s3 = s3[:len(s3)-1]      // shrink by 1
	fmt.Printf("O(1) swap-delete [2]: %v\n", s3)
	// [10, 20, 50, 40] — 50 moved to position 2

	fmt.Println()

	// Delete a RANGE of elements [i:j]
	s4 := []int{1, 2, 3, 4, 5, 6, 7, 8}
	from, to := 2, 5 // delete indices 2,3,4 (values 3,4,5)
	s4 = append(s4[:from], s4[to:]...)
	fmt.Printf("Delete range [2:5]: %v\n", s4) // [1 2 6 7 8]

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Insert at index
// ─────────────────────────────────────────────────────────────────────────────
//
// Go has no built-in insert. Pattern uses append + copy.
// Go 1.21 added slices.Insert from the "slices" package.

func section7Insert() {
	fmt.Println("=== SECTION 7: Insert at Index ===")
	fmt.Println()

	// Insert value at index i
	s := []int{1, 2, 4, 5}
	i := 2   // insert before index 2
	val := 3 // value to insert

	// Method: grow by 1, shift right from i, set s[i]
	s = append(s, 0)           // extend len by 1 (make room)
	copy(s[i+1:], s[i:])       // shift s[i:] one position right
	s[i] = val                 // place new value
	fmt.Printf("After insert %d at [%d]: %v\n", val, i, s) // [1 2 3 4 5]

	fmt.Println()

	// Alternative using append (more readable but creates a temp slice)
	s2 := []int{1, 2, 4, 5}
	i2 := 2
	val2 := 3
	// Append tail to a new slice containing just the new element,
	// then prepend head using a second append
	s2 = append(s2[:i2], append([]int{val2}, s2[i2:]...)...)
	fmt.Printf("Append-style insert: %v\n", s2)
	// NOTE: creates a temporary allocation; the copy method above is better

	// Insert multiple elements
	s3 := []int{1, 2, 6, 7}
	insertAt := 2
	newElems := []int{3, 4, 5}

	s3 = append(s3, newElems...) // grow
	copy(s3[insertAt+len(newElems):], s3[insertAt:]) // shift right
	copy(s3[insertAt:], newElems) // insert
	fmt.Printf("Insert multiple at [%d]: %v\n", insertAt, s3) // [1 2 3 4 5 6 7]

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 8: Other useful slice operations
// ─────────────────────────────────────────────────────────────────────────────

func section8OtherOperations() {
	fmt.Println("=== SECTION 8: Other Useful Operations ===")
	fmt.Println()

	// REVERSE a slice in place
	s := []int{1, 2, 3, 4, 5}
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i] // swap
	}
	fmt.Printf("Reversed: %v\n", s)

	// FILTER: keep only elements matching a predicate
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	evens := nums[:0] // reuse backing array — start with empty slice
	for _, n := range nums {
		if n%2 == 0 {
			evens = append(evens, n)
		}
	}
	fmt.Printf("Evens (in-place filter): %v\n", evens)
	// This modifies the backing array! Use a new slice for safe version:
	safeEvens := make([]int, 0)
	for _, n := range nums {
		if n%2 == 0 {
			safeEvens = append(safeEvens, n)
		}
	}
	fmt.Printf("Evens (safe filter): %v\n", safeEvens)

	// MAP (transform): apply function to every element
	doubled := make([]int, len(nums))
	for i, n := range nums {
		doubled[i] = n * 2
	}
	fmt.Printf("Doubled: %v\n", doubled)

	// CONTAINS check (linear scan)
	contains := func(s []int, target int) bool {
		for _, v := range s {
			if v == target {
				return true
			}
		}
		return false
	}
	fmt.Printf("Contains 7: %v\n", contains(nums, 7))
	fmt.Printf("Contains 11: %v\n", contains(nums, 11))

	// REDUCE: sum, product, etc.
	sum := 0
	for _, n := range nums {
		sum += n
	}
	fmt.Printf("Sum: %d\n", sum)

	// TRUNCATE: shrink without releasing memory
	// Setting len to 0 clears the slice but keeps backing array
	s2 := make([]int, 0, 100)
	for i := 0; i < 10; i++ {
		s2 = append(s2, i)
	}
	fmt.Printf("Before truncate: len=%d cap=%d\n", len(s2), cap(s2))
	s2 = s2[:0] // len=0, cap=100 — backing array still allocated
	fmt.Printf("After s2[:0]:    len=%d cap=%d\n", len(s2), cap(s2))
	// Useful for reusing a buffer: clear and refill without re-allocating

	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║      Slice Operations: The Complete Toolkit          ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	section1AppendGrowth()
	section2AppendVariants()
	section3Copy()
	section4ThreeIndexSlice()
	section5Prepend()
	section6Delete()
	section7Insert()
	section8OtherOperations()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║  KEY OPERATIONS CHEAT SHEET                         ║")
	fmt.Println("║  append(s, x)          add element                   ║")
	fmt.Println("║  append(a, b...)       append slice to slice         ║")
	fmt.Println("║  copy(dst, src)        independent copy              ║")
	fmt.Println("║  s[lo:hi:max]          limit cap of sub-slice        ║")
	fmt.Println("║  append([]T{x}, s...)  prepend                       ║")
	fmt.Println("║  append(s[:i],s[i+1:]) delete at i (order-preserving)║")
	fmt.Println("║  s[i]=s[last];s=s[:last] delete at i (O(1))         ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
}
