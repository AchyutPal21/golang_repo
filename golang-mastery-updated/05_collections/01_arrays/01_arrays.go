// 01_arrays.go
// Topic: Arrays in Go — fixed-size value types
//
// Arrays are the foundation of all collection types in Go, but they are
// surprisingly rare in everyday application code. Understanding WHY they
// exist and WHEN to use them vs slices is the key insight this file builds.
//
// Run: go run 05_collections/01_arrays.go

package main

import (
	"crypto/sha256"
	"fmt"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: What is an array?
// ─────────────────────────────────────────────────────────────────────────────
//
// An array is a FIXED-SIZE, CONTIGUOUS block of memory that holds elements
// of a SINGLE type. The size is part of the TYPE itself:
//
//   [3]int  ≠  [4]int   ← these are two DIFFERENT types
//
// This is the most important property: [3]int and [4]int are incompatible types.
// You cannot assign one to the other, pass one where the other is expected, etc.
//
// Memory layout for [3]int (each int = 8 bytes on 64-bit):
//
//   address:  0x00  0x08  0x10
//   value:    [  0 |  0  |  0 ]
//
// All elements are ZERO-INITIALIZED automatically. No garbage values.

func section1ArrayBasics() {
	fmt.Println("=== SECTION 1: Array Basics ===")

	// Zero-initialized by default
	var nums [5]int
	fmt.Printf("Zero-initialized array: %v\n", nums) // [0 0 0 0 0]

	// The length is PART of the type. len() is a compile-time constant here.
	fmt.Printf("Length: %d\n", len(nums)) // 5

	// Accessing elements (0-indexed, like all Go collections)
	nums[0] = 10
	nums[4] = 50
	fmt.Printf("After assignment: %v\n", nums) // [10 0 0 0 50]

	// Accessing out of bounds: PANIC at runtime (or compile error if constant index)
	// nums[5] = 99 // ← compile error: index 5 out of bounds [5]
	// i := 5; nums[i] = 99 // ← runtime panic: index out of range [5] with length 5

	// Arrays of different types
	var flags [3]bool
	var names [4]string
	fmt.Printf("bool array: %v\n", flags)  // [false false false]
	fmt.Printf("string array: %v\n", names) // [  (all empty strings)  ]

	// TYPE INCOMPATIBILITY: this is the crucial Go rule
	var a3 [3]int
	var a4 [4]int
	_ = a3
	_ = a4
	// a3 = a4  // ← compile error: cannot use a4 (type [4]int) as type [3]int
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Array literal syntax
// ─────────────────────────────────────────────────────────────────────────────
//
// Go offers several syntaxes for initializing arrays with values.

func section2ArrayLiterals() {
	fmt.Println("=== SECTION 2: Array Literals ===")

	// Full literal: list every element
	primes := [6]int{2, 3, 5, 7, 11, 13}
	fmt.Printf("Primes: %v\n", primes)

	// Ellipsis (...): compiler COUNTS the elements for you
	// Same as [5]string{"a","b","c","d","e"}
	letters := [...]string{"a", "b", "c", "d", "e"}
	fmt.Printf("Letters: %v  (type: %T)\n", letters, letters)
	// Note: the type is still [5]string, not [...] — compiler resolves it

	// Sparse initialization: specify only certain indices
	// All unspecified elements are zero-valued
	sparse := [10]int{0: 100, 5: 500, 9: 900}
	fmt.Printf("Sparse: %v\n", sparse)
	// Output: [100 0 0 0 0 500 0 0 0 900]

	// Mix of positional and indexed
	mixed := [5]string{1: "one", 3: "three"}
	fmt.Printf("Mixed: %v\n", mixed)
	// Output: [ one  three ]  (index 0,2,4 are empty string "")

	// 2D array literal
	matrix := [2][3]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	fmt.Printf("Matrix: %v\n", matrix)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Multi-dimensional arrays
// ─────────────────────────────────────────────────────────────────────────────
//
// Multi-dimensional arrays are arrays of arrays. Each "row" is stored
// contiguously in memory (row-major order, like C).
//
// Memory layout for [2][3]int:
//
//   [row0_col0][row0_col1][row0_col2][row1_col0][row1_col1][row1_col2]
//    8 bytes    8 bytes    8 bytes    8 bytes    8 bytes    8 bytes
//
// Total: 48 bytes, all contiguous. Very cache-friendly!

func section3MultiDimensional() {
	fmt.Println("=== SECTION 3: Multi-Dimensional Arrays ===")

	// Declare and initialize a 3×3 grid
	var grid [3][3]int
	for i := range grid {
		for j := range grid[i] {
			grid[i][j] = i*3 + j + 1
		}
	}

	fmt.Println("3x3 grid:")
	for _, row := range grid {
		fmt.Printf("  %v\n", row)
	}

	// Identity matrix
	var identity [3][3]float64
	for i := range identity {
		identity[i][i] = 1.0
	}
	fmt.Println("3x3 identity matrix:")
	for _, row := range identity {
		fmt.Printf("  %v\n", row)
	}

	// 3D array
	var cube [2][2][2]int
	cube[0][0][0] = 1
	cube[1][1][1] = 8
	fmt.Printf("3D cube: %v\n", cube)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Array comparison with ==
// ─────────────────────────────────────────────────────────────────────────────
//
// Arrays in Go ARE comparable with == if their element type is comparable.
// Two arrays are equal if ALL corresponding elements are equal.
//
// This is DIFFERENT from many languages (C, Java) where array == compares
// pointer/reference identity, not element values.
//
// Slices, by contrast, CANNOT be compared with == (except against nil).
// This is one of the rare cases where arrays have an ADVANTAGE over slices.

func section4ArrayComparison() {
	fmt.Println("=== SECTION 4: Array Comparison ===")

	a := [3]int{1, 2, 3}
	b := [3]int{1, 2, 3}
	c := [3]int{1, 2, 4}

	fmt.Printf("a == b: %v\n", a == b) // true  (element-wise comparison)
	fmt.Printf("a == c: %v\n", a == c) // false (last element differs)
	fmt.Printf("a != c: %v\n", a != c) // true

	// Arrays can be used as MAP KEYS because they're comparable!
	// (Slices cannot be map keys — they're not comparable)
	visitCount := map[[2]int]int{}
	visitCount[[2]int{0, 0}]++
	visitCount[[2]int{1, 2}]++
	visitCount[[2]int{0, 0}]++
	fmt.Printf("Visit counts: %v\n", visitCount) // map[(0,0):2  (1,2):1]

	// Practical: use [4]byte as a map key for IP addresses
	type IPv4 [4]byte
	routes := map[IPv4]string{
		{127, 0, 0, 1}:   "localhost",
		{192, 168, 1, 1}: "router",
	}
	fmt.Printf("IP routes: %v\n", routes)

	// String arrays — only if element type is comparable
	s1 := [2]string{"hello", "world"}
	s2 := [2]string{"hello", "world"}
	fmt.Printf("String array comparison: %v\n", s1 == s2) // true

	// NOTE: arrays of slices would NOT be comparable:
	// var x [2][]int  // cannot use in == because []int is not comparable

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Arrays are VALUE types (copied on assignment & function call)
// ─────────────────────────────────────────────────────────────────────────────
//
// THIS IS THE MOST IMPORTANT AND MOST SURPRISING THING ABOUT GO ARRAYS.
//
// In Go, arrays are VALUE types. When you assign an array to a new variable
// or pass it to a function, the ENTIRE array is COPIED.
//
// Compare with slices: slices pass a small 3-word header, not the data.
// Compare with C/Java: arrays pass a pointer/reference.
//
// Consequence:
//   - Modifying the copy does NOT affect the original.
//   - For large arrays, this can be expensive (copying 1MB of data per call).
//   - To mutate the original, you MUST pass a pointer.

// modifyArrayValue receives a COPY of the array
// Any changes here have NO effect on the caller's array
func modifyArrayValue(arr [5]int) {
	arr[0] = 9999 // modifies LOCAL copy only
	fmt.Printf("  Inside modifyArrayValue: arr[0] = %d\n", arr[0])
}

// modifyArrayPointer receives a POINTER to the original array
// Changes here DO affect the caller's array
func modifyArrayPointer(arr *[5]int) {
	arr[0] = 9999 // modifies the ORIGINAL
	fmt.Printf("  Inside modifyArrayPointer: arr[0] = %d\n", arr[0])
}

func section5ValueSemantics() {
	fmt.Println("=== SECTION 5: Arrays are Value Types (Copied!) ===")

	original := [5]int{1, 2, 3, 4, 5}
	fmt.Printf("Original before: %v\n", original)

	// Assignment copies the array
	copyOfArray := original
	copyOfArray[0] = 100
	fmt.Printf("Original after modifying copy: %v\n", original)  // unchanged!
	fmt.Printf("Copy: %v\n", copyOfArray)

	// Function call copies the array
	fmt.Printf("Before function call: original[0] = %d\n", original[0])
	modifyArrayValue(original)
	fmt.Printf("After modifyArrayValue: original[0] = %d (unchanged!)\n", original[0])

	// Using a pointer to mutate the original
	modifyArrayPointer(&original)
	fmt.Printf("After modifyArrayPointer: original[0] = %d (changed!)\n", original[0])

	// Cost of copying: for large arrays this matters
	// [1024*1024]byte = 1 MB — passing by value would copy 1MB per call
	// Always pass large arrays by pointer (or better: use a slice)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Iterating over arrays
// ─────────────────────────────────────────────────────────────────────────────

func section6Iteration() {
	fmt.Println("=== SECTION 6: Iterating Arrays ===")

	fruits := [4]string{"apple", "banana", "cherry", "date"}

	// for loop with index
	fmt.Println("Index loop:")
	for i := 0; i < len(fruits); i++ {
		fmt.Printf("  fruits[%d] = %s\n", i, fruits[i])
	}

	// for-range loop: preferred Go style
	// range returns (index, value) — value is a COPY of the element
	fmt.Println("Range loop:")
	for i, fruit := range fruits {
		fmt.Printf("  [%d] %s\n", i, fruit)
	}

	// range with only index (discard value)
	fmt.Println("Range index only:")
	for i := range fruits {
		fmt.Printf("  index %d\n", i)
	}

	// range with only value (discard index with _)
	fmt.Println("Range values only:")
	for _, fruit := range fruits {
		fmt.Printf("  %s\n", fruit)
	}

	// GOTCHA: modifying the range variable does NOT modify the array
	// because 'fruit' is a COPY of the element
	nums := [3]int{10, 20, 30}
	for _, v := range nums {
		v *= 2 // does NOT modify nums
		_ = v
	}
	fmt.Printf("After range-modify: %v (unchanged!)\n", nums)

	// To modify, use the index
	for i := range nums {
		nums[i] *= 2 // DOES modify nums
	}
	fmt.Printf("After index-modify: %v\n", nums) // [20 40 60]

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Why arrays are RARELY used directly — slices are preferred
// ─────────────────────────────────────────────────────────────────────────────
//
// In most Go code, you'll see []int instead of [n]int. Why?
//
// 1. DYNAMIC SIZE: Slices can grow. Arrays cannot.
//    Real data (HTTP requests, user inputs, DB rows) has variable length.
//
// 2. FUNCTION SIGNATURES: [5]int only accepts exactly 5 elements.
//    A function taking []int works with any slice length.
//
// 3. IDIOMATIC APIs: All standard library functions use slices.
//    sort.Ints([]int), append, copy, etc. — all work on slices.
//
// 4. INTEROPERABILITY: You can take a slice OF an array: arr[:]
//    But you cannot take an array of a slice.
//
// However, there ARE cases where arrays are the RIGHT choice...

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 8: When arrays ARE the right choice
// ─────────────────────────────────────────────────────────────────────────────

// USE CASE 1: Cryptographic hashes have FIXED, known sizes.
// sha256.Sum256 returns [32]byte — NOT []byte.
// This enforces at compile time that you can't accidentally pass
// a wrong-length buffer to a function expecting exactly 32 bytes.
func section8WhenArraysAreRight() {
	fmt.Println("=== SECTION 8: When Arrays ARE the Right Choice ===")

	// Cryptographic hashes: the size IS part of the type contract
	data := []byte("hello, Go")
	hash := sha256.Sum256(data) // returns [32]byte, not []byte
	fmt.Printf("SHA256 hash type: %T\n", hash)
	fmt.Printf("SHA256 hash: %x\n", hash)
	fmt.Printf("Hash length guaranteed at compile time: %d bytes\n", len(hash))

	// Compare two hashes safely with ==
	hash2 := sha256.Sum256(data)
	hash3 := sha256.Sum256([]byte("different"))
	fmt.Printf("Same data → equal hashes: %v\n", hash == hash2)
	fmt.Printf("Diff data → equal hashes: %v\n", hash == hash3)

	// USE CASE 2: Fixed-size I/O buffers
	// When reading from a network or file with a known packet size,
	// a fixed array avoids heap allocation.
	var fixedBuffer [512]byte // on the stack, no heap allocation
	_ = fixedBuffer

	// USE CASE 3: Small, fixed-size lookup tables
	// Days of week, months, chess board (8x8), etc.
	var chessBoard [8][8]byte
	// Initialize with chess notation
	backRank := [8]byte{'R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'}
	chessBoard[0] = backRank
	for i := range chessBoard[1] {
		chessBoard[1][i] = 'P'
	}
	fmt.Println("Chess board (first 3 rows):")
	for i := 0; i < 3; i++ {
		fmt.Printf("  %s\n", chessBoard[i])
	}

	// USE CASE 4: [N]byte for fixed-size identifiers, MAC addresses, etc.
	type MACAddress [6]byte
	mac := MACAddress{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}
	fmt.Printf("MAC: %02X:%02X:%02X:%02X:%02X:%02X\n",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])

	// The MAC type is comparable (can be used as map key),
	// takes EXACTLY 6 bytes, and the constraint is enforced at compile time.

	// USE CASE 5: Stack allocation optimization
	// Small arrays (few hundred bytes) can live on the stack, not heap.
	// This means: no garbage collection pressure, faster allocation.
	// Slices always point to heap-allocated backing arrays (in most cases).

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 9: Converting between arrays and slices
// ─────────────────────────────────────────────────────────────────────────────

func section9ArraySliceConversion() {
	fmt.Println("=== SECTION 9: Array ↔ Slice Conversion ===")

	arr := [5]int{10, 20, 30, 40, 50}

	// arr[:] creates a slice that REFERENCES the array (no copy!)
	// The slice header points to arr's memory.
	sliceOfArr := arr[:]
	fmt.Printf("Array:  %v\n", arr)
	fmt.Printf("Slice:  %v (len=%d, cap=%d)\n", sliceOfArr, len(sliceOfArr), cap(sliceOfArr))

	// Modifying through the slice MODIFIES the original array
	sliceOfArr[0] = 999
	fmt.Printf("After slice[0]=999, array: %v\n", arr) // [999 20 30 40 50]

	// Sub-slices of array
	middle := arr[1:4]
	fmt.Printf("arr[1:4]: %v\n", middle) // [20 30 40]

	// Go 1.17+: you can also use &arr[0] to get a pointer to the first element
	// and pass that to C functions (unsafe territory, rarely needed)

	// Converting []T to [N]T (Go 1.20+)
	// s := []int{1, 2, 3, 4, 5}
	// arr2 := [5]int(s)  // creates a copy (requires len(s) >= 5)
	// fmt.Printf("Slice to array: %v\n", arr2)

	// Go 1.17+: slice to array pointer (zero-copy, shares memory)
	arrPtr := (*[3]int)(sliceOfArr) // sliceOfArr must have len >= 3
	arrPtr[0] = 777
	fmt.Printf("After arrPtr[0]=777, original array: %v\n", arr)
	// Shows that arrPtr shares memory with arr

	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║           Go Arrays: Deep Dive                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	section1ArrayBasics()
	section2ArrayLiterals()
	section3MultiDimensional()
	section4ArrayComparison()
	section5ValueSemantics()
	section6Iteration()
	section8WhenArraysAreRight()
	section9ArraySliceConversion()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║  KEY TAKEAWAYS                                       ║")
	fmt.Println("║  1. Size is PART of the type: [3]int ≠ [4]int       ║")
	fmt.Println("║  2. Arrays are VALUE types — assignment COPIES them  ║")
	fmt.Println("║  3. Zero-initialized by default (no garbage values)  ║")
	fmt.Println("║  4. Arrays are COMPARABLE with == (unlike slices)    ║")
	fmt.Println("║  5. Use arrays for fixed-size data: hashes, buffers  ║")
	fmt.Println("║  6. In most code, use SLICES instead of arrays       ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
}
