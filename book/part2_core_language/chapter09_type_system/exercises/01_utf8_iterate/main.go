// FILE: book/part2_core_language/chapter09_type_system/exercises/01_utf8_iterate/main.go
// EXERCISE 9.2 — UTF-8 iteration.
//
// Run (from the chapter folder):
//   go run ./exercises/01_utf8_iterate
//
// Task: complete `runeAt(s, n)` to return the n-th RUNE (0-indexed) of s,
// and a boolean indicating whether the index was in range. Use the
// for-range pattern; do NOT use s[n], that returns bytes.

package main

import "fmt"

// runeAt returns the n-th rune of s and true if found; otherwise zero rune
// and false. n is 0-indexed.
//
// Implementation hint: range over s; track an index counter; when it
// matches n, return the rune. If the loop ends, the index was out of
// range.
func runeAt(s string, n int) (rune, bool) {
	i := 0
	for _, r := range s {
		if i == n {
			return r, true
		}
		i++
	}
	return 0, false
}

func main() {
	cases := []struct {
		s string
		n int
	}{
		{"hello", 0},
		{"hello", 4},
		{"hello", 5}, // out of range
		{"héllo", 1}, // should be 'é' — this is the test
		{"你好世界", 2},
		{"🎉🌍", 0},
	}

	for _, c := range cases {
		r, ok := runeAt(c.s, c.n)
		if ok {
			fmt.Printf("runeAt(%q, %d) = %q\n", c.s, c.n, r)
		} else {
			fmt.Printf("runeAt(%q, %d) = (out of range)\n", c.s, c.n)
		}
	}
}
