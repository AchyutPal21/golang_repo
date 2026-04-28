// FILE: book/part2_core_language/chapter16_pointers/examples/03_pointer_patterns/main.go
// CHAPTER: 16 — Pointers and Memory Addressing
// TOPIC: Optional values via *T, in-place update, pointer receivers,
//        pointer to interface (antipattern), unsafe.Sizeof.
//
// Run (from the chapter folder):
//   go run ./examples/03_pointer_patterns

package main

import (
	"fmt"
	"unsafe"
)

// --- Optional values using *T ---

// Config uses pointer fields to distinguish "not set" from zero value.
type Config struct {
	Host    string
	Port    *int  // nil means "use default"; 0 would mean "bind to random port"
	Debug   *bool // nil means "inherit from environment"
	Workers *int
}

func intPtr(n int) *int   { return &n }
func boolPtr(b bool) *bool { return &b }

func resolveConfig(c Config) Config {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == nil {
		c.Port = intPtr(8080)
	}
	if c.Debug == nil {
		c.Debug = boolPtr(false)
	}
	if c.Workers == nil {
		c.Workers = intPtr(4)
	}
	return c
}

// --- In-place update ---

type Stats struct {
	Hits   int64
	Misses int64
}

func (s *Stats) RecordHit()  { s.Hits++ }
func (s *Stats) RecordMiss() { s.Misses++ }
func (s *Stats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// --- Pointer to interface: an antipattern ---

type Stringer interface {
	String() string
}

type myStr struct{ v string }

func (m myStr) String() string { return m.v }

// WRONG: taking the address of an interface value.
// Callers rarely need *Stringer; pass Stringer directly.
func printBad(s *Stringer) {
	fmt.Println("via *Stringer:", (*s).String())
}

// CORRECT: pass the interface value directly.
func printGood(s Stringer) {
	fmt.Println("via Stringer:", s.String())
}

// --- unsafe.Sizeof / Alignof ---

type alignDemo struct {
	A bool    // 1 byte
	B int64   // 8 bytes — but alignment forces 7 bytes padding after A
	C uint8   // 1 byte
	D float32 // 4 bytes
}

type alignPacked struct {
	A uint8   // 1
	C uint8   // 1
	D float32 // 4
	B int64   // 8
}

func main() {
	// --- optional pointer fields ---
	partial := Config{Host: "api.example.com", Debug: boolPtr(true)}
	full := resolveConfig(partial)
	fmt.Printf("host=%s port=%d debug=%v workers=%d\n",
		full.Host, *full.Port, *full.Debug, *full.Workers)

	fmt.Println()

	// --- in-place update ---
	var s Stats
	for range 7 {
		s.RecordHit()
	}
	for range 3 {
		s.RecordMiss()
	}
	fmt.Printf("hits=%d misses=%d rate=%.1f%%\n", s.Hits, s.Misses, s.HitRate()*100)

	fmt.Println()

	// --- pointer to interface ---
	m := myStr{"hello"}
	var iface Stringer = m

	printBad(&iface)  // technically works but never do this
	printGood(iface)  // idiomatic

	fmt.Println()

	// --- unsafe.Sizeof and struct layout ---
	fmt.Printf("alignDemo   size=%d align=%d\n",
		unsafe.Sizeof(alignDemo{}), unsafe.Alignof(alignDemo{}))
	fmt.Printf("alignPacked size=%d align=%d\n",
		unsafe.Sizeof(alignPacked{}), unsafe.Alignof(alignPacked{}))
	fmt.Println("(reordering fields to larger-first saves", unsafe.Sizeof(alignDemo{})-unsafe.Sizeof(alignPacked{}), "bytes per struct)")

	fmt.Println()

	// --- pointer arithmetic is not allowed in Go ---
	// The following does not compile:
	// p := &x
	// p++ // invalid operation
	//
	// Use unsafe.Add for legitimate cases (cgo, unsafe memory access).
	// For ordinary Go: use slices and indices.
	fmt.Println("Go has no pointer arithmetic outside unsafe package")
}
