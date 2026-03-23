// FILE: 08_standard_library/08_regexp_package.go
// TOPIC: regexp Package — compile, match, find, replace, named groups
//
// Run: go run 08_standard_library/08_regexp_package.go

package main

import (
	"fmt"
	"regexp"
)

// Package-level: compile ONCE, reuse many times (compilation is expensive)
var (
	reEmail   = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	reDate    = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	reWords   = regexp.MustCompile(`\b\w+\b`)
	reNamed   = regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: regexp Package")
	fmt.Println("════════════════════════════════════════")

	// ── Compile vs MustCompile ──────────────────────────────────────────
	// regexp.Compile(pattern) → returns (*Regexp, error)
	// regexp.MustCompile(pattern) → panics if invalid (use for package-level vars)
	// Use MustCompile for patterns known at compile time (they're constants).
	// Use Compile for user-provided patterns (they might be invalid).
	fmt.Println("\n── Compile vs MustCompile ──")
	r, err := regexp.Compile(`\d+`)
	fmt.Printf("  Compile(\"\\d+\"): r=%v, err=%v\n", r != nil, err)
	_, err2 := regexp.Compile(`[invalid`)
	fmt.Printf("  Compile(\"[invalid\"): err=%v\n", err2)

	// ── MatchString — simple check ──────────────────────────────────────
	fmt.Println("\n── Email validation ──")
	emails := []string{"user@example.com", "invalid@", "a.b+c@d.org", "bad"}
	for _, e := range emails {
		fmt.Printf("  %q → valid: %v\n", e, reEmail.MatchString(e))
	}

	// ── FindString and FindAllString ─────────────────────────────────────
	fmt.Println("\n── FindString / FindAllString ──")
	text := "Call 123-4567 or 987-6543 or visit office 555-0000"
	phoneRe := regexp.MustCompile(`\d{3}-\d{4}`)

	first := phoneRe.FindString(text)
	fmt.Printf("  FindString: %q\n", first)

	all := phoneRe.FindAllString(text, -1)  // -1 = find all
	fmt.Printf("  FindAllString: %v\n", all)

	some := phoneRe.FindAllString(text, 2)  // limit to 2
	fmt.Printf("  FindAllString(n=2): %v\n", some)

	// ── FindStringSubmatch — capture groups ──────────────────────────────
	fmt.Println("\n── Capture groups ──")
	dateStr := "Event on 2024-03-15 and backup on 2024-04-20"
	match := reDate.FindStringSubmatch(dateStr)
	if match != nil {
		fmt.Printf("  Full match: %q\n", match[0])
		fmt.Printf("  Year: %q  Month: %q  Day: %q\n", match[1], match[2], match[3])
	}

	// All matches with groups:
	allMatches := reDate.FindAllStringSubmatch(dateStr, -1)
	fmt.Printf("  All date matches:\n")
	for _, m := range allMatches {
		fmt.Printf("    date=%q year=%q month=%q day=%q\n", m[0], m[1], m[2], m[3])
	}

	// ── Named capture groups ─────────────────────────────────────────────
	fmt.Println("\n── Named capture groups ──")
	m := reNamed.FindStringSubmatch("Release: 2024-03-15")
	if m != nil {
		// SubexpNames() returns ["", "year", "month", "day"]
		names := reNamed.SubexpNames()
		for i, name := range names {
			if i > 0 && name != "" {
				fmt.Printf("  %s = %q\n", name, m[i])
			}
		}
	}

	// ── ReplaceAllString ─────────────────────────────────────────────────
	fmt.Println("\n── ReplaceAllString ──")
	sensitive := "Password: secret123, API key: abc456def"
	masked := regexp.MustCompile(`\b[a-zA-Z0-9]{6,}\b`).ReplaceAllString(sensitive, "****")
	fmt.Printf("  Masked: %q\n", masked)

	// Replace using capture groups ($1, $2, ...):
	dateFormatted := reDate.ReplaceAllString("Date: 2024-03-15", "Date: $3/$2/$1")
	fmt.Printf("  Reformatted date: %q\n", dateFormatted)

	// ── ReplaceAllStringFunc — dynamic replacement ───────────────────────
	fmt.Println("\n── ReplaceAllStringFunc ──")
	shout := reWords.ReplaceAllStringFunc("hello world go", func(w string) string {
		return "[" + w + "]"
	})
	fmt.Printf("  Wrapped words: %q\n", shout)

	// ── Split ────────────────────────────────────────────────────────────
	fmt.Println("\n── Split ──")
	csv := "a, b,  c ,d"
	parts := regexp.MustCompile(`\s*,\s*`).Split(csv, -1)
	fmt.Printf("  Split on ',': %v\n", parts)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  MustCompile at package level — compile once, reuse forever")
	fmt.Println("  MatchString      → bool check")
	fmt.Println("  FindString       → first match")
	fmt.Println("  FindAllString    → all matches (-1 = unlimited)")
	fmt.Println("  FindStringSubmatch → match + capture groups [0]=full, [1+]=groups")
	fmt.Println("  (?P<name>...)    → named capture groups")
	fmt.Println("  ReplaceAllString → replace with literal or $1/$2 groups")
	fmt.Println("  ReplaceAllStringFunc → dynamic replacement")
	fmt.Println("  Go uses RE2 syntax — no lookahead/lookbehind (by design: O(n) time)")
}
