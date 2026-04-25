// FILE: book/part1_foundations/chapter06_coming_from_another_language/examples/02_python_to_go/main.go
// CHAPTER: 06 — Coming From Another Language
// TOPIC: A Python program translated to Go, side-by-side in comments.
//
// Run (from the chapter folder):
//   go run ./examples/02_python_to_go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The fastest way to feel the Python → Go translation is to read both
//   versions side by side. Each section below shows the Python original (in
//   block comments) and its idiomatic Go translation (live code).
//
//   The program: count word frequencies in a hard-coded passage. In Python
//   you'd reach for collections.Counter and a list comprehension. In Go,
//   you write a map and a for loop, and it's about as much code.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"sort"
	"strings"
)

/*
PYTHON ORIGINAL:

    from collections import Counter
    from typing import List, Tuple

    SAMPLE = (
        "The quick brown fox jumps over the lazy dog. "
        "The dog was not amused. "
    )

    def normalize(s: str) -> List[str]:
        return [w.strip(".,").lower() for w in s.split() if w.strip(".,")]

    def top(words: List[str], n: int) -> List[Tuple[str, int]]:
        return Counter(words).most_common(n)

    if __name__ == "__main__":
        for word, count in top(normalize(SAMPLE), 5):
            print(f"{word}: {count}")
*/

// SAMPLE is the corpus. In Python this would be a multi-line concatenated
// string; in Go we use a raw string literal (backticks) to skip escaping.
const SAMPLE = `The quick brown fox jumps over the lazy dog.
The dog was not amused.`

// normalize splits the input into lowercase tokens with punctuation stripped.
//
// Python idioms it does NOT use:
//   - list comprehension (we use a for loop)
//   - generator expression (same)
//   - method chaining like .strip().lower() — we use the equivalent function
//     calls explicitly
func normalize(s string) []string {
	tokens := strings.Fields(s)              // split on whitespace
	out := make([]string, 0, len(tokens))    // Go pre-sizes; Python doesn't
	for _, t := range tokens {
		t = strings.Trim(t, ".,!?;:")
		t = strings.ToLower(t)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// pair is a (word, count) tuple. Python returns tuples directly; Go returns
// a slice of structs. Slightly more verbose; statically typed.
type pair struct {
	word  string
	count int
}

// top returns the n most-common words. Python's Counter.most_common does
// this in one call; we build the histogram with a map, sort by count, take
// the first n.
func top(words []string, n int) []pair {
	count := map[string]int{} // map literal — like a Python dict
	for _, w := range words {
		count[w]++
	}

	pairs := make([]pair, 0, len(count))
	for w, c := range count {
		pairs = append(pairs, pair{w, c})
	}

	// Sort descending by count, then alphabetically for ties (deterministic).
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].word < pairs[j].word
	})

	if n < len(pairs) {
		pairs = pairs[:n]
	}
	return pairs
}

func main() {
	for _, p := range top(normalize(SAMPLE), 5) {
		// fmt.Printf is roughly Python's f-string. The verbs are different:
		// %s for strings, %d for integers.
		fmt.Printf("%s: %d\n", p.word, p.count)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Things that DIDN'T translate from the Python:
	//   - collections.Counter — Go has no equivalent; we built the
	//     histogram manually with a map. About 8 extra lines.
	//   - the list comprehension in normalize() — Go has no
	//     comprehensions; we used a for loop with append. About 5 extra
	//     lines.
	//   - implicit return tuples — Go uses a struct (`pair`) for
	//     readability when there are multiple values. Slightly more
	//     verbose; the type-system payoff is real.
	//
	// Things that translated CLEANLY:
	//   - the overall structure (normalize → top → print)
	//   - string operations (split, lower, trim)
	//   - map/dict access
	//
	// The Go program is ~30 lines longer in total. The compile-time type
	// checking and ~50x runtime speed are the trades.
	// ─────────────────────────────────────────────────────────────────────
}
