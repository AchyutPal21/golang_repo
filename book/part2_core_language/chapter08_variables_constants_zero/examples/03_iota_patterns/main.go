// FILE: book/part2_core_language/chapter08_variables_constants_zero/examples/03_iota_patterns/main.go
// CHAPTER: 08 — Variables, Constants, and the Zero Value
// TOPIC: Three real-world iota patterns: typed enum, bit flags, unit ladder.
//
// Run (from the chapter folder):
//   go run ./examples/03_iota_patterns
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   `iota` is one of those features people misuse for years before
//   internalizing the patterns. This file shows the three patterns you'll
//   actually want to copy: typed enum (with String method), bit-flag set,
//   and the KB/MB/GB unit ladder. Each is a verbatim production idiom.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"strings"
)

// ─── Pattern 1: Typed enum ──────────────────────────────────────────────────
//
// Use when you have a small set of named values that callers must pick
// from. The custom type makes wrong values impossible at compile time.

type Weekday int

const (
	Sunday Weekday = iota // 0
	Monday                // 1 (implicit "= iota")
	Tuesday               // 2
	Wednesday             // 3
	Thursday              // 4
	Friday                // 5
	Saturday              // 6
)

// String makes Weekday print readably with fmt.Println. Idiomatic for any
// typed-enum: ALWAYS write a String() method.
func (d Weekday) String() string {
	names := [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	if int(d) < 0 || int(d) >= len(names) {
		return fmt.Sprintf("Weekday(%d)", int(d))
	}
	return names[d]
}

// ─── Pattern 2: Bit flags ───────────────────────────────────────────────────
//
// Use when you have INDEPENDENT options that can be combined. Each value is
// a power of two; a "set" is the bitwise OR of its members.

type Permission uint8

const (
	PermRead    Permission = 1 << iota // 1 << 0 = 1
	PermWrite                          // 1 << 1 = 2
	PermExecute                        // 1 << 2 = 4
	PermDelete                         // 1 << 3 = 8
)

// Has reports whether p contains q (i.e. all bits of q are set in p).
func (p Permission) Has(q Permission) bool {
	return p&q == q
}

func (p Permission) String() string {
	if p == 0 {
		return "[none]"
	}
	var parts []string
	if p.Has(PermRead) {
		parts = append(parts, "read")
	}
	if p.Has(PermWrite) {
		parts = append(parts, "write")
	}
	if p.Has(PermExecute) {
		parts = append(parts, "execute")
	}
	if p.Has(PermDelete) {
		parts = append(parts, "delete")
	}
	return "[" + strings.Join(parts, "+") + "]"
}

// ─── Pattern 3: Unit ladder ─────────────────────────────────────────────────
//
// Use when you have powers-of-N constants. The (iota+1)*10 trick is the
// classic way to build KB/MB/GB/TB.

const (
	_  = iota // discard 0 with the blank identifier
	KB = 1 << (10 * iota)
	MB // 1 << 20
	GB // 1 << 30
	TB // 1 << 40
	PB // 1 << 50
)

func main() {
	// ─── Pattern 1 demo ─────────────────────────────────────────────────
	fmt.Println("=== Pattern 1: Typed enum ===")
	for d := Sunday; d <= Saturday; d++ {
		fmt.Printf("  %s = %d\n", d, int(d))
	}
	// Compile-time safety: a function that wants a Weekday cannot accept
	// an arbitrary int. Try uncommenting:
	//   var x int = 3
	//   greetOn(x)  // compile error: cannot use x (type int) as Weekday
	greetOn(Wednesday)

	// ─── Pattern 2 demo ─────────────────────────────────────────────────
	fmt.Println("\n=== Pattern 2: Bit flags ===")
	none := Permission(0)
	all := PermRead | PermWrite | PermExecute | PermDelete
	rw := PermRead | PermWrite

	fmt.Printf("  none = %s\n", none)
	fmt.Printf("  all  = %s\n", all)
	fmt.Printf("  rw   = %s\n", rw)
	fmt.Printf("  rw.Has(PermRead)    = %v\n", rw.Has(PermRead))
	fmt.Printf("  rw.Has(PermExecute) = %v\n", rw.Has(PermExecute))

	// ─── Pattern 3 demo ─────────────────────────────────────────────────
	fmt.Println("\n=== Pattern 3: Unit ladder ===")
	fmt.Printf("  KB = %d bytes\n", KB)
	fmt.Printf("  MB = %d bytes\n", MB)
	fmt.Printf("  GB = %d bytes\n", GB)
	fmt.Printf("  TB = %d bytes\n", TB)

	// Real use: a buffer-size config knob expressed in MB.
	const bufferSize = 64 * MB
	fmt.Printf("\n  bufferSize = 64*MB = %d bytes (%d MB)\n", bufferSize, bufferSize/MB)
}

func greetOn(d Weekday) {
	fmt.Printf("  greetOn(%s) — see you on %s\n", d, d)
}
