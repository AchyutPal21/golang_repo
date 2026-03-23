// 02_closures.go
//
// TOPIC: Closures — Variable Capture, Closure Factories, Stateful Functions,
//        the Loop-Variable Gotcha, and Memoization
//
// A closure is a function value that "closes over" variables from its surrounding
// lexical scope — it carries those variables with it wherever it goes.
//
// HOW IT WORKS INTERNALLY:
//   When Go compiles a closure, it allocates the captured variables on the HEAP
//   (not the stack), even if they were declared as local variables. This is called
//   "escape analysis". The closure holds a reference (pointer) to those heap
//   variables. If you create multiple closures that capture the same variable,
//   they ALL share the same underlying storage — this is the source of the famous
//   loop variable bug.
//
// WHY CLOSURES?
//   Closures let you bundle DATA and BEHAVIOR together without defining a struct.
//   They're the lightweight alternative to single-method interfaces and anonymous
//   inner classes (Java). They enable:
//     - Stateful functions (counter, accumulator)
//     - Function factories (make a function configured at creation time)
//     - Memoization (cache results inside the closure)
//     - Callback customization (pass a function that "remembers" context)

package main

import (
	"fmt"
	"strings"
	"sync"
)

// ─── 1. WHAT IS A CLOSURE? ────────────────────────────────────────────────────
//
// A simple closure: the inner function references 'message' from the outer scope.
// Even after outerFunction returns, the closure keeps 'message' alive on the heap.

func simpleClosure() {
	message := "Hello from the outer scope"

	// 'sayIt' is a closure — it captures 'message' by REFERENCE.
	// If 'message' changes, 'sayIt' will see the new value.
	sayIt := func() {
		fmt.Println(sayIt_description(), message)
	}

	sayIt()
	message = "Message was updated!" // mutation visible inside the closure
	sayIt()                           // prints updated message — closure sees it
}

func sayIt_description() string { return "  closure sees:" }

// ─── 2. VARIABLE CAPTURE IS BY REFERENCE ─────────────────────────────────────
//
// This is the most important thing to understand about closures.
// Go closures capture the VARIABLE (the memory location), not the VALUE at the
// time of capture. This is unlike some other languages (e.g., Swift's [capture list]).
//
// Compare:
//   Python: closures capture by reference too (same gotcha)
//   JavaScript: 'let' closures capture by reference (var had the same bug)
//   Rust: closures can capture by value or by reference explicitly
//   Go: always by reference (but you can force by-value via a local copy trick)

func captureByReference() {
	x := 10
	inc := func() { x++ }   // captures x by reference
	get := func() int { return x } // also captures the SAME x

	fmt.Printf("  x = %d\n", get())
	inc()
	inc()
	fmt.Printf("  x after 2 increments = %d\n", get())
	fmt.Printf("  outer x = %d (same variable!)\n", x)
	// x is 12 in the outer scope too — because inc() and get() share the
	// exact same 'x'. There's only ONE x, on the heap.
}

// ─── 3. CLOSURE AS FUNCTION FACTORY ──────────────────────────────────────────
//
// A function factory creates and returns a new closure configured by the arguments
// passed to the factory. Each call to the factory produces an INDEPENDENT closure
// with its own copy of the captured variables.
//
// This is powerful because you defer configuration to call time, not definition time.

// makeAdder returns a closure that adds 'n' to any value passed to it.
// Each closure has its OWN 'n' — they don't share it.
func makeAdder(n int) func(int) int {
	return func(x int) int {
		return x + n
	}
}

// makeGreeter returns a greeting function specialized for a given prefix.
func makeGreeter(prefix string) func(string) string {
	return func(name string) string {
		return prefix + ", " + name + "!"
	}
}

// makeRateLimiter is a realistic factory: returns a function that tracks how
// many times it's been called and enforces a limit.
func makeRateLimiter(limit int) func() bool {
	count := 0 // captured by the returned closure; lives on heap
	return func() bool {
		if count >= limit {
			return false // limit exceeded
		}
		count++
		return true
	}
}

// ─── 4. STATEFUL FUNCTIONS (CLOSURES WITH MUTABLE CAPTURED STATE) ─────────────
//
// Because closures capture variables by reference, they can maintain state
// across calls — like a struct with methods, but lighter weight.
//
// When to use a closure vs a struct:
//   - Closure: when the state is simple and you don't need to expose it
//   - Struct:  when you need multiple methods, or you want to inspect state

func makeCounter(start int) func() int {
	n := start
	return func() int {
		current := n
		n++
		return current
	}
}

// makeAccumulator maintains a running total.
func makeAccumulator() func(int) int {
	total := 0
	return func(n int) int {
		total += n
		return total
	}
}

// makeStack returns push/pop/peek functions, all sharing the same slice.
// This demonstrates multiple closures sharing ONE piece of captured state.
func makeStack() (push func(int), pop func() (int, bool), peek func() (int, bool)) {
	var data []int // shared by all three closures

	push = func(v int) {
		data = append(data, v)
	}
	pop = func() (int, bool) {
		if len(data) == 0 {
			return 0, false
		}
		v := data[len(data)-1]
		data = data[:len(data)-1]
		return v, true
	}
	peek = func() (int, bool) {
		if len(data) == 0 {
			return 0, false
		}
		return data[len(data)-1], true
	}
	return
}

// ─── 5. THE LOOP VARIABLE CAPTURE GOTCHA ─────────────────────────────────────
//
// This is one of the most common Go bugs for newcomers (and tripped up even
// experienced developers before Go 1.22).
//
// THE BUG (pre-Go 1.22 behavior):
//   When you create closures inside a for loop, all closures capture the SAME
//   loop variable — not a snapshot of its value. By the time the closures run,
//   the loop has finished and the variable holds its FINAL value.
//
// GO 1.22 CHANGE:
//   As of Go 1.22, range loop variables are re-declared per iteration, meaning
//   each closure gets its own variable. The bug described below no longer occurs
//   with range-for loops in Go 1.22+. However, classic for-loops (for i := 0; ...)
//   still have the old behavior, and understanding this is important for reading
//   pre-1.22 code.
//
// THE FIX (works in all versions):
//   Create a local copy of the loop variable inside the loop body.

func loopCaptureBug() {
	// BUGGY pattern (classic for loop — still exhibits the bug in Go 1.22+):
	buggy := make([]func(), 5)
	for i := 0; i < 5; i++ {
		// All closures capture the SAME 'i'. By the time they run, i == 5.
		buggy[i] = func() {
			fmt.Printf("    buggy: i = %d\n", i)
		}
	}
	fmt.Println("  Buggy closures (classic for loop, all capture final i):")
	for _, f := range buggy {
		f() // all print 5!
	}

	// FIX 1: Local copy inside the loop body
	fixed1 := make([]func(), 5)
	for i := 0; i < 5; i++ {
		i := i // shadow 'i' with a new variable scoped to this iteration
		// Now each closure captures its OWN 'i', not the shared loop variable.
		fixed1[i] = func() {
			fmt.Printf("    fixed1: i = %d\n", i)
		}
	}
	fmt.Println("  Fixed closures (local copy trick):")
	for _, f := range fixed1 {
		f()
	}

	// FIX 2: Pass i as a parameter to an immediately-invoked closure.
	// Parameters are passed BY VALUE, so each gets its own copy.
	fixed2 := make([]func(), 5)
	for i := 0; i < 5; i++ {
		func(val int) {
			fixed2[val] = func() {
				fmt.Printf("    fixed2: i = %d\n", val)
			}
		}(i) // immediately invoke, passing current i by value
	}
	fmt.Println("  Fixed closures (immediately-invoked with param):")
	for _, f := range fixed2 {
		f()
	}

	// NOTE: range-for loops in Go 1.22+ are NOT buggy:
	fixed3 := make([]func(), 5)
	for i := range 5 { // Go 1.22 range-over-integer
		// In Go 1.22+, 'i' is re-declared per iteration — no bug here.
		fixed3[i] = func() {
			fmt.Printf("    fixed3 (Go 1.22 range): i = %d\n", i)
		}
	}
	fmt.Println("  Go 1.22 range-for (no bug):")
	for _, f := range fixed3 {
		f()
	}
}

// ─── 6. MEMOIZATION WITH CLOSURES ────────────────────────────────────────────
//
// Memoization = caching the results of expensive function calls based on inputs.
// A closure is perfect for this: the cache map lives inside the closure and
// persists across calls without being visible to the outside world.
//
// This is a simple single-threaded version. For concurrent use, you'd need
// a sync.Mutex or sync.Map (shown below).

// memoize wraps any func(int)int with a cache.
// The returned function checks the cache first; computes and stores on miss.
func memoize(fn func(int) int) func(int) int {
	cache := make(map[int]int) // captured by the returned closure
	return func(n int) int {
		if v, ok := cache[n]; ok {
			return v // cache hit
		}
		result := fn(n)   // cache miss: compute
		cache[n] = result // store in cache
		return result
	}
}

// Fibonacci is a canonical example: naive recursion is O(2^n).
// With memoization it becomes O(n).
// Note: this is a non-recursive memoized version for simplicity.
func makeMemoFib() func(int) int {
	cache := map[int]int{0: 0, 1: 1} // pre-seed base cases
	var fib func(int) int
	fib = func(n int) int {
		if v, ok := cache[n]; ok {
			return v
		}
		result := fib(n-1) + fib(n-2)
		cache[n] = result
		return result
	}
	return fib
}

// Thread-safe memoize using sync.Mutex — important in concurrent programs.
func memoizeSafe(fn func(int) int) func(int) int {
	var mu sync.Mutex
	cache := make(map[int]int)
	return func(n int) int {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := cache[n]; ok {
			return v
		}
		result := fn(n)
		cache[n] = result
		return result
	}
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  CLOSURES IN GO")
	fmt.Println(sep)

	// 1. Simple closure
	fmt.Println("\n── 1. Closure Captures by Reference ──")
	simpleClosure()

	// 2. Mutation visible through closure
	fmt.Println("\n── 2. Shared Variable (by Reference) ──")
	captureByReference()

	// 3. Function factory
	fmt.Println("\n── 3. Closure as Function Factory ──")
	add5 := makeAdder(5)
	add10 := makeAdder(10)
	fmt.Printf("  add5(3)  = %d\n", add5(3))
	fmt.Printf("  add10(3) = %d\n", add10(3))
	// add5 and add10 are INDEPENDENT — changing one doesn't affect the other.

	hello := makeGreeter("Hello")
	hi := makeGreeter("Hi")
	fmt.Println(" ", hello("Alice"))
	fmt.Println(" ", hi("Bob"))

	limiter := makeRateLimiter(3)
	for i := range 5 {
		fmt.Printf("  call %d: allowed=%v\n", i+1, limiter())
	}

	// 4. Stateful closures
	fmt.Println("\n── 4. Stateful Closures ──")
	counter := makeCounter(1)
	fmt.Printf("  counter: %d %d %d\n", counter(), counter(), counter())
	// Note: function call order in a single expression is unspecified in Go.
	// The above prints 1 2 3 only because Go evaluates left-to-right in practice,
	// but for correctness, call sequentially:

	acc := makeAccumulator()
	for _, v := range []int{10, 20, 30, 40} {
		fmt.Printf("  acc(%d) = %d\n", v, acc(v))
	}

	push, pop, peek := makeStack()
	push(1); push(2); push(3)
	if top, ok := peek(); ok {
		fmt.Printf("  peek = %d\n", top)
	}
	if v, ok := pop(); ok {
		fmt.Printf("  pop  = %d\n", v)
	}
	if top, ok := peek(); ok {
		fmt.Printf("  peek after pop = %d\n", top)
	}

	// 5. Loop capture gotcha
	fmt.Println("\n── 5. Loop Variable Capture Gotcha ──")
	loopCaptureBug()

	// 6. Memoization
	fmt.Println("\n── 6. Memoization with Closures ──")
	callCount := 0
	expensiveFn := func(n int) int {
		callCount++
		// Simulate expensive computation
		return n * n
	}
	memoExpensive := memoize(expensiveFn)

	for _, n := range []int{5, 10, 5, 10, 15} {
		fmt.Printf("  memoExpensive(%2d) = %3d  (total compute calls: %d)\n",
			n, memoExpensive(n), callCount)
	}
	// Notice: 5 and 10 are only computed once even though queried twice.

	fib := makeMemoFib()
	fmt.Print("  Fibonacci sequence: ")
	for i := range 10 {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(fib(i))
	}
	fmt.Println()

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • Closures capture variables by REFERENCE (not by value)")
	fmt.Println("  • All closures from one scope share the same captured variables")
	fmt.Println("  • Loop bug: classic for-loop var is shared; use local copy to fix")
	fmt.Println("  • Go 1.22+: range-for vars are per-iteration (no bug)")
	fmt.Println("  • Closures enable stateful functions without defining structs")
	fmt.Println("  • Memoization: cache lives in closure — private and persistent")
	fmt.Println(sep)
}
