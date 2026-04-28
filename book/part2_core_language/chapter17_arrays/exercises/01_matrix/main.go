// EXERCISE 17.1 — Matrix operations on [N][N]int arrays.
//
// Implement transpose, trace, and multiply for 3×3 integer matrices.
// Because arrays have value semantics, functions receive and return copies.
//
// Run (from the chapter folder):
//   go run ./exercises/01_matrix

package main

import "fmt"

type Mat3 [3][3]int

// transpose returns the transpose of m (rows become columns).
func transpose(m Mat3) Mat3 {
	var t Mat3
	for i := range m {
		for j := range m[i] {
			t[j][i] = m[i][j]
		}
	}
	return t
}

// trace returns the sum of the diagonal elements.
func trace(m Mat3) int {
	return m[0][0] + m[1][1] + m[2][2]
}

// multiply returns the matrix product a × b.
func multiply(a, b Mat3) Mat3 {
	var c Mat3
	for i := range a {
		for j := range b[0] {
			for k := range b {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}

func printMat(label string, m Mat3) {
	fmt.Println(label)
	for _, row := range m {
		fmt.Println(" ", row)
	}
}

func main() {
	m := Mat3{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	printMat("m:", m)
	printMat("transpose(m):", transpose(m))
	fmt.Println("trace(m):", trace(m)) // 1+5+9 = 15

	fmt.Println()

	identity := Mat3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	printMat("m × identity:", multiply(m, identity))

	a := Mat3{{1, 2, 0}, {0, 3, 0}, {0, 0, 1}}
	b := Mat3{{2, 0, 0}, {1, 3, 0}, {0, 0, 1}}
	printMat("a × b:", multiply(a, b))
}
