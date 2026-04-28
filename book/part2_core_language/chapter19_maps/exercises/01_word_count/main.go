// EXERCISE 19.1 — Word count with ranking.
//
// Implement TopN(text string, n int) []WordCount that returns the n most
// frequent words in text (case-insensitive, punctuation stripped),
// sorted by frequency descending, then alphabetically for ties.
//
// Run (from the chapter folder):
//   go run ./exercises/01_word_count

package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type WordCount struct {
	Word  string
	Count int
}

// normalize strips leading/trailing punctuation and lowercases.
func normalize(w string) string {
	w = strings.TrimFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.ToLower(w)
}

// TopN returns the n most frequent words.
func TopN(text string, n int) []WordCount {
	freq := make(map[string]int)
	for _, raw := range strings.Fields(text) {
		w := normalize(raw)
		if w != "" {
			freq[w]++
		}
	}

	pairs := make([]WordCount, 0, len(freq))
	for w, c := range freq {
		pairs = append(pairs, WordCount{w, c})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Word < pairs[j].Word
	})

	if n > len(pairs) {
		n = len(pairs)
	}
	return pairs[:n]
}

func main() {
	text := `To be or not to be, that is the question.
Whether 'tis nobler in the mind to suffer
the slings and arrows of outrageous fortune,
or to take arms against a sea of troubles.`

	fmt.Println("Top 10 words:")
	for i, wc := range TopN(text, 10) {
		fmt.Printf("  %2d. %-12s %d\n", i+1, wc.Word, wc.Count)
	}

	fmt.Println()

	short := "go go go python python rust"
	fmt.Println("Top 2 of short text:", TopN(short, 2))
}
