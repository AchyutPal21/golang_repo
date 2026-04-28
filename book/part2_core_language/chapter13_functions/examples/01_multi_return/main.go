// FILE: book/part2_core_language/chapter13_functions/examples/01_multi_return/main.go
// CHAPTER: 13 — Functions: First-Class Citizens
// TOPIC: Multiple return values, named returns, naked return, error idiom.
//
// Run (from the chapter folder):
//   go run ./examples/01_multi_return

package main

import (
	"errors"
	"fmt"
	"strconv"
)

// divide returns a result and an error — Go's standard "value, error" idiom.
// The caller is forced to deal with the error path.
func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// minMax uses named return values as documentation. The names appear in
// generated godoc and make the signature self-describing.
func minMax(nums []int) (min, max int) {
	if len(nums) == 0 {
		return 0, 0
	}
	min, max = nums[0], nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return // naked return — only acceptable in short functions
}

// parseCoord parses "x,y" pairs and demonstrates multi-error-check chaining.
func parseCoord(xs, ys string) (x, y int, err error) {
	x, err = strconv.Atoi(xs)
	if err != nil {
		return // named return carries the partial state + error
	}
	y, err = strconv.Atoi(ys)
	return
}

// fileStats is a realistic example: a function returning three independent
// values. Callers can ignore values they don't need with _.
func fileStats(content string) (lines, words, bytes int) {
	bytes = len(content)
	inWord := false
	for _, ch := range content {
		switch ch {
		case '\n':
			lines++
			inWord = false
		case ' ', '\t', '\r':
			inWord = false
		default:
			if !inWord {
				words++
				inWord = true
			}
		}
	}
	return
}

func main() {
	// --- divide ---
	if result, err := divide(10, 3); err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("10 / 3 = %.4f\n", result)
	}

	if _, err := divide(5, 0); err != nil {
		fmt.Println("caught:", err)
	}

	fmt.Println()

	// --- minMax ---
	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	lo, hi := minMax(nums)
	fmt.Printf("minMax(%v) → min=%d  max=%d\n", nums, lo, hi)

	fmt.Println()

	// --- parseCoord ---
	if x, y, err := parseCoord("12", "34"); err != nil {
		fmt.Println("parse error:", err)
	} else {
		fmt.Printf("coord = (%d, %d)\n", x, y)
	}

	if _, _, err := parseCoord("12", "bad"); err != nil {
		fmt.Println("parse error:", err)
	}

	fmt.Println()

	// --- fileStats ---
	text := "hello world\nfoo bar baz\n"
	l, w, b := fileStats(text)
	fmt.Printf("lines=%d  words=%d  bytes=%d\n", l, w, b)

	// Caller discards what it doesn't need.
	_, wordCount, _ := fileStats(text)
	fmt.Println("word count only:", wordCount)
}
