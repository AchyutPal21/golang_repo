// FILE: book/part2_core_language/chapter16_pointers/examples/02_escape_analysis/main.go
// CHAPTER: 16 — Pointers and Memory Addressing
// TOPIC: Stack vs heap allocation, escape analysis, how to read
//        -gcflags="-m" output, and when allocations matter.
//
// Run (from the chapter folder):
//   go run ./examples/02_escape_analysis
//
// To see escape analysis decisions:
//   go build -gcflags="-m" ./examples/02_escape_analysis
//
// To see verbose decisions:
//   go build -gcflags="-m -m" ./examples/02_escape_analysis

package main

import "fmt"

// stackAlloc: p does NOT escape — the compiler can keep it on the stack.
// (The value is used only within this function's frame.)
func stackAlloc() int {
	x := 42 // likely stays on stack
	p := &x
	return *p
}

// heapAlloc: the pointer escapes the function — x must be on the heap
// so it survives after stackAlloc2 returns.
func heapAlloc() *int {
	x := 42 // escapes to heap: address is returned
	return &x
}

// interfaceEscape: passing a value to an interface causes it to escape,
// because the interface stores a pointer internally.
func interfaceEscape() {
	x := 42
	var i interface{} = x // x may escape — interface boxing
	fmt.Println(i)
}

// bigStruct: large values often escape because keeping them on the stack
// would overflow it or prevent inlining.
type BigStruct struct {
	data [1024]int64
}

func makeBigStruct() *BigStruct {
	b := BigStruct{}      // escapes to heap
	b.data[0] = 99
	return &b
}

// closureEscape: a variable captured by a closure that outlives the
// enclosing function must escape to the heap.
func closureEscape() func() int {
	count := 0 // escapes: captured by returned closure
	return func() int {
		count++
		return count
	}
}

// noEscape: the closure does not outlive the function, so count can
// stay on the stack.
func noEscape() int {
	count := 0
	add := func(n int) { count += n } // does not escape
	add(1)
	add(2)
	return count
}

func main() {
	fmt.Println("stackAlloc:", stackAlloc())

	p := heapAlloc()
	fmt.Println("heapAlloc:", *p)

	interfaceEscape()

	big := makeBigStruct()
	fmt.Println("bigStruct[0]:", big.data[0])

	counter := closureEscape()
	fmt.Println("closureEscape:", counter(), counter())

	fmt.Println("noEscape:", noEscape())

	fmt.Println()
	fmt.Println("To inspect escape decisions, run:")
	fmt.Println("  go build -gcflags=\"-m\" ./examples/02_escape_analysis")
}
