// EXERCISE 13.1 — Build a string-processing pipeline.
//
// Implement pipeline() which takes a string and a variadic list of
// Transformers, applies them left-to-right, and returns the final result.
//
// Then build a Caesar cipher using two function factories:
//   - shiftBy(n int) Transformer — shifts each letter by n positions
//   - onlyLetters() Transformer  — strips non-letter characters
//
// Run (from the chapter folder):
//   go run ./exercises/01_pipeline

package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Transformer func(string) string

// pipeline applies each transformer in order.
func pipeline(s string, steps ...Transformer) string {
	for _, step := range steps {
		s = step(s)
	}
	return s
}

// shiftBy returns a Transformer that Caesar-shifts every letter by n.
func shiftBy(n int) Transformer {
	n = ((n % 26) + 26) % 26 // normalise to [0,25]
	return func(s string) string {
		var b strings.Builder
		for _, r := range s {
			switch {
			case r >= 'a' && r <= 'z':
				b.WriteRune('a' + (r-'a'+rune(n))%26)
			case r >= 'A' && r <= 'Z':
				b.WriteRune('A' + (r-'A'+rune(n))%26)
			default:
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}

// onlyLetters strips every character that is not a Unicode letter.
func onlyLetters() Transformer {
	return func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if unicode.IsLetter(r) {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}

func main() {
	// Basic pipeline
	result := pipeline(
		"  Hello, World!  ",
		strings.TrimSpace,
		strings.ToLower,
		func(s string) string { return "[" + s + "]" },
	)
	fmt.Println("pipeline result:", result)

	fmt.Println()

	// Caesar cipher: strip punctuation, shift by 13 (ROT-13)
	plaintext := "Attack at dawn!"
	encoded := pipeline(plaintext, onlyLetters(), shiftBy(13))
	decoded := pipeline(encoded, shiftBy(-13))

	fmt.Println("plaintext:", plaintext)
	fmt.Println("encoded:  ", encoded)
	fmt.Println("decoded:  ", decoded)

	fmt.Println()

	// Compose encode+decode in a single pipeline
	roundtrip := pipeline(plaintext, onlyLetters(), shiftBy(7), shiftBy(-7))
	fmt.Println("roundtrip:", roundtrip)
}
