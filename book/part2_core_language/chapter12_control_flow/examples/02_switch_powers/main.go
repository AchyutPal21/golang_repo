// FILE: book/part2_core_language/chapter12_control_flow/examples/02_switch_powers/main.go
// CHAPTER: 12 — Control Flow
// TOPIC: switch features: tagless, multi-value case, fallthrough, init.
//
// Run (from the chapter folder):
//   go run ./examples/02_switch_powers

package main

import "fmt"

func size(n int) string {
	// Tagless switch: each case is a bool expression. Cleaner than
	// chained if-else when there are 3+ branches.
	switch {
	case n < 0:
		return "negative"
	case n == 0:
		return "zero"
	case n < 10:
		return "small"
	case n < 100:
		return "medium"
	default:
		return "large"
	}
}

func vowel(r rune) bool {
	// Multi-value case: one case matches several values.
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func describeRune(r rune) string {
	// Switch with initializer + tagless: compute once, branch many.
	switch n := int(r); {
	case n < 32:
		return "control"
	case n < 127:
		return "ascii"
	default:
		return "non-ascii"
	}
}

func levelMessage(level int) string {
	// Explicit fallthrough — rarely useful, but here's the form.
	// Note: each case can ONLY fall through to the next; you can't skip.
	switch level {
	case 0:
		return "ok"
	case 1:
		fmt.Println("notice: warning level")
		fallthrough
	case 2:
		return "warning"
	case 3:
		return "critical"
	default:
		return "unknown"
	}
}

func main() {
	for _, n := range []int{-1, 0, 5, 42, 999} {
		fmt.Printf("size(%4d) = %s\n", n, size(n))
	}

	fmt.Println()
	for _, r := range []rune{'a', 'b', 'e', 'z'} {
		fmt.Printf("vowel(%q)  = %v\n", r, vowel(r))
	}

	fmt.Println()
	for _, r := range []rune{'\t', 'A', '世'} {
		fmt.Printf("describeRune(%q) = %s\n", r, describeRune(r))
	}

	fmt.Println()
	for _, lvl := range []int{0, 1, 2, 3, 99} {
		fmt.Printf("levelMessage(%d) = %s\n", lvl, levelMessage(lvl))
	}
}
