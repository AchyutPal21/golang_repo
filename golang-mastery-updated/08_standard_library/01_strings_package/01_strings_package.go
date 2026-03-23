// 01_strings_package.go
//
// The strings package: Go's comprehensive toolkit for UTF-8 string manipulation.
//
// WHY A DEDICATED PACKAGE?
// In Go, strings are immutable byte slices encoded in UTF-8. Every "modification"
// actually creates a new string. The strings package provides safe, correct, and
// (mostly) efficient operations without you having to manually slice and dice bytes.
//
// KEY MENTAL MODEL:
// - strings package works on Go's immutable string type
// - For mutable/streaming work, use bytes.Buffer or strings.Builder
// - All functions are Unicode-aware (they handle multi-byte runes correctly)
// - Functions that return strings always allocate; Builder avoids repeated allocation

package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: Searching / Testing
// ─────────────────────────────────────────────────────────────────────────────

func searchingFunctions() {
	fmt.Println("═══ SECTION 1: Searching & Testing ═══")

	s := "Hello, Gophers! Go is great."

	// strings.Contains
	// WHY: Checks substring existence. Uses a fast byte-search algorithm internally.
	// Returns bool — straightforward and idiomatic Go.
	fmt.Println(strings.Contains(s, "Go"))       // true
	fmt.Println(strings.Contains(s, "Python"))   // false
	fmt.Println(strings.Contains(s, ""))          // true — empty string is always "contained"

	// strings.HasPrefix / strings.HasSuffix
	// WHY: Very common in parsing — e.g., checking URL schemes, file extensions,
	// protocol headers. More readable than s[:n] == "prefix".
	fmt.Println(strings.HasPrefix(s, "Hello"))   // true
	fmt.Println(strings.HasSuffix(s, "great."))  // true
	fmt.Println(strings.HasPrefix(s, "World"))   // false

	// strings.Count
	// WHY: Counts non-overlapping instances. For overlapping matches you'd need
	// a manual loop. Count("", "") returns len(s)+1 — a known quirk.
	fmt.Println(strings.Count(s, "Go"))  // 2  (one in "Gophers", one in "Go is")
	fmt.Println(strings.Count(s, "o"))   // 3
	fmt.Println(strings.Count("aaa", "aa")) // 1 — non-overlapping!

	// strings.Index / strings.LastIndex
	// WHY: Returns byte offset (not rune index). For multi-byte UTF-8, you may
	// need to work in rune space. Returns -1 if not found.
	fmt.Println(strings.Index(s, "Go"))      // 7  (byte position)
	fmt.Println(strings.LastIndex(s, "Go"))  // 19

	// strings.IndexRune — find a specific rune
	fmt.Println(strings.IndexRune(s, 'G'))   // 0

	// strings.IndexAny — find first occurrence of ANY rune in a set
	fmt.Println(strings.IndexAny(s, "aeiou")) // 1 (the 'e' in Hello)

	// strings.ContainsAny / strings.ContainsRune
	fmt.Println(strings.ContainsAny(s, "xyz")) // false
	fmt.Println(strings.ContainsRune(s, '!'))  // true

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Splitting & Joining
// ─────────────────────────────────────────────────────────────────────────────

func splitJoinFunctions() {
	fmt.Println("═══ SECTION 2: Split & Join ═══")

	csv := "alice,bob,charlie,dave"

	// strings.Split
	// WHY: Tokenizes a string by a separator. Returns a slice of strings.
	// If sep is "", it splits into individual UTF-8 characters.
	// COMMON MISTAKE: strings.Split("a,,b", ",") gives ["a", "", "b"] — empty strings!
	parts := strings.Split(csv, ",")
	fmt.Println(parts)        // [alice bob charlie dave]
	fmt.Println(len(parts))   // 4

	// Splitting with empty separator
	chars := strings.Split("Go", "")
	fmt.Println(chars) // [G o]

	// strings.SplitN — limit number of substrings
	// WHY: When you only care about the first N pieces (e.g., parsing "key=value=extra")
	kv := "name=John=Doe"
	twoparts := strings.SplitN(kv, "=", 2)
	fmt.Println(twoparts) // [name John=Doe]

	// strings.SplitAfter — keeps the separator at the end of each piece
	// WHY: Useful for splitting log lines while preserving newlines
	lines := strings.SplitAfter("a\nb\nc", "\n")
	fmt.Println(lines) // [a\n b\n c]

	// strings.Fields
	// WHY: Splits on ANY whitespace (spaces, tabs, newlines) and ignores leading/trailing.
	// This is what you want 90% of the time when splitting human-readable text.
	// strings.Split(s, " ") fails with double spaces; Fields does not.
	text := "  hello   world   go  "
	words := strings.Fields(text)
	fmt.Println(words)      // [hello world go]
	fmt.Println(len(words)) // 3

	// strings.FieldsFunc — split on a custom predicate
	// WHY: Flexible alternative when your separator is not a fixed string.
	customSplit := strings.FieldsFunc("one1two2three3four", func(r rune) bool {
		return unicode.IsDigit(r)
	})
	fmt.Println(customSplit) // [one two three four]

	// strings.Join
	// WHY: The inverse of Split. Joins a slice with a separator.
	// MORE EFFICIENT than concatenation in a loop because it pre-allocates.
	rejoined := strings.Join(parts, " | ")
	fmt.Println(rejoined) // alice | bob | charlie | dave

	// Common pattern: filter and rejoin
	filtered := make([]string, 0)
	for _, p := range parts {
		if len(p) > 3 {
			filtered = append(filtered, p)
		}
	}
	fmt.Println(strings.Join(filtered, ", ")) // alice, charlie, dave

	// strings.Cut (Go 1.18+)
	// WHY: The cleanest way to split a string around the FIRST occurrence of a separator.
	// Returns (before, after, found). Replaces the clunky SplitN(s, sep, 2) pattern.
	before, after, found := strings.Cut("user:password@host", ":")
	fmt.Printf("before=%q after=%q found=%v\n", before, after, found)
	// before="user" after="password@host" found=true

	_, _, notFound := strings.Cut("nocolon", ":")
	fmt.Println("notFound:", notFound) // false

	// strings.CutPrefix / strings.CutSuffix (Go 1.20+)
	// WHY: Cleaner than HasPrefix + TrimPrefix when you need to check-and-strip.
	rest, wasCut := strings.CutPrefix("https://example.com", "https://")
	fmt.Printf("rest=%q wasCut=%v\n", rest, wasCut) // rest="example.com" wasCut=true

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Trimming
// ─────────────────────────────────────────────────────────────────────────────

func trimmingFunctions() {
	fmt.Println("═══ SECTION 3: Trimming ═══")

	// strings.TrimSpace
	// WHY: Remove leading and trailing whitespace (spaces, tabs, newlines, etc.)
	// This is the most commonly needed trim. Internally calls TrimFunc with unicode.IsSpace.
	padded := "   hello world   \n\t"
	fmt.Printf("%q\n", strings.TrimSpace(padded)) // "hello world"

	// strings.Trim — trim a CUTSET of characters (any char in the set, from both ends)
	// WHY: "cutset" means ANY character in the string is trimmed, not the whole string.
	// COMMON MISTAKE: confusing Trim with TrimPrefix — Trim removes individual chars,
	// TrimPrefix removes the whole prefix exactly.
	fmt.Println(strings.Trim("***hello***", "*"))      // hello
	fmt.Println(strings.Trim("xyzHello worldxyz", "xyz")) // Hello world (x,y,z removed from edges)
	fmt.Println(strings.Trim("abcdefg", "gfedcba"))    // "" (all chars are in cutset)

	// strings.TrimLeft / strings.TrimRight
	// WHY: One-sided trimming for cases like stripping leading zeros or trailing newlines.
	fmt.Println(strings.TrimLeft("000123", "0"))   // 123
	fmt.Println(strings.TrimRight("hello\n\n", "\n")) // hello

	// strings.TrimPrefix / strings.TrimSuffix — remove exact substring (not a cutset!)
	// WHY: More precise than Trim when you want to remove a specific known string.
	// Does NOT remove character-by-character; removes the whole prefix if it exists.
	fmt.Println(strings.TrimPrefix("Hello, World", "Hello, ")) // World
	fmt.Println(strings.TrimPrefix("Hello, World", "Bye, "))   // Hello, World (no change)
	fmt.Println(strings.TrimSuffix("image.png", ".png"))       // image

	// strings.TrimFunc — trim based on a predicate function
	// WHY: Maximum flexibility. Remove any leading/trailing chars matching your logic.
	result := strings.TrimFunc("123Hello456", unicode.IsDigit)
	fmt.Println(result) // Hello

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Replacement & Case
// ─────────────────────────────────────────────────────────────────────────────

func replacementAndCase() {
	fmt.Println("═══ SECTION 4: Replacement & Case ═══")

	// strings.Replace
	// WHY: Replace first N occurrences. Pass -1 for "all".
	// IMPORTANT: This always allocates a new string.
	s := "aababab"
	fmt.Println(strings.Replace(s, "ab", "X", 1))  // aXabab  (only first)
	fmt.Println(strings.Replace(s, "ab", "X", 2))  // aXXab
	fmt.Println(strings.Replace(s, "ab", "X", -1)) // aXXX (all)

	// strings.ReplaceAll — sugar for Replace(s, old, new, -1)
	// WHY: Clearer intent when you want to replace all occurrences.
	fmt.Println(strings.ReplaceAll("foo bar foo baz foo", "foo", "qux"))
	// qux bar qux baz qux

	// strings.NewReplacer — multiple replacements in ONE pass
	// WHY: Much more efficient than chaining ReplaceAll calls when you have many
	// substitutions. Internally builds a trie for O(n) replacement.
	// Use case: HTML escaping, template variable substitution.
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	html := r.Replace(`<a href="url">Hello & World</a>`)
	fmt.Println(html)
	// &lt;a href=&quot;url&quot;&gt;Hello &amp; World&lt;/a&gt;

	// Case functions
	// WHY: ToUpper/ToLower are Unicode-aware — they handle accented chars correctly.
	// COMMON MISTAKE: using ToUpper for case-insensitive comparison — use EqualFold instead!
	fmt.Println(strings.ToUpper("hello world"))  // HELLO WORLD
	fmt.Println(strings.ToLower("HELLO WORLD"))  // hello world
	fmt.Println(strings.Title("hello world"))    // Hello World (deprecated — use golang.org/x/text)
	fmt.Println(strings.ToTitle("hello world"))  // HELLO WORLD (all caps — not title case!)

	// strings.EqualFold — case-insensitive comparison
	// WHY: More correct than ToLower(a) == ToLower(b) because:
	// 1. Avoids two allocations
	// 2. Handles Unicode case-folding edge cases (e.g., German ß == SS)
	fmt.Println(strings.EqualFold("Go", "go"))     // true
	fmt.Println(strings.EqualFold("Go", "GO"))     // true
	fmt.Println(strings.EqualFold("Go", "java"))   // false
	fmt.Println(strings.EqualFold("ß", "SS"))      // true (Unicode folding!)

	// strings.Map — transform each rune
	// WHY: Apply a custom function to every rune. More flexible than Replace.
	rot13 := strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		}
		return r
	}, "Hello, World!")
	fmt.Println(rot13) // Uryyb, Jbeyq!

	// strings.Repeat
	fmt.Println(strings.Repeat("Go! ", 3)) // Go! Go! Go!

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: strings.Builder — Efficient String Construction
// ─────────────────────────────────────────────────────────────────────────────

func builderDemo() {
	fmt.Println("═══ SECTION 5: strings.Builder ═══")

	// WHY strings.Builder EXISTS:
	// In Go, strings are immutable. Every `s = s + "more"` or `s += "more"`:
	// 1. Allocates a new backing array
	// 2. Copies existing content
	// 3. Copies the new content
	// For N concatenations of strings of average length L, this is O(N²L) work.
	// strings.Builder maintains a []byte internally and grows exponentially,
	// giving O(N) amortized performance.
	//
	// RULE: Use Builder whenever you're building a string in a loop.
	// Use Join when you already have a []string.

	var b strings.Builder

	// Write methods — mirroring the io.Writer interface
	b.WriteString("Hello")           // most common
	b.WriteRune(',')                  // write a single rune
	b.WriteByte(' ')                  // write a single byte
	fmt.Fprintf(&b, "%s!", "World")  // Builder implements io.Writer, so fmt.Fprintf works!

	result := b.String() // Get the result — zero copy internally (since Go 1.10)
	fmt.Println(result)  // Hello, World!
	fmt.Println("Len:", b.Len()) // 13

	// Reset — reuse the builder (keeps allocated capacity)
	// WHY: In a loop, Reset is much cheaper than creating a new Builder each iteration
	b.Reset()
	fmt.Println("After reset, len:", b.Len()) // 0

	// Pre-allocate with Grow
	// WHY: If you know the final size, Grow avoids all intermediate reallocations
	b.Grow(100)
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&b, "item%d ", i)
	}
	fmt.Println(b.String())

	// COMMON MISTAKE: copying a Builder
	// A Builder must not be copied after first use (it has a pointer to its buffer).
	// The compiler will warn you if you pass it by value after writing to it.
	// Always pass *strings.Builder.

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: strings.NewReader
// ─────────────────────────────────────────────────────────────────────────────

func readerDemo() {
	fmt.Println("═══ SECTION 6: strings.NewReader ═══")

	// strings.NewReader creates an io.Reader from a string.
	// WHY: Many functions in Go accept io.Reader (json.NewDecoder, http.NewRequest body,
	// xml.NewDecoder, etc.). NewReader lets you pass a string to any of these without
	// writing it to a temp file or converting to []byte and using bytes.NewReader.
	//
	// strings.NewReader also implements:
	// - io.Reader (Read)
	// - io.ReaderAt (ReadAt)
	// - io.Seeker (Seek)
	// - io.WriterTo (WriteTo)
	// - io.ByteScanner (ReadByte, UnreadByte)
	// - io.RuneScanner (ReadRune, UnreadRune)

	reader := strings.NewReader("Hello, Go!")
	fmt.Println("Size:", reader.Len()) // 10 — unread bytes remaining

	buf := make([]byte, 5)
	n, _ := reader.Read(buf)
	fmt.Printf("Read %d bytes: %q\n", n, buf[:n]) // "Hello"
	fmt.Println("Remaining:", reader.Len())        // 5

	// Seek back to beginning
	reader.Seek(0, 0) // io.SeekStart = 0
	fmt.Println("After seek, remaining:", reader.Len()) // 10

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Performance — Builder vs + operator
// ─────────────────────────────────────────────────────────────────────────────

func performanceComparison() {
	fmt.Println("═══ SECTION 7: Performance Comparison ═══")

	const N = 100_000
	items := make([]string, N)
	for i := range items {
		items[i] = "x"
	}

	// Method 1: + operator in a loop — O(N²) — BAD for large N
	start := time.Now()
	s1 := ""
	for _, item := range items {
		s1 += item // each iteration: allocate, copy old, copy new
	}
	plusTime := time.Since(start)

	// Method 2: strings.Builder — O(N) amortized — GOOD
	start = time.Now()
	var b strings.Builder
	b.Grow(N) // optional but eliminates ALL reallocations
	for _, item := range items {
		b.WriteString(item)
	}
	s2 := b.String()
	builderTime := time.Since(start)

	// Method 3: strings.Join — O(N) with one allocation — BEST when you have []string
	start = time.Now()
	s3 := strings.Join(items, "")
	joinTime := time.Since(start)

	fmt.Printf("+ operator:      %v (result len %d)\n", plusTime, len(s1))
	fmt.Printf("strings.Builder: %v (result len %d)\n", builderTime, len(s2))
	fmt.Printf("strings.Join:    %v (result len %d)\n", joinTime, len(s3))
	fmt.Println()
	fmt.Println("LESSON: Use Builder in loops, Join when you have a []string already.")
	fmt.Println("        + is fine for < ~5 concatenations in non-loop code.")

	// WHY Join is often fastest:
	// It calculates the total length first, makes ONE allocation, then copies.
	// Builder may still do O(log N) reallocations as it grows exponentially.
	// But with Grow(), Builder matches Join performance.

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 8: Miscellaneous Useful Functions
// ─────────────────────────────────────────────────────────────────────────────

func miscFunctions() {
	fmt.Println("═══ SECTION 8: Miscellaneous ═══")

	// strings.Clone (Go 1.20+)
	// WHY: Forces a copy of the string's backing array. Useful when you have a
	// small string derived from a large one (via slicing) and want to free the
	// large backing array for GC.
	large := strings.Repeat("x", 1_000_000)
	small := large[:5]
	// small still holds a reference to large's backing array — 1MB not freed!
	independent := strings.Clone(small) // now truly independent copy
	_ = independent
	_ = large

	// strings.Compare
	// WHY: Returns -1, 0, 1 like C's strcmp. Mostly only needed for sort.Search
	// or custom sorting. For equality, == is simpler and equally fast.
	fmt.Println(strings.Compare("a", "b"))  // -1
	fmt.Println(strings.Compare("b", "a"))  // 1
	fmt.Println(strings.Compare("a", "a"))  // 0

	// Checking emptiness
	s := ""
	fmt.Println(len(s) == 0)              // idiomatic in Go
	fmt.Println(s == "")                  // also fine
	// strings.Contains(s, "") is true but NOT the right way to check emptiness

	// strings.IndexFunc — find first rune matching predicate
	idx := strings.IndexFunc("Hello123", unicode.IsDigit)
	fmt.Println(idx) // 5

	// strings.LastIndexFunc
	lastIdx := strings.LastIndexFunc("Hello123World456", unicode.IsDigit)
	fmt.Println(lastIdx) // 15

	// Summary of functions returning -1 on "not found":
	// Index, LastIndex, IndexByte, IndexRune, IndexAny, LastIndexAny, IndexFunc, LastIndexFunc

	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         Go Standard Library: strings Package          ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	searchingFunctions()
	splitJoinFunctions()
	trimmingFunctions()
	replacementAndCase()
	builderDemo()
	readerDemo()
	performanceComparison()
	miscFunctions()

	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. Use EqualFold for case-insensitive comparison (not ToLower+==)")
	fmt.Println("  2. Use Builder in loops; use Join when you have a []string")
	fmt.Println("  3. Trim removes chars from a CUTSET; TrimPrefix removes exact string")
	fmt.Println("  4. strings.Cut (1.18+) is cleaner than SplitN for key=value parsing")
	fmt.Println("  5. NewReplacer is efficient for multi-substitution (HTML escaping etc.)")
	fmt.Println("  6. strings.Fields handles multiple/mixed whitespace; Split does not")
}
