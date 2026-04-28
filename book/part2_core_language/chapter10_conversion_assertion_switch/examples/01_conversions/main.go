// FILE: book/part2_core_language/chapter10_conversion_assertion_switch/examples/01_conversions/main.go
// CHAPTER: 10 — Type Conversion, Assertion, Switch
// TOPIC: Conversion (T(x)) — compile-time, concrete-to-concrete.
//
// Run (from the chapter folder):
//   go run ./examples/01_conversions

package main

import "fmt"

// Celsius is a named type with underlying type float64. Named types and
// their underlying type convert freely; the type system uses the name to
// catch misuse at compile time.
type Celsius float64

// Fahrenheit is a separate named type. Even though both are float64
// underneath, you cannot assign one to the other without conversion.
type Fahrenheit float64

func (c Celsius) ToFahrenheit() Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

func main() {
	// ─── Numeric conversions: lossy is allowed, no warning ────────────────
	var big int64 = 300
	var small int8 = int8(big)
	fmt.Printf("int64 300 → int8 %d (truncated, sign bit set)\n", small)

	// Sign change:
	var n int32 = -1
	var u uint32 = uint32(n)
	fmt.Printf("int32 -1 → uint32 %d\n", u)

	// Float → int truncates toward zero:
	f := 3.7
	i := int(f)
	fmt.Printf("float64 3.7 → int %d (truncates, no rounding)\n", i)

	// ─── Named types: underlying type convertible, peers not ──────────────
	temp := Celsius(98.6)
	fl := float64(temp)         // OK — underlying type
	temp2 := Celsius(fl)        // OK
	fahr := temp.ToFahrenheit() // OK — method
	fmt.Printf("Celsius %v = %v as float64; %v in Fahrenheit\n", temp, fl, fahr)
	_ = temp2

	// This would NOT compile:
	//   var x Fahrenheit = temp  // error: cannot use temp (Celsius) as Fahrenheit
	// You must convert through the underlying type:
	x := Fahrenheit(temp)
	fmt.Printf("Manual: Celsius(98.6) reinterpreted as Fahrenheit (no math!) = %v\n", x)

	// ─── string ↔ []byte conversion always copies ────────────────────────
	s := "hello"
	b := []byte(s)         // copy
	b[0] = 'H'              // mutating bytes is fine
	s2 := string(b)         // copy back
	fmt.Printf("\nstring → []byte → string: %q → %v → %q\n", s, b, s2)
	fmt.Printf("Original is unchanged: %q\n", s)

	// ─── string ↔ []rune (UTF-8 decode/encode) ───────────────────────────
	greeting := "héllo"
	runes := []rune(greeting)  // decode UTF-8
	runes[1] = 'a'              // change the second rune
	greeting2 := string(runes) // re-encode UTF-8
	fmt.Printf("\nstring → []rune → string: %q → %v → %q\n",
		greeting, runes, greeting2)

	// ─── string(int) — the GOTCHA ─────────────────────────────────────────
	//
	// `go vet` warns about this. It interprets the int as a Unicode code
	// point. Almost never what you want.
	//
	//   wrong := string(65)   // "A", not "65"!
	//
	// The right way:
	correct := fmt.Sprint(65)
	fmt.Printf("\nFormatting 65 as a string: %q (use fmt.Sprint or strconv.Itoa)\n", correct)
}
