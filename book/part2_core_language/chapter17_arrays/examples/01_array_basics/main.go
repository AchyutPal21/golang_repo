// FILE: book/part2_core_language/chapter17_arrays/examples/01_array_basics/main.go
// CHAPTER: 17 — Arrays: The Real Underlying Type
// TOPIC: Declaration, initialisation, length, indexing, iteration,
//        multi-dimensional arrays, array as comparable type.
//
// Run (from the chapter folder):
//   go run ./examples/01_array_basics

package main

import "fmt"

func main() {
	// --- Declaration and zero initialisation ---
	var a [5]int
	fmt.Println("zero array:", a)

	// --- Composite literal ---
	b := [5]int{10, 20, 30, 40, 50}
	fmt.Println("literal:", b)

	// --- Ellipsis: compiler counts elements ---
	c := [...]string{"alpha", "beta", "gamma"}
	fmt.Println("ellipsis:", c, "len:", len(c))

	// --- Sparse initialisation ---
	d := [10]int{0: 1, 5: 5, 9: 9}
	fmt.Println("sparse:", d)

	// --- Indexing and mutation ---
	b[2] = 300
	fmt.Println("after b[2]=300:", b)

	// --- len is compile-time constant for arrays ---
	fmt.Println("len(b):", len(b))

	// --- Iteration ---
	fmt.Print("range b: ")
	for i, v := range b {
		fmt.Printf("[%d]=%d ", i, v)
	}
	fmt.Println()

	// --- Multi-dimensional ---
	var matrix [3][3]int
	for i := range matrix {
		for j := range matrix[i] {
			matrix[i][j] = i*3 + j + 1
		}
	}
	fmt.Println("3×3 matrix:")
	for _, row := range matrix {
		fmt.Println(" ", row)
	}

	// --- Arrays are comparable ---
	x := [3]int{1, 2, 3}
	y := [3]int{1, 2, 3}
	z := [3]int{1, 2, 4}
	fmt.Println("x == y:", x == y) // true
	fmt.Println("x == z:", x == z) // false

	// Arrays of different lengths are different types — won't compile:
	// var p [3]int
	// var q [4]int
	// p == q // compile error: mismatched types
}
