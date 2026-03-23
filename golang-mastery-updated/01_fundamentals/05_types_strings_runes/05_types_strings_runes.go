// FILE: 01_fundamentals/05_types_strings_runes.go
// TOPIC: Strings & Runes — UTF-8 internals, byte vs rune iteration, string immutability
//
// Run: go run 01_fundamentals/05_types_strings_runes.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Go strings are NOT just arrays of characters. They are byte slices
//   encoded in UTF-8. If you loop over a string with index [i], you get BYTES.
//   If you want CHARACTERS (runes), you need range. Getting this wrong causes
//   garbled output, wrong lengths, and index out-of-bounds panics when handling
//   any non-ASCII text (most of the world's languages).
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Strings & Runes")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// WHAT IS A STRING IN GO?
	// ─────────────────────────────────────────────────────────────────────
	//
	// A string in Go is a READ-ONLY slice of bytes.
	// Internally it's a struct with two fields:
	//   { ptr *byte, len int }
	//
	// The bytes are UTF-8 encoded. UTF-8 is a variable-width encoding:
	//   - ASCII characters (0-127): 1 byte
	//   - Most European characters: 2 bytes
	//   - Most CJK characters (Chinese, Japanese, Korean): 3 bytes
	//   - Some emoji and rare scripts: 4 bytes
	//
	// IMMUTABILITY: You CANNOT modify individual bytes of a string.
	//   s := "hello"
	//   s[0] = 'H'   ← compile error: cannot assign to s[0]
	//
	// WHY IMMUTABLE?
	//   - Strings can be safely shared across goroutines without locking
	//   - String literals are stored in read-only memory segment
	//   - Allows Go to optimize string headers (no copy when passing)
	//
	// To modify: convert to []byte, modify, convert back to string.

	s := "Hello, 世界"  // contains ASCII and Chinese characters
	fmt.Printf("\n── String internals ──\n")
	fmt.Printf("  string: %q\n", s)
	fmt.Printf("  len(s) = %d bytes  (NOT characters!)\n", len(s))
	fmt.Printf("  utf8.RuneCountInString(s) = %d runes (characters)\n",
		utf8.RuneCountInString(s))
	// "世" takes 3 bytes in UTF-8, "界" takes 3 bytes → 6 extra bytes for 2 chars

	// ─────────────────────────────────────────────────────────────────────
	// STRING LITERALS — Two forms
	// ─────────────────────────────────────────────────────────────────────

	// 1. Interpreted string literal: double quotes, supports escape sequences
	interpreted := "Line 1\nLine 2\tTabbed\n"
	// Common escape sequences:
	//   \n = newline     \t = tab      \r = carriage return
	//   \" = double quote \\ = backslash
	//   \xHH = hex byte  \uHHHH = Unicode (4 hex digits)  \UHHHHHHHH = Unicode (8 hex digits)
	fmt.Printf("\n── Interpreted literal ──\n%s", interpreted)

	// 2. Raw string literal: backticks, NO escape processing, can span lines
	// Use for: regex patterns, JSON templates, SQL queries, multiline text
	raw := `SELECT *
FROM users
WHERE name = "Achyut"
AND active = true`
	fmt.Printf("\n── Raw string literal (backtick) ──\n%s\n", raw)

	// Regex example — raw strings are perfect here (no double-escaping \)
	regexPattern := `\d{4}-\d{2}-\d{2}`  // matches date like 2024-01-15
	fmt.Printf("  Regex pattern: %s\n", regexPattern)

	// ─────────────────────────────────────────────────────────────────────
	// INDEXING: s[i] gives BYTES, not characters
	// ─────────────────────────────────────────────────────────────────────

	ascii := "Hello"
	fmt.Printf("\n── Byte indexing (ASCII string) ──\n")
	for i := 0; i < len(ascii); i++ {
		fmt.Printf("  s[%d] = %d = %q\n", i, ascii[i], ascii[i])
	}

	// For non-ASCII, byte indexing gives you raw UTF-8 bytes, NOT characters:
	chinese := "世界"
	fmt.Printf("\n── Byte indexing (UTF-8 multi-byte string) ──\n")
	fmt.Printf("  string: %q  len=%d bytes, but %d runes\n",
		chinese, len(chinese), utf8.RuneCountInString(chinese))
	for i := 0; i < len(chinese); i++ {
		fmt.Printf("  byte[%d] = 0x%X\n", i, chinese[i])
	}
	// You see 6 raw bytes, not 2 characters. Don't index by byte for Unicode!

	// ─────────────────────────────────────────────────────────────────────
	// RANGE OVER STRING: gives RUNES (Unicode code points)
	// ─────────────────────────────────────────────────────────────────────
	//
	// When you use 'range' on a string, Go automatically decodes UTF-8
	// and gives you:
	//   - i: the BYTE index of the start of the rune (not 0,1,2,3...)
	//   - r: the rune value (Unicode code point, type rune = int32)
	//
	// This is the correct way to iterate over characters in a string.

	fmt.Printf("\n── Range over string (rune iteration) ──\n")
	for i, r := range "Hello, 世界" {
		fmt.Printf("  byte_idx=%d rune=%c (U+%04X)\n", i, r, r)
	}
	// Notice: '世' starts at byte_idx=7, '界' starts at byte_idx=10
	// Each Chinese char takes 3 bytes, so byte indices skip by 3.

	// ─────────────────────────────────────────────────────────────────────
	// CONVERTING BETWEEN string, []byte, []rune
	// ─────────────────────────────────────────────────────────────────────
	//
	// string ↔ []byte:
	//   Use when you need to MODIFY the bytes (strings are immutable).
	//   Conversion COPIES the data (strings are immutable, []byte is mutable).
	//
	// string ↔ []rune:
	//   Use when you need to index by CHARACTER position.
	//   e.g., "get the 3rd character" → convert to []rune first.

	original := "Hello, 世界"

	// string → []byte (get mutable bytes)
	bytes := []byte(original)
	bytes[0] = 'h'  // modify first byte (safe because it's ASCII)
	modified := string(bytes)
	fmt.Printf("\n── string ↔ []byte ──\n")
	fmt.Printf("  original:  %q\n", original)
	fmt.Printf("  modified:  %q\n", modified)

	// string → []rune (get characters by index)
	runes := []rune(original)
	fmt.Printf("\n── string ↔ []rune ──\n")
	fmt.Printf("  rune count: %d\n", len(runes))
	fmt.Printf("  rune[7]: %c  ← 8th character (the '世')\n", runes[7])
	// If we had used bytes[7], we'd get 0xE4 (first byte of UTF-8 encoding of '世')

	// ─────────────────────────────────────────────────────────────────────
	// STRING CONCATENATION — + operator vs strings.Builder
	// ─────────────────────────────────────────────────────────────────────
	//
	// + creates a NEW string (allocates memory) on each concatenation.
	// In a loop: str += "x" runs N times → N allocations → O(N²) performance.
	//
	// strings.Builder is the efficient way to build strings incrementally.
	// It maintains a buffer and only allocates once at the end.
	// Use Builder when you're building a string in a loop or from many pieces.

	// BAD (for loops): each += allocates a new string
	result := ""
	for i := 0; i < 5; i++ {
		result += fmt.Sprintf("item%d ", i)
	}
	fmt.Printf("\n── String concatenation ──\n")
	fmt.Printf("  + operator result: %q\n", result)

	// GOOD: strings.Builder
	var sb strings.Builder
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "item%d ", i)
	}
	fmt.Printf("  Builder result:    %q\n", sb.String())

	// ─────────────────────────────────────────────────────────────────────
	// STRING COMPARISON — Lexicographic, uses byte values
	// ─────────────────────────────────────────────────────────────────────
	//
	// Strings in Go can be compared with ==, !=, <, >, <=, >=
	// Comparison is LEXICOGRAPHIC (dictionary order) based on byte values.
	// This means uppercase comes before lowercase (ASCII 'A'=65 < 'a'=97).
	//
	// For case-insensitive comparison: strings.EqualFold()

	fmt.Printf("\n── String comparison ──\n")
	fmt.Printf("  \"abc\" < \"abd\" : %v\n", "abc" < "abd")
	fmt.Printf("  \"ABC\" < \"abc\" : %v  (uppercase < lowercase in ASCII)\n", "ABC" < "abc")
	fmt.Printf("  strings.EqualFold(\"Go\",\"go\") : %v\n", strings.EqualFold("Go", "go"))

	// ─────────────────────────────────────────────────────────────────────
	// RUNE LITERALS AND CHARACTER CODES
	// ─────────────────────────────────────────────────────────────────────

	fmt.Printf("\n── Rune literals ──\n")
	fmt.Printf("  'A'       = %d\n", 'A')
	fmt.Printf("  'a'       = %d\n", 'a')
	fmt.Printf("  '0'       = %d\n", '0')
	fmt.Printf("  '\\n'     = %d  (newline)\n", '\n')
	fmt.Printf("  '\\t'     = %d  (tab)\n", '\t')
	fmt.Printf("  '世'      = %d  (Unicode: U+4E16)\n", '世')
	fmt.Printf("  '\\u4e16' = %c  (same as '世')\n", '\u4e16')

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  string = immutable []byte, UTF-8 encoded")
	fmt.Println("  len(s) = byte count (NOT character count)")
	fmt.Println("  range s → runes (correct for Unicode)")
	fmt.Println("  s[i] → byte (only safe for ASCII)")
	fmt.Println("  []rune(s) to index by character position")
	fmt.Println("  strings.Builder for efficient string building")
	fmt.Println("  strings.EqualFold for case-insensitive compare")
}
