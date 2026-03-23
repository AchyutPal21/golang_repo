// FILE: 01_fundamentals/03_types_integers/03_types_integers.go
// TOPIC: Integer Types — Every int type, size, range, overflow, byte/rune
//
// Run: go run 01_fundamentals/03_types_integers/03_types_integers.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Choosing the wrong integer type is a source of overflow bugs, performance
//   problems, and API incompatibilities. Most languages hide this behind one
//   "number" type. Go exposes the hardware reality deliberately.
//   Every system programmer needs to know these cold.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math"
	"unsafe"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Integer Types")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// SIGNED INTEGERS — Can hold negative and positive values
	// ─────────────────────────────────────────────────────────────────────
	//
	// int8   = 8 bits  = 1 byte  → range: -128 to 127
	// int16  = 16 bits = 2 bytes → range: -32,768 to 32,767
	// int32  = 32 bits = 4 bytes → range: -2,147,483,648 to 2,147,483,647
	// int64  = 64 bits = 8 bytes → range: -9.2×10^18 to 9.2×10^18
	//
	// Formula: for N bits, range is -2^(N-1) to 2^(N-1) - 1
	// Why "-1" on the positive side? One bit encodes the sign.
	// Two's complement representation (the standard on all modern hardware).

	var i8 int8 = 127
	var i16 int16 = 32767
	var i32 int32 = 2147483647
	var i64 int64 = 9223372036854775807

	fmt.Println("\n── Signed integers ──")
	fmt.Printf("  int8  max = %d  (size: %d bytes)\n", i8, unsafe.Sizeof(i8))
	fmt.Printf("  int16 max = %d  (size: %d bytes)\n", i16, unsafe.Sizeof(i16))
	fmt.Printf("  int32 max = %d  (size: %d bytes)\n", i32, unsafe.Sizeof(i32))
	fmt.Printf("  int64 max = %d  (size: %d bytes)\n", i64, unsafe.Sizeof(i64))

	// math package provides the min/max constants:
	fmt.Printf("\n  math.MinInt8  = %d,  math.MaxInt8  = %d\n", math.MinInt8, math.MaxInt8)
	fmt.Printf("  math.MinInt16 = %d, math.MaxInt16 = %d\n", math.MinInt16, math.MaxInt16)
	fmt.Printf("  math.MinInt32 = %d, math.MaxInt32 = %d\n", math.MinInt32, math.MaxInt32)
	fmt.Printf("  math.MinInt64 = %d, math.MaxInt64 = %d\n", math.MinInt64, math.MaxInt64)

	// ─────────────────────────────────────────────────────────────────────
	// UNSIGNED INTEGERS — Only non-negative values, double the positive range
	// ─────────────────────────────────────────────────────────────────────
	//
	// uint8   = 0 to 255
	// uint16  = 0 to 65,535
	// uint32  = 0 to 4,294,967,295
	// uint64  = 0 to 18,446,744,073,709,551,615
	//
	// USE UNSIGNED WHEN:
	//   - Value is inherently non-negative (array index, byte count, bit flags)
	//   - You need the full bit width for large positive values
	//   - Working with bit manipulation
	//
	// BEWARE:
	//   Subtracting from a uint that results in negative → wraps to huge number!
	//   var x uint8 = 0; x--  → x becomes 255 (wraps around)

	var u8 uint8 = 255
	var u16 uint16 = 65535
	var u32 uint32 = 4294967295
	var u64 uint64 = 18446744073709551615

	fmt.Println("\n── Unsigned integers ──")
	fmt.Printf("  uint8  max = %d  (size: %d bytes)\n", u8, unsafe.Sizeof(u8))
	fmt.Printf("  uint16 max = %d  (size: %d bytes)\n", u16, unsafe.Sizeof(u16))
	fmt.Printf("  uint32 max = %d  (size: %d bytes)\n", u32, unsafe.Sizeof(u32))
	fmt.Printf("  uint64 max = %d  (size: %d bytes)\n", u64, unsafe.Sizeof(u64))

	// ─────────────────────────────────────────────────────────────────────
	// int and uint — Platform-dependent size (the "default" integer types)
	// ─────────────────────────────────────────────────────────────────────
	//
	// int  and uint  are EITHER 32 or 64 bits depending on the platform.
	//   On a 64-bit OS/CPU (almost everything today): int = int64
	//   On a 32-bit OS/CPU (embedded, old hardware): int = int32
	//
	// WHY EXISTS:
	//   int is the "natural" integer type for the CPU. Loop counters, array
	//   indices, lengths — these should use int because the CPU handles them
	//   natively at its word size. Using int64 explicitly on a 32-bit CPU
	//   requires two operations per arithmetic step.
	//
	// RULE OF THUMB:
	//   - Use int for general-purpose integers (loop counters, lengths)
	//   - Use int64/uint64 when you need a guaranteed 64-bit range
	//   - Use int32/uint32 when talking to C code or binary protocols
	//   - Use int8/uint8 for memory-constrained data (large arrays of small values)

	var n int = 1000
	var m uint = 1000
	fmt.Printf("\n── int and uint (platform-dependent) ──\n")
	fmt.Printf("  int  size: %d bytes, value: %d\n", unsafe.Sizeof(n), n)
	fmt.Printf("  uint size: %d bytes, value: %d\n", unsafe.Sizeof(m), m)

	// ─────────────────────────────────────────────────────────────────────
	// SPECIAL TYPES: byte and rune
	// ─────────────────────────────────────────────────────────────────────
	//
	// byte  = alias for uint8  → used for raw binary data, file I/O, network bytes
	// rune  = alias for int32  → used for Unicode code points (a "character")
	//
	// WHY TWO SEPARATE NAMES?
	//   Go strings are sequences of BYTES (UTF-8 encoded).
	//   When you want to talk about CHARACTERS (Unicode code points), use rune.
	//   Using byte signals "I'm treating this as raw binary data".
	//   Using rune signals "I'm treating this as a Unicode character".
	//
	// A rune can represent any Unicode character:
	//   'A'  → U+0041 → rune value 65
	//   '€'  → U+20AC → rune value 8364
	//   '界' → U+754C → rune value 30028

	var b byte = 'A'       // 'A' as a byte (ASCII value 65)
	var r rune = '界'       // Chinese character, Unicode code point 30028
	var r2 rune = '\u20AC' // Euro sign via Unicode escape

	fmt.Println("\n── byte and rune ──")
	fmt.Printf("  byte 'A'    = %d  (char: %c)  type: %T\n", b, b, b)
	fmt.Printf("  rune '界'   = %d  (char: %c)  type: %T\n", r, r, r)
	fmt.Printf("  rune '\\u20AC' = %d (char: %c)  type: %T\n", r2, r2, r2)

	// ─────────────────────────────────────────────────────────────────────
	// INTEGER OVERFLOW — What happens when you exceed the range
	// ─────────────────────────────────────────────────────────────────────
	//
	// Go does NOT panic on integer overflow. It silently WRAPS around.
	// This matches hardware behavior (2's complement arithmetic).
	//
	// DETECTION: The math package has overflow-safe functions for int64.
	// For critical code (financial, crypto), always check for overflow explicitly.
	//
	// Example: int8 max is 127. Adding 1 to 127 wraps to -128.
	// This is called "two's complement overflow" — the bit pattern wraps.

	var maxInt8 int8 = 127
	maxInt8++ // wraps: 127 + 1 = -128 in int8
	fmt.Printf("\n── Overflow (silent wrap) ──\n")
	fmt.Printf("  int8(127) + 1 = %d  (wrapped to -128!)\n", maxInt8)

	var maxUint8 uint8 = 255
	maxUint8++ // wraps: 255 + 1 = 0 in uint8
	fmt.Printf("  uint8(255) + 1 = %d (wrapped to 0!)\n", maxUint8)

	// ─────────────────────────────────────────────────────────────────────
	// TYPE CONVERSION — Go is STRICTLY typed, no implicit conversions
	// ─────────────────────────────────────────────────────────────────────
	//
	// Unlike C/Java, Go NEVER automatically converts between numeric types.
	// int32 + int64 is a COMPILE ERROR. You must explicitly convert.
	//
	// Syntax: T(value)
	//
	// WHY STRICT TYPING?
	//   Implicit conversions hide bugs. If you convert int64 → int32,
	//   you might lose data. Go forces you to be explicit about this.

	var small int8 = 100
	var big int64 = int64(small) // explicit conversion int8 → int64 (safe, no data loss)
	var small2 int8 = int8(big)  // explicit conversion int64 → int8 (potentially lossy!)
	fmt.Printf("\n── Type conversion ──\n")
	fmt.Printf("  int8(%d) → int64(%d) → int8(%d)\n", small, big, small2)

	// Narrowing conversion with data loss:
	var large int32 = 300
	var narrow int8 = int8(large) // 300 doesn't fit in int8 (max 127), truncates bits
	fmt.Printf("  int32(%d) → int8(%d)  ← DATA LOSS! truncated\n", large, narrow)

	// ─────────────────────────────────────────────────────────────────────
	// UINTPTR — Special type for pointer arithmetic
	// ─────────────────────────────────────────────────────────────────────
	//
	// uintptr is an integer large enough to hold a pointer value.
	// It's the same size as unsafe.Pointer but is an INTEGER, not a pointer.
	//
	// USE CASES:
	//   - Interfacing with C via cgo
	//   - Low-level memory operations with unsafe package
	//   - Storing pointer-sized integers (like handles from OS APIs)
	//
	// DANGER: The garbage collector does NOT trace uintptr values.
	// If you store a pointer as uintptr, the GC might move the object
	// and the uintptr becomes a dangling reference. Only use with care.
	//
	// For normal Go code: never use uintptr.

	var ptr uintptr = 0xDEADBEEF
	fmt.Printf("\n── uintptr (for low-level use only) ──\n")
	fmt.Printf("  uintptr size: %d bytes, value: 0x%X\n", unsafe.Sizeof(ptr), ptr)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  int/uint    → default, platform word size (use for most things)")
	fmt.Println("  int8..int64 → explicit sizes for protocols, memory efficiency")
	fmt.Println("  byte        → uint8 alias, raw binary data")
	fmt.Println("  rune        → int32 alias, Unicode code points")
	fmt.Println("  Overflow: silent wrap, watch out!")
	fmt.Println("  Conversion: always explicit, T(value)")
}
