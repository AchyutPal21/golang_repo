// 02_strconv_package.go
//
// The strconv package: converting between strings and other types.
//
// WHY strconv EXISTS:
// Go's fmt package can convert anything to a string (via Sprintf), but it's
// heavy — it uses reflection and interface boxing. strconv is purpose-built,
// allocation-aware, and much faster for the common case of converting numbers.
//
// The two core problems strconv solves:
//   1. Parse: string → typed value  (Atoi, ParseInt, ParseFloat, ParseBool)
//   2. Format: typed value → string (Itoa, FormatInt, FormatFloat, FormatBool)
//
// There are also Append variants (AppendInt, etc.) that write into an existing
// []byte, avoiding any allocation at all.

package main

import (
	"errors"
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: Atoi / Itoa — the most common conversions
// ─────────────────────────────────────────────────────────────────────────────

func atoiItoaDemo() {
	fmt.Println("═══ SECTION 1: Atoi / Itoa ═══")

	// strconv.Itoa — int → string
	// WHY: The name comes from C's itoa (integer to ASCII).
	// This is sugar for FormatInt(int64(i), 10). Use it for decimal int → string.
	s := strconv.Itoa(42)
	fmt.Printf("Itoa(42) = %q (type: %T)\n", s, s) // "42"

	s2 := strconv.Itoa(-1000)
	fmt.Printf("Itoa(-1000) = %q\n", s2) // "-1000"

	// strconv.Atoi — string → int (base 10 only)
	// WHY: Returns (int, error). This forces you to handle parse failures,
	// which is correct — user input can be anything.
	n, err := strconv.Atoi("123")
	if err == nil {
		fmt.Printf("Atoi(%q) = %d\n", "123", n) // 123
	}

	// Error case — Atoi returns *strconv.NumError
	n2, err2 := strconv.Atoi("abc")
	fmt.Printf("Atoi(%q) error: %v\n", "abc", err2)
	fmt.Printf("n2 = %d (zero value on error)\n", n2)

	// COMMON MISTAKE: using fmt.Sprintf for number→string
	// These are equivalent but Itoa is faster:
	//   strconv.Itoa(n)          — fast, no reflection
	//   fmt.Sprintf("%d", n)     — slower, uses reflection + formatting
	//   fmt.Sprint(n)            — same as above
	// Benchmark: Itoa is ~5-10x faster than Sprintf for integer formatting.

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: ParseInt — the full-power integer parser
// ─────────────────────────────────────────────────────────────────────────────

func parseIntDemo() {
	fmt.Println("═══ SECTION 2: ParseInt / ParseUint ═══")

	// strconv.ParseInt(s, base, bitSize) (int64, error)
	//
	// base:    2–36, or 0 (auto-detect from prefix: 0x=hex, 0=octal, 0b=binary)
	// bitSize: 0=int, 8=int8, 16=int16, 32=int32, 64=int64
	//          Controls the range check — result is still int64 but fits in bitSize bits.
	//
	// WHY ParseInt OVER Atoi?
	//   - Atoi only handles base-10 and always returns int (platform-dependent size)
	//   - ParseInt handles any base and enforces a bit-width range check
	//   - Use ParseInt when parsing hex, binary, octal, or needing int32/int64 specifically

	// Decimal (base 10)
	v1, _ := strconv.ParseInt("42", 10, 64)
	fmt.Printf("ParseInt(%q, 10, 64) = %d\n", "42", v1)

	// Hexadecimal (base 16)
	v2, _ := strconv.ParseInt("1F", 16, 64)
	fmt.Printf("ParseInt(%q, 16, 64) = %d (0x1F)\n", "1F", v2) // 31

	// Binary (base 2)
	v3, _ := strconv.ParseInt("1010", 2, 64)
	fmt.Printf("ParseInt(%q, 2, 64) = %d (binary 1010)\n", "1010", v3) // 10

	// Octal (base 8)
	v4, _ := strconv.ParseInt("17", 8, 64)
	fmt.Printf("ParseInt(%q, 8, 64) = %d (octal 17)\n", "17", v4) // 15

	// Auto-detect base with base=0 (reads prefix)
	v5, _ := strconv.ParseInt("0xFF", 0, 64)
	fmt.Printf("ParseInt(%q, 0, 64) = %d (auto hex)\n", "0xFF", v5) // 255

	v6, _ := strconv.ParseInt("0b1010", 0, 64)
	fmt.Printf("ParseInt(%q, 0, 64) = %d (auto binary)\n", "0b1010", v6) // 10

	// Negative numbers
	v7, _ := strconv.ParseInt("-255", 10, 64)
	fmt.Printf("ParseInt(%q, 10, 64) = %d\n", "-255", v7) // -255

	// Bit-size range overflow
	_, err := strconv.ParseInt("1000", 10, 8) // int8 max = 127
	fmt.Printf("ParseInt(%q, 10, 8) error: %v\n", "1000", err) // range error

	// ParseUint — same but for unsigned integers (no negative numbers)
	u, _ := strconv.ParseUint("255", 10, 8) // fits in uint8
	fmt.Printf("ParseUint(%q, 10, 8) = %d\n", "255", u)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: ParseFloat
// ─────────────────────────────────────────────────────────────────────────────

func parseFloatDemo() {
	fmt.Println("═══ SECTION 3: ParseFloat ═══")

	// strconv.ParseFloat(s, bitSize) (float64, error)
	//
	// bitSize: 32 or 64
	//   - 32: result fits in float32 precision (but returned as float64)
	//   - 64: full float64 precision
	//
	// WHY not just use fmt.Sscanf?
	// ParseFloat is faster and gives you the *strconv.NumError with detail.

	f1, _ := strconv.ParseFloat("3.14", 64)
	fmt.Printf("ParseFloat(%q, 64) = %.10f\n", "3.14", f1)

	f2, _ := strconv.ParseFloat("3.14", 32)
	fmt.Printf("ParseFloat(%q, 32) = %.10f (float32 precision)\n", "3.14", f2)
	// Note: f2 has float32 precision even though it's stored as float64

	// Scientific notation is supported
	f3, _ := strconv.ParseFloat("1.5e10", 64)
	fmt.Printf("ParseFloat(%q, 64) = %g\n", "1.5e10", f3) // 1.5e+10

	// Special values
	f4, _ := strconv.ParseFloat("Inf", 64)
	f5, _ := strconv.ParseFloat("-Inf", 64)
	f6, _ := strconv.ParseFloat("NaN", 64)
	fmt.Printf("Inf=%v  -Inf=%v  NaN=%v\n", f4, f5, f6)

	// Error handling
	_, err := strconv.ParseFloat("not-a-number", 64)
	fmt.Printf("ParseFloat error: %v\n", err)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: ParseBool
// ─────────────────────────────────────────────────────────────────────────────

func parseBoolDemo() {
	fmt.Println("═══ SECTION 4: ParseBool / FormatBool ═══")

	// strconv.ParseBool accepts:
	// true:  "1", "t", "T", "TRUE", "true", "True"
	// false: "0", "f", "F", "FALSE", "false", "False"
	// WHY: Config files, environment variables, and CLI flags often represent booleans
	// as strings. This single function handles all common variants.

	trueValues := []string{"1", "t", "T", "TRUE", "true", "True"}
	for _, v := range trueValues {
		b, _ := strconv.ParseBool(v)
		fmt.Printf("ParseBool(%q) = %v\n", v, b)
	}

	_, err := strconv.ParseBool("yes") // "yes" is NOT accepted!
	fmt.Printf("ParseBool(%q) error: %v\n", "yes", err)

	// FormatBool — bool → "true" or "false"
	fmt.Println(strconv.FormatBool(true))  // true
	fmt.Println(strconv.FormatBool(false)) // false

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: FormatInt / FormatFloat — typed values to strings
// ─────────────────────────────────────────────────────────────────────────────

func formatDemo() {
	fmt.Println("═══ SECTION 5: FormatInt / FormatFloat ═══")

	// strconv.FormatInt(i int64, base int) string
	// WHY: Convert an integer to any base. Essential for hex output, binary debugging.
	fmt.Println(strconv.FormatInt(255, 10))  // 255  (decimal)
	fmt.Println(strconv.FormatInt(255, 16))  // ff   (hex)
	fmt.Println(strconv.FormatInt(255, 2))   // 11111111  (binary)
	fmt.Println(strconv.FormatInt(255, 8))   // 377  (octal)
	fmt.Println(strconv.FormatInt(255, 36))  // 73   (base 36)

	// Negative numbers
	fmt.Println(strconv.FormatInt(-42, 10)) // -42
	fmt.Println(strconv.FormatInt(-42, 16)) // -2a

	// strconv.FormatFloat(f float64, fmt byte, prec, bitSize int) string
	//
	// fmt bytes:
	//   'e' — scientific: -1.234567e+08
	//   'E' — scientific uppercase: -1.234567E+08
	//   'f' — no exponent: 123456789.123456
	//   'g' — shortest representation (uses 'e' for large, 'f' for small)
	//   'G' — like 'g' but uppercase
	//   'x' — hex mantissa, binary exponent: -0x1.23abcp+20
	//   'b' — binary exponent: -123456p-78
	//
	// prec: number of digits after decimal (-1 = shortest)
	// bitSize: 32 or 64 (affects precision rounding)

	pi := 3.141592653589793

	fmt.Printf("'f' prec=2:  %s\n", strconv.FormatFloat(pi, 'f', 2, 64))  // 3.14
	fmt.Printf("'f' prec=10: %s\n", strconv.FormatFloat(pi, 'f', 10, 64)) // 3.1415926536
	fmt.Printf("'e' prec=4:  %s\n", strconv.FormatFloat(pi, 'e', 4, 64))  // 3.1416e+00
	fmt.Printf("'g' prec=-1: %s\n", strconv.FormatFloat(pi, 'g', -1, 64)) // shortest
	fmt.Printf("'g' prec=3:  %s\n", strconv.FormatFloat(pi, 'g', 3, 64))  // 3.14

	// prec=-1 gives the minimum digits to represent the value exactly (round-trip safe)
	fmt.Println(strconv.FormatFloat(0.1+0.2, 'f', -1, 64)) // 0.30000000000000004

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Append variants — zero allocation
// ─────────────────────────────────────────────────────────────────────────────

func appendVariantsDemo() {
	fmt.Println("═══ SECTION 6: Append Variants (Zero Allocation) ═══")

	// AppendInt, AppendFloat, AppendBool, AppendQuote
	//
	// WHY THESE EXIST:
	// strconv.Itoa(n) always allocates a new string.
	// AppendInt appends the digits directly into an EXISTING []byte — no allocation
	// if the slice has sufficient capacity.
	//
	// This is critical in hot paths: log formatters, network protocol encoders,
	// JSON serializers — all use the Append variants to avoid allocations.

	buf := make([]byte, 0, 64) // pre-allocated buffer

	buf = strconv.AppendInt(buf, 42, 10)
	buf = append(buf, ',')
	buf = strconv.AppendInt(buf, 255, 16) // "ff"
	buf = append(buf, ',')
	buf = strconv.AppendFloat(buf, 3.14, 'f', 2, 64)
	buf = append(buf, ',')
	buf = strconv.AppendBool(buf, true)

	fmt.Printf("Buffer: %s\n", buf) // 42,ff,3.14,true

	// AppendQuote — adds Go-syntax quoting (with escape sequences)
	buf2 := make([]byte, 0, 32)
	buf2 = strconv.AppendQuote(buf2, `Hello "World"`)
	fmt.Printf("AppendQuote: %s\n", buf2) // "Hello \"World\""

	// Real-world pattern: building a CSV row without allocations
	row := make([]byte, 0, 128)
	values := []int{1, 42, 100, 9999}
	for i, v := range values {
		if i > 0 {
			row = append(row, ',')
		}
		row = strconv.AppendInt(row, int64(v), 10)
	}
	fmt.Printf("CSV row: %s\n", row)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Quote / Unquote — Go string literal syntax
// ─────────────────────────────────────────────────────────────────────────────

func quoteDemo() {
	fmt.Println("═══ SECTION 7: Quote / Unquote ═══")

	// strconv.Quote — wraps a string in double quotes and escapes special chars
	// WHY: Useful for debug output, code generation, and safe logging (shows
	// invisible characters, newlines, tabs explicitly).
	fmt.Println(strconv.Quote("Hello, World"))     // "Hello, World"
	fmt.Println(strconv.Quote("tab:\there"))       // "tab:\there"
	fmt.Println(strconv.Quote("newline:\nend"))    // "newline:\nend"
	fmt.Println(strconv.Quote(`quote: "hi"`))      // "quote: \"hi\""
	fmt.Println(strconv.Quote("unicode: \u00e9"))  // "unicode: é"

	// strconv.QuoteToASCII — like Quote but escapes non-ASCII runes
	fmt.Println(strconv.QuoteToASCII("unicode: \u00e9")) // "unicode: \u00e9"

	// strconv.Unquote — parses a quoted Go string literal
	// WHY: Useful when reading Go source files, JSON-style data, or config that
	// uses Go string syntax.
	s, err := strconv.Unquote(`"Hello\tWorld"`)
	fmt.Printf("Unquote: %q err=%v\n", s, err) // "Hello\tWorld" err=<nil>

	s2, err2 := strconv.Unquote(`'a'`) // single char (rune literal)
	fmt.Printf("Unquote rune: %q err=%v\n", s2, err2)

	// QuoteRune — quote a rune
	fmt.Println(strconv.QuoteRune('A'))   // 'A'
	fmt.Println(strconv.QuoteRune('\n'))  // '\n'
	fmt.Println(strconv.QuoteRune('é'))   // 'é'

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 8: Error Handling — *strconv.NumError
// ─────────────────────────────────────────────────────────────────────────────

func errorHandlingDemo() {
	fmt.Println("═══ SECTION 8: Error Handling ═══")

	// All Parse functions return *strconv.NumError on failure.
	// strconv.NumError has fields:
	//   Func: the function name ("ParseInt", "ParseFloat", etc.)
	//   Num:  the input string
	//   Err:  strconv.ErrRange or strconv.ErrSyntax
	//
	// strconv.ErrSyntax: the string is not valid for the type
	// strconv.ErrRange:  the value is valid but out of range for the bit size

	// ErrSyntax example
	_, err := strconv.ParseInt("abc", 10, 64)
	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		fmt.Printf("Func: %s\n", numErr.Func)
		fmt.Printf("Num:  %s\n", numErr.Num)
		fmt.Printf("Err:  %v\n", numErr.Err)
		fmt.Printf("Is ErrSyntax: %v\n", errors.Is(numErr.Err, strconv.ErrSyntax))
	}

	fmt.Println()

	// ErrRange example
	_, err2 := strconv.ParseInt("9999999999999999999", 10, 64)
	var numErr2 *strconv.NumError
	if errors.As(err2, &numErr2) {
		fmt.Printf("Is ErrRange: %v\n", errors.Is(numErr2.Err, strconv.ErrRange))
	}

	// Pattern: parse with fallback default
	parseWithDefault := func(s string, def int) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			return def
		}
		return n
	}

	fmt.Println(parseWithDefault("42", 0))    // 42
	fmt.Println(parseWithDefault("bad", 0))   // 0 (default)
	fmt.Println(parseWithDefault("", -1))     // -1 (default)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 9: Real-world patterns
// ─────────────────────────────────────────────────────────────────────────────

func realWorldPatterns() {
	fmt.Println("═══ SECTION 9: Real-World Patterns ═══")

	// Pattern 1: Parse environment variable with default
	getEnvInt := func(key string, def int) int {
		// In real code: val := os.Getenv(key)
		val := "100" // simulating os.Getenv
		if val == "" {
			return def
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			return def
		}
		return n
	}
	fmt.Println("PORT:", getEnvInt("PORT", 8080)) // 100

	// Pattern 2: Build a number-formatted filename without allocation
	buildFilename := func(prefix string, n int) string {
		var buf [64]byte
		b := append(buf[:0], prefix...)
		b = strconv.AppendInt(b, int64(n), 10)
		b = append(b, ".log"...)
		return string(b)
	}
	fmt.Println(buildFilename("server-", 42)) // server-42.log

	// Pattern 3: Validate that a string is a valid integer
	isInt := func(s string) bool {
		_, err := strconv.ParseInt(s, 10, 64)
		return err == nil
	}
	fmt.Println(isInt("42"))   // true
	fmt.Println(isInt("-5"))   // true
	fmt.Println(isInt("3.14")) // false
	fmt.Println(isInt(""))     // false

	// Pattern 4: Convert hex color to RGB
	hexToRGB := func(hex string) (r, g, b uint8) {
		hex = strings.TrimPrefix(hex, "#")
		val, _ := strconv.ParseUint(hex, 16, 32)
		return uint8(val >> 16), uint8(val >> 8), uint8(val)
	}

	// We need to import strings for TrimPrefix, so inline it:
	hexColor := "#FF8040"
	hexColor = hexColor[1:] // strip #
	val, _ := strconv.ParseUint(hexColor, 16, 32)
	r, g, b := uint8(val>>16), uint8(val>>8), uint8(val)
	fmt.Printf("RGB(%d, %d, %d)\n", r, g, b) // RGB(255, 128, 64)
	_ = hexToRGB

	fmt.Println()
}

// need strings for TrimPrefix in pattern 4
var strings_TrimPrefix = func(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         Go Standard Library: strconv Package          ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	atoiItoaDemo()
	parseIntDemo()
	parseFloatDemo()
	parseBoolDemo()
	formatDemo()
	appendVariantsDemo()
	quoteDemo()
	errorHandlingDemo()
	realWorldPatterns()

	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. Atoi/Itoa for simple base-10 int conversions")
	fmt.Println("  2. ParseInt for other bases or explicit bit-width control")
	fmt.Println("  3. AppendInt/AppendFloat avoid allocations in hot paths")
	fmt.Println("  4. Always handle the error from Parse functions")
	fmt.Println("  5. Use errors.As(*strconv.NumError) to distinguish ErrSyntax vs ErrRange")
	fmt.Println("  6. Itoa is ~5-10x faster than fmt.Sprintf for integers")
}
