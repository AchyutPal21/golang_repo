// FILE: book/part2_core_language/chapter09_type_system/examples/01_int_sizes/main.go
// CHAPTER: 09 — The Type System
// TOPIC: Integer sizes, ranges, and overflow demonstrated.
//
// Run (from the chapter folder):
//   go run ./examples/01_int_sizes
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Newcomers often assume `int` is some default size. Run this to see
//   what your platform actually gives you, plus the named-size types and
//   wraparound on overflow.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math"
	"unsafe"
)

func main() {
	fmt.Println("=== Sizes (bytes) on this platform ===")
	var (
		i   int
		i8  int8
		i16 int16
		i32 int32
		i64 int64
		u   uint
		f32 float32
		f64 float64
		b   bool
		r   rune
		by  byte
	)
	fmt.Printf("  int     %d   (platform-dependent)\n", unsafe.Sizeof(i))
	fmt.Printf("  int8    %d\n", unsafe.Sizeof(i8))
	fmt.Printf("  int16   %d\n", unsafe.Sizeof(i16))
	fmt.Printf("  int32   %d\n", unsafe.Sizeof(i32))
	fmt.Printf("  int64   %d\n", unsafe.Sizeof(i64))
	fmt.Printf("  uint    %d\n", unsafe.Sizeof(u))
	fmt.Printf("  float32 %d\n", unsafe.Sizeof(f32))
	fmt.Printf("  float64 %d\n", unsafe.Sizeof(f64))
	fmt.Printf("  bool    %d\n", unsafe.Sizeof(b))
	fmt.Printf("  rune    %d   (alias for int32)\n", unsafe.Sizeof(r))
	fmt.Printf("  byte    %d   (alias for uint8)\n", unsafe.Sizeof(by))

	fmt.Println("\n=== Ranges ===")
	fmt.Printf("  int8    %d..%d\n", math.MinInt8, math.MaxInt8)
	fmt.Printf("  int16   %d..%d\n", math.MinInt16, math.MaxInt16)
	fmt.Printf("  int32   %d..%d\n", math.MinInt32, math.MaxInt32)
	fmt.Printf("  int64   %d..%d\n", math.MinInt64, math.MaxInt64)
	fmt.Printf("  uint8   0..%d\n", math.MaxUint8)
	fmt.Printf("  uint16  0..%d\n", math.MaxUint16)
	fmt.Printf("  uint32  0..%d\n", math.MaxUint32)

	fmt.Println("\n=== Silent overflow demonstration ===")
	fmt.Println("(Go integers wrap on overflow — no panic, no exception)")

	maxI8 := int8(math.MaxInt8)
	fmt.Printf("  int8(127) + 1 = %d\n", maxI8+1) // wraps to -128

	var u8 uint8 = 0
	u8--
	fmt.Printf("  uint8(0) - 1 = %d\n", u8) // wraps to 255

	var u32 uint32 = math.MaxUint32
	u32++
	fmt.Printf("  uint32(MAX) + 1 = %d\n", u32) // wraps to 0

	fmt.Println("\n=== Lesson ===")
	fmt.Println("Validate inputs at boundaries; use math/bits or math/big for")
	fmt.Println("checked arithmetic when overflow is plausible.")
}
