// FILE: book/part6_production_engineering/chapter87_performance_patterns/examples/02_zero_alloc/main.go
// CHAPTER: 87 — Performance Patterns
// TOPIC: Zero-allocation patterns — stack vs heap, string/[]byte conversion,
//        pre-allocated slices, strconv vs fmt, unsafe string tricks.
//
// Run:
//   go run ./part6_production_engineering/chapter87_performance_patterns/examples/02_zero_alloc

package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// ─────────────────────────────────────────────────────────────────────────────
// STRING ↔ []BYTE WITHOUT ALLOCATION
// ─────────────────────────────────────────────────────────────────────────────

// stringToBytes converts a string to []byte without copying.
// ONLY safe when the []byte is not mutated and the string outlives the slice.
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// bytesToString converts []byte to string without copying.
// ONLY safe when the []byte is not mutated after the conversion.
func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

// ─────────────────────────────────────────────────────────────────────────────
// STRCONV vs FMT — avoid allocations in hot paths
// ─────────────────────────────────────────────────────────────────────────────

func appendIntSlow(dst []byte, n int) []byte {
	// fmt.Sprintf always allocates a new string.
	s := fmt.Sprintf("%d", n)
	return append(dst, s...)
}

func appendIntFast(dst []byte, n int) []byte {
	// strconv.AppendInt writes directly into dst — zero alloc.
	return strconv.AppendInt(dst, int64(n), 10)
}

func appendFloatFast(dst []byte, f float64) []byte {
	return strconv.AppendFloat(dst, f, 'f', 2, 64)
}

// ─────────────────────────────────────────────────────────────────────────────
// STRINGS.BUILDER vs + CONCATENATION
// ─────────────────────────────────────────────────────────────────────────────

func concatSlow(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p // each += allocates a new string
	}
	return result
}

func concatFast(parts []string) string {
	var sb strings.Builder
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	sb.Grow(total) // pre-size to avoid internal reallocations
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// PRE-ALLOCATED SLICES
// ─────────────────────────────────────────────────────────────────────────────

func collectSlow(n int) []int {
	var result []int // starts nil — will grow with every doubling
	for i := 0; i < n; i++ {
		result = append(result, i*i)
	}
	return result
}

func collectFast(n int) []int {
	result := make([]int, 0, n) // exact capacity upfront
	for i := 0; i < n; i++ {
		result = append(result, i*i)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// STACK VS HEAP — keeping allocations on the stack
// ─────────────────────────────────────────────────────────────────────────────

// Point stays on the stack when returned by value.
type Point struct{ X, Y float64 }

func newPointValue() Point      { return Point{3.14, 2.71} } // stack
func newPointPointer() *Point   { return &Point{3.14, 2.71} } // heap

// ─────────────────────────────────────────────────────────────────────────────
// MEASUREMENT
// ─────────────────────────────────────────────────────────────────────────────

type stats struct{ allocs, bytes uint64 }

func snap() stats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return stats{m.Mallocs, m.TotalAlloc}
}
func (a stats) diff(b stats) stats { return stats{b.allocs - a.allocs, b.bytes - a.bytes} }

func measure(label string, fn func()) stats {
	runtime.GC()
	a := snap()
	fn()
	return a.diff(snap())
}

const N = 50_000

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 87: Zero-Allocation Patterns ===")
	fmt.Println()

	// ── UNSAFE STRING/BYTE CONVERSION ─────────────────────────────────────────
	fmt.Println("--- unsafe string/[]byte conversion ---")
	s := "hello, world"
	b := stringToBytes(s)
	s2 := bytesToString(b)
	fmt.Printf("  original: %q  len=%d\n", s, len(s))
	fmt.Printf("  as bytes: %v\n", b[:5])
	fmt.Printf("  back:     %q\n", s2)
	fmt.Println("  WARNING: only safe with immutable data & same lifetime.")
	fmt.Println()

	// ── STRCONV vs FMT ────────────────────────────────────────────────────────
	fmt.Println("--- strconv.AppendInt vs fmt.Sprintf ---")
	buf := make([]byte, 0, 64)
	buf = appendIntFast(buf, 12345)
	buf = append(buf, ' ')
	buf = appendFloatFast(buf, 3.14159)
	fmt.Printf("  appended: %s\n", buf)

	t1 := time.Now()
	buf2 := make([]byte, 0, 64)
	for i := 0; i < N; i++ {
		buf2 = buf2[:0]
		buf2 = appendIntSlow(buf2, i)
	}
	d1 := time.Since(t1)

	t2 := time.Now()
	buf3 := make([]byte, 0, 64)
	for i := 0; i < N; i++ {
		buf3 = buf3[:0]
		buf3 = appendIntFast(buf3, i)
	}
	d2 := time.Since(t2)

	s1 := measure("fmt.Sprintf int", func() {
		buf4 := make([]byte, 0, 64)
		for i := 0; i < N; i++ {
			buf4 = buf4[:0]
			buf4 = appendIntSlow(buf4, i)
		}
	})
	s2stat := measure("strconv.AppendInt", func() {
		buf4 := make([]byte, 0, 64)
		for i := 0; i < N; i++ {
			buf4 = buf4[:0]
			buf4 = appendIntFast(buf4, i)
		}
	})

	fmt.Printf("  fmt.Sprintf:       %6v  allocs=%d  bytes=%d\n", d1.Round(time.Microsecond), s1.allocs, s1.bytes)
	fmt.Printf("  strconv.AppendInt: %6v  allocs=%d  bytes=%d\n", d2.Round(time.Microsecond), s2stat.allocs, s2stat.bytes)
	fmt.Println()

	// ── STRING CONCATENATION ──────────────────────────────────────────────────
	fmt.Println("--- string concatenation ---")
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = strconv.Itoa(i)
	}
	sc := measure("concat slow (+=)", func() {
		for i := 0; i < 1000; i++ {
			_ = concatSlow(parts)
		}
	})
	sf := measure("concat fast (Builder)", func() {
		for i := 0; i < 1000; i++ {
			_ = concatFast(parts)
		}
	})
	fmt.Printf("  concat slow: allocs=%d bytes=%d\n", sc.allocs, sc.bytes)
	fmt.Printf("  concat fast: allocs=%d bytes=%d\n", sf.allocs, sf.bytes)
	fmt.Println()

	// ── PRE-ALLOCATED SLICES ──────────────────────────────────────────────────
	fmt.Println("--- pre-allocated slices ---")
	ps := measure("append without cap", func() {
		for i := 0; i < 1000; i++ {
			_ = collectSlow(200)
		}
	})
	pf := measure("append with cap", func() {
		for i := 0; i < 1000; i++ {
			_ = collectFast(200)
		}
	})
	fmt.Printf("  without cap: allocs=%d bytes=%d\n", ps.allocs, ps.bytes)
	fmt.Printf("  with cap:    allocs=%d bytes=%d\n", pf.allocs, pf.bytes)
	fmt.Println()

	// ── STACK VS HEAP ─────────────────────────────────────────────────────────
	fmt.Println("--- stack vs heap allocation ---")
	sv := measure("return by value (stack)", func() {
		for i := 0; i < N; i++ {
			p := newPointValue()
			_ = p.X + p.Y
		}
	})
	sh := measure("return pointer (heap)", func() {
		for i := 0; i < N; i++ {
			p := newPointPointer()
			_ = p.X + p.Y
		}
	})
	fmt.Printf("  by value:  allocs=%d bytes=%d\n", sv.allocs, sv.bytes)
	fmt.Printf("  by pointer: allocs=%d bytes=%d\n", sh.allocs, sh.bytes)
	fmt.Println()

	fmt.Println("Zero-alloc checklist:")
	fmt.Println("  - Use strconv.Append* instead of fmt.Sprintf in hot paths")
	fmt.Println("  - Pre-size slices with make([]T, 0, cap)")
	fmt.Println("  - Use strings.Builder with Grow() for string building")
	fmt.Println("  - Prefer value receivers for small structs")
	fmt.Println("  - Use unsafe conversions only when provably safe")
}
