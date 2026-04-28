// FILE: book/part2_core_language/chapter08_variables_constants_zero/examples/02_zero_values/main.go
// CHAPTER: 08 — Variables, Constants, and the Zero Value
// TOPIC: The zero value of every type, demonstrated.
//
// Run (from the chapter folder):
//   go run ./examples/02_zero_values
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Idiomatic Go relies on zero values being useful. This program prints
//   the zero value of every built-in type plus a struct, an array, an
//   interface, a function, and a channel. Use it as a lookup whenever you
//   forget what `var x map[string]int` gives you.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// Person is here just to demonstrate that struct zero values are
// "every field at its own zero value, recursively."
type Person struct {
	Name    string
	Age     int
	Tags    []string
	Friends map[string]*Person
}

// Speaker is an interface — its zero value is a nil interface.
type Speaker interface {
	Speak() string
}

func main() {
	fmt.Println("=== Zero values by type ===")

	// Numeric zeros
	var i int
	var i32 int32
	var i64 int64
	var u uint
	var f32 float32
	var f64 float64
	var c64 complex64
	fmt.Printf("int       %v\n", i)
	fmt.Printf("int32     %v\n", i32)
	fmt.Printf("int64     %v\n", i64)
	fmt.Printf("uint      %v\n", u)
	fmt.Printf("float32   %v\n", f32)
	fmt.Printf("float64   %v\n", f64)
	fmt.Printf("complex64 %v\n", c64)

	// Bool, string
	var b bool
	var s string
	fmt.Printf("bool      %v\n", b)
	fmt.Printf("string    %q\n", s) // %q to make "" visible

	// Pointer
	var p *int
	fmt.Printf("*int      %v (== nil? %v)\n", p, p == nil)

	// Slice — nil but ranges, len()'s, and append()'s like an empty slice
	var sl []int
	fmt.Printf("[]int     %v (len=%d, cap=%d, nil=%v)\n",
		sl, len(sl), cap(sl), sl == nil)
	fmt.Printf("            range works on nil slice? ")
	for range sl {
		_ = "won't print"
	}
	fmt.Println("yes (just doesn't iterate)")

	// Map — nil; reads return zero, writes PANIC
	var m map[string]int
	fmt.Printf("map[K]V   %v (len=%d, nil=%v)\n", m, len(m), m == nil)
	v, ok := m["missing"] // legal: returns zero, false
	fmt.Printf("            m[\"missing\"] → (%d, %v)  (read is fine)\n", v, ok)
	// m["k"] = 1  // ← would PANIC: assignment to entry in nil map

	// Channel — nil; sends and receives BLOCK FOREVER
	var ch chan int
	fmt.Printf("chan T    %v (nil=%v) — sends/receives would block forever\n",
		ch, ch == nil)

	// Function — nil; calling panics
	var fn func(int) int
	fmt.Printf("func      <nil> (nil=%v) — calling would panic\n", fn == nil)
	_ = fn // silence unused

	// Interface — nil; calling a method panics
	var sp Speaker
	fmt.Printf("interface %v (nil=%v) — calling would panic\n", sp, sp == nil)

	// Array — every element zero-valued
	var arr [3]int
	fmt.Printf("[3]int    %v\n", arr)

	// Struct — every field zero-valued, recursively
	var person Person
	fmt.Printf("struct    %+v\n", person)

	fmt.Println("\n=== Zero-value tricks ===")

	// `bytes.Buffer{}` is usable at zero (no allocation needed)
	// `sync.Mutex{}` is usable at zero (unlocked)
	// `sync.WaitGroup{}` is usable at zero (zero pending)
	// We can't import them here without making this file complicated, so
	// trust me: idiomatic Go relies on this everywhere.
	fmt.Println("bytes.Buffer{}     — usable at zero (writes work without setup)")
	fmt.Println("sync.Mutex{}       — usable at zero (unlocked)")
	fmt.Println("sync.WaitGroup{}   — usable at zero (zero pending)")

	// The two cases where zero-value is NOT enough:
	fmt.Println("\nzero is NOT enough for:")
	fmt.Println("  channels — must call make(chan T)")
	fmt.Println("  maps     — must call make(map[K]V) before writing")
}
