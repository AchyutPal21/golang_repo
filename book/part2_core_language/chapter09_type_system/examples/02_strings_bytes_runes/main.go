// FILE: book/part2_core_language/chapter09_type_system/examples/02_strings_bytes_runes/main.go
// CHAPTER: 09 — The Type System
// TOPIC: The canonical UTF-8 demonstration: bytes vs runes vs characters.
//
// Run (from the chapter folder):
//   go run ./examples/02_strings_bytes_runes
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   "How do I get the 5th character?" is a question that looks trivial and
//   is actually subtle in Go. This file walks through it on a string with
//   ASCII, accented Latin, CJK, and emoji — every one a different number of
//   bytes per rune.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	// ASCII only — bytes == runes.
	ascii := "hello"
	report("ASCII", ascii)

	// Accented Latin — 'é' is 2 bytes, 1 rune.
	french := "héllo"
	report("French", french)

	// CJK — Chinese characters are 3 bytes each.
	chinese := "你好"
	report("Chinese", chinese)

	// Emoji — single emojis are 4 bytes, multi-codepoint emojis are more.
	emoji := "🎉🌍"
	report("Emoji", emoji)

	fmt.Println("\n=== Indexing pitfall ===")
	s := "héllo"
	fmt.Printf("  s[0] = %d (the FIRST BYTE — 'h')\n", s[0])
	fmt.Printf("  s[1] = %d (FIRST byte of 'é', NOT 'é' itself!)\n", s[1])
	fmt.Printf("  s[2] = %d (SECOND byte of 'é')\n", s[2])

	fmt.Println("\n=== Iterating with range gives you runes ===")
	for i, r := range s {
		fmt.Printf("  byte index %d: rune %q (U+%04X, %d bytes)\n",
			i, r, r, utf8.RuneLen(r))
	}

	fmt.Println("\n=== Bytes vs runes vs characters ===")
	fmt.Printf("  len(s)                     = %d bytes\n", len(s))
	fmt.Printf("  utf8.RuneCountInString(s)  = %d runes\n", utf8.RuneCountInString(s))
	fmt.Printf("  ('characters' as users see = depends on grapheme clusters,\n")
	fmt.Printf("   which can span multiple runes; out of scope here)\n")

	fmt.Println("\n=== Conversion costs ===")
	fmt.Println("  []byte(s) — copies bytes, O(n)")
	fmt.Println("  []rune(s) — decodes UTF-8 into a slice of int32, O(n)")
	fmt.Println("  string(b) — copies bytes, O(n)")
	fmt.Println("  string(r) — encodes runes into UTF-8, O(n)")
	fmt.Println("  All four allocate. Avoid in hot loops.")
}

func report(label, s string) {
	fmt.Printf("\n%s: %q\n", label, s)
	fmt.Printf("  len=%d (bytes)   runes=%d\n",
		len(s), utf8.RuneCountInString(s))
	fmt.Printf("  bytes: ")
	for i := 0; i < len(s); i++ {
		fmt.Printf("%02x ", s[i])
	}
	fmt.Println()
}
