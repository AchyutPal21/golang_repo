// FILE: book/part6_production_engineering/chapter88_gc_escape/examples/01_escape_analysis/main.go
// CHAPTER: 88 — GC & Escape Analysis
// TOPIC: Escape analysis — what causes values to escape to the heap, how to
//        read compiler output, and patterns that keep values on the stack.
//
// Build with escape annotations:
//   go build -gcflags="-m -m" ./part6_production_engineering/chapter88_gc_escape/examples/01_escape_analysis/
// Run:
//   go run ./part6_production_engineering/chapter88_gc_escape/examples/01_escape_analysis/

package main

import (
	"fmt"
	"runtime"
	"strconv"
)

// ─────────────────────────────────────────────────────────────────────────────
// CAUSE 1: returning a pointer to a local variable
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	Host    string
	Port    int
	Timeout int
}

// returnsPointer — Config escapes: caller holds pointer after function returns.
func returnsPointer() *Config {
	c := Config{Host: "localhost", Port: 8080, Timeout: 30}
	return &c // ← escapes to heap
}

// returnsValue — Config stays on the caller's stack frame.
func returnsValue() Config {
	return Config{Host: "localhost", Port: 8080, Timeout: 30} // ← stack
}

// ─────────────────────────────────────────────────────────────────────────────
// CAUSE 2: storing into an interface
// ─────────────────────────────────────────────────────────────────────────────

// assignToInterface — any value stored in an interface escapes.
func assignToInterface(n int) any {
	return n // int escapes: boxed into interface{} on heap
}

// noInterface — stays on stack.
func noInterface(n int) string {
	return strconv.Itoa(n) // only the string header might escape
}

// ─────────────────────────────────────────────────────────────────────────────
// CAUSE 3: closures capturing local variables
// ─────────────────────────────────────────────────────────────────────────────

func makeClosure() func() int {
	x := 42 // x escapes: the closure outlives this stack frame
	return func() int { return x }
}

// ─────────────────────────────────────────────────────────────────────────────
// CAUSE 4: slice/map growth beyond compile-time-known size
// ─────────────────────────────────────────────────────────────────────────────

// smallSlice — compiler may keep on stack if size is constant and small.
func smallSliceOnStack() [8]int {
	var arr [8]int // fixed array — stays on stack
	for i := range arr {
		arr[i] = i * i
	}
	return arr
}

// dynamicSlice — backing array is always on the heap (make).
func dynamicSlice(n int) []int {
	s := make([]int, n) // heap allocation
	for i := range s {
		s[i] = i
	}
	return s
}

// ─────────────────────────────────────────────────────────────────────────────
// CAUSE 5: fmt.Println forces arguments to escape
// ─────────────────────────────────────────────────────────────────────────────

// fmtEscapes — every argument passed to variadic ...any escapes.
func fmtEscapes(n int) {
	fmt.Println(n) // n escapes to heap here
}

// noEscapePrint — using strconv + direct write avoids boxing.
func noEscapePrint(buf []byte, n int) []byte {
	return strconv.AppendInt(buf, int64(n), 10)
}

// ─────────────────────────────────────────────────────────────────────────────
// ESCAPE ANALYSIS SUMMARY TABLE
// ─────────────────────────────────────────────────────────────────────────────

type escapeCase struct {
	pattern string
	escapes bool
	why     string
}

var escapeCases = []escapeCase{
	{"return &localVar", true, "pointer outlives the function's stack frame"},
	{"interface{}(value)", true, "interface needs heap box for type info"},
	{"closure captures var", true, "closure outlives stack frame of defining fn"},
	{"make([]T, n) (dynamic n)", true, "size unknown at compile time"},
	{"[8]int (fixed array)", false, "size known, no pointer escapes"},
	{"return Config{} by value", false, "caller owns the copy"},
	{"strconv.AppendInt(buf,n)", false, "no boxing, buf is caller-owned"},
	{"func(n int) string", false, "string built from stack bytes"},
}

// ─────────────────────────────────────────────────────────────────────────────
// MEASURING HEAP ALLOCATIONS
// ─────────────────────────────────────────────────────────────────────────────

type snap struct{ allocs, bytes uint64 }

func readSnap() snap {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return snap{m.Mallocs, m.TotalAlloc}
}
func (a snap) diff(b snap) snap { return snap{b.allocs - a.allocs, b.bytes - a.bytes} }

const iters = 100_000

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 88: Escape Analysis ===")
	fmt.Println()

	// ── ESCAPE CASES TABLE ────────────────────────────────────────────────────
	fmt.Println("--- Escape cause reference table ---")
	fmt.Printf("  %-38s  %-7s  %s\n", "Pattern", "Escapes", "Reason")
	fmt.Printf("  %s\n", strconv.Itoa(0)[:0]+
		"--------------------------------------  -------  -----------------------")
	for _, c := range escapeCases {
		e := "NO"
		if c.escapes {
			e = "YES"
		}
		fmt.Printf("  %-38s  %-7s  %s\n", c.pattern, e, c.why)
	}
	fmt.Println()

	// ── ALLOCATION MEASUREMENT: POINTER vs VALUE ──────────────────────────────
	fmt.Println("--- returnsPointer vs returnsValue ---")
	runtime.GC()
	s1 := readSnap()
	for i := 0; i < iters; i++ {
		_ = returnsPointer()
	}
	d1 := s1.diff(readSnap())

	runtime.GC()
	s2 := readSnap()
	for i := 0; i < iters; i++ {
		c := returnsValue()
		_ = c
	}
	d2 := s2.diff(readSnap())
	fmt.Printf("  returnsPointer: allocs=%d bytes=%d\n", d1.allocs, d1.bytes)
	fmt.Printf("  returnsValue:   allocs=%d bytes=%d\n", d2.allocs, d2.bytes)
	fmt.Println()

	// ── INTERFACE BOXING ──────────────────────────────────────────────────────
	fmt.Println("--- interface boxing ---")
	runtime.GC()
	s3 := readSnap()
	for i := 0; i < iters; i++ {
		_ = assignToInterface(i)
	}
	d3 := s3.diff(readSnap())

	runtime.GC()
	s4 := readSnap()
	for i := 0; i < iters; i++ {
		_ = noInterface(i)
	}
	d4 := s4.diff(readSnap())
	fmt.Printf("  assignToInterface: allocs=%d bytes=%d\n", d3.allocs, d3.bytes)
	fmt.Printf("  noInterface:       allocs=%d bytes=%d\n", d4.allocs, d4.bytes)
	fmt.Println()

	// ── HOW TO READ ESCAPE OUTPUT ─────────────────────────────────────────────
	fmt.Println("--- How to read compiler escape output ---")
	output := `$ go build -gcflags="-m" ./...

./main.go:34:2: moved to heap: c          ← returnsPointer
./main.go:47:9: n escapes to heap         ← assignToInterface (boxing)
./main.go:54:3: x escapes to heap         ← closure capture
./main.go:65:14: make([]int, n) escapes   ← dynamic slice

Flags:
  -gcflags="-m"     one level of escape notes
  -gcflags="-m -m"  full decision trace (verbose)

Tips:
  • "moved to heap" = allocation you control
  • "escapes to heap" = caused by another package (e.g. fmt)
  • Fix: change return type, add a sink parameter, avoid interfaces`
	fmt.Println(output)
	fmt.Println()

	// ── CLOSURE DEMO ──────────────────────────────────────────────────────────
	fmt.Println("--- Closure escape ---")
	fn := makeClosure()
	fmt.Printf("  Closure result: %d\n", fn())
	fmt.Println("  x (captured) escaped to heap when makeClosure returned.")
	fmt.Println()

	fmt.Println("Stack vs heap summary:")
	fmt.Println("  Stack: cheap, automatic cleanup, LIFO lifetime")
	fmt.Println("  Heap:  flexible lifetime, GC pressure, cache unfriendly")
	fmt.Println("  Goal: maximize stack usage; push heap to per-request boundaries")
}
