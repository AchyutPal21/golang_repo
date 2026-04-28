// FILE: book/part2_core_language/chapter12_control_flow/examples/03_labels_goto/main.go
// CHAPTER: 12 — Control Flow
// TOPIC: Labelled break/continue, plus a legitimate goto retry.
//
// Run (from the chapter folder):
//   go run ./examples/03_labels_goto

package main

import (
	"errors"
	"fmt"
	"time"
)

// findFirstNegative returns the (row, col) of the first negative element,
// using a labelled break to escape both loops at once.
func findFirstNegative(matrix [][]int) (row, col int, found bool) {
outer:
	for r, rowSlice := range matrix {
		for c, v := range rowSlice {
			if v < 0 {
				return r, c, true
			}
			_ = c
		}
		_ = r
		continue outer
	}
	return 0, 0, false
}

// --- Retry loop using goto ---
//
// goto is appropriate here because the retry decision is *after* the
// attempt's bookkeeping, and using a for-loop would require restructuring
// in a way that obscures the "we're going back to try again" intent.
// Reasonable people disagree; both forms are legitimate.

var attempts int

func unstable() error {
	attempts++
	if attempts < 3 {
		return errors.New("temporary")
	}
	return nil
}

func tryWithGoto() error {
	const maxRetries = 5
	tries := 0
retry:
	if err := unstable(); err != nil {
		tries++
		if tries >= maxRetries {
			return fmt.Errorf("exceeded retries: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
		goto retry
	}
	return nil
}

// Same intent, written without goto. Most reviewers will prefer this.
func tryWithFor() error {
	const maxRetries = 5
	for tries := 0; tries < maxRetries; tries++ {
		if err := unstable(); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("exceeded retries")
}

func main() {
	matrix := [][]int{
		{1, 2, 3},
		{4, -5, 6},
		{7, 8, 9},
	}
	r, c, ok := findFirstNegative(matrix)
	if ok {
		fmt.Printf("first negative at (%d, %d) = %d\n", r, c, matrix[r][c])
	}

	attempts = 0
	if err := tryWithGoto(); err != nil {
		fmt.Println("goto form:", err)
	} else {
		fmt.Println("goto form succeeded after", attempts, "attempts")
	}

	attempts = 0
	if err := tryWithFor(); err != nil {
		fmt.Println("for form:", err)
	} else {
		fmt.Println("for form succeeded after", attempts, "attempts")
	}
}
