package main

// =============================================================================
// MODULE 02: FUNCTIONS — Every feature, every pattern
// =============================================================================
// Run: go run 02_functions/main.go
// =============================================================================

import (
	"fmt"
	"math"
	"strings"
)

// =============================================================================
// BASICS: Function declaration anatomy
// =============================================================================
// func  functionName  (parameter list)  (return types) { body }
//  ^         ^               ^                ^
// keyword  name      name type pairs    optional

// Simple function — no return value
func greet(name string) {
	fmt.Println("Hello,", name)
}

// Function with return value
func add(x int, y int) int {
	return x + y
}

// Shorthand: same type parameters — list type once at the end
func multiply(x, y int) int {
	return x * y
}

// =============================================================================
// MULTIPLE RETURN VALUES — Go's most unique feature
// =============================================================================
// Go functions can return ANY number of values.
// This replaces the need for output parameters (like in C).
// Convention: last return value is often an error.

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("cannot divide by zero")
	}
	return a / b, nil
}

// Returning multiple values — real world example
func minMax(arr []int) (min, max int) {
	min, max = arr[0], arr[0]
	for _, v := range arr[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// =============================================================================
// NAMED RETURN VALUES
// =============================================================================
// You can name return values — they become variables initialized to zero.
// A bare "return" returns whatever the named variables hold.
// Use sparingly — only when it adds clarity.

func circle(radius float64) (area, circumference float64) {
	area = math.Pi * radius * radius
	circumference = 2 * math.Pi * radius
	return // bare return — returns area and circumference
}

// Better with explicit return (most Go pros prefer this for clarity)
func circleExplicit(radius float64) (area float64, circumference float64) {
	area = math.Pi * radius * radius
	circumference = 2 * math.Pi * radius
	return area, circumference
}

// =============================================================================
// VARIADIC FUNCTIONS — Accept any number of arguments
// =============================================================================
// The ...type syntax makes a parameter variadic.
// Variadic parameter becomes a SLICE inside the function.
// ONLY the LAST parameter can be variadic.

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func printAll(sep string, values ...string) {
	fmt.Println(strings.Join(values, sep))
}

// Spread operator: pass a slice into a variadic function with ...
// someFunc(slice...)

// =============================================================================
// FUNCTIONS AS FIRST-CLASS VALUES
// =============================================================================
// In Go, functions are FIRST-CLASS citizens:
// - Assign to variables
// - Pass as arguments
// - Return from functions
// - Store in data structures

// Function type: func(params) returns
type MathOperation func(int, int) int // custom type for a function signature

func applyOperation(a, b int, op MathOperation) int {
	return op(a, b)
}

// Higher-order function: takes and returns functions
func makeMultiplier(factor int) func(int) int {
	return func(n int) int {
		return n * factor
	}
}

// =============================================================================
// ANONYMOUS FUNCTIONS (Function Literals)
// =============================================================================
// Functions without a name — defined inline.
// Can be immediately invoked (IIFE) or stored in a variable.

// =============================================================================
// CLOSURES — Functions that capture their environment
// =============================================================================
// A closure is an anonymous function that CAPTURES variables from the
// enclosing scope. The captured variables live as long as the closure lives.
// Key insight: closures share the SAME variable, not a copy.

func makeCounter() func() int {
	count := 0 // this variable is CAPTURED by the returned closure
	return func() int {
		count++ // each call modifies the SAME count
		return count
	}
}

// Multiple closures sharing same variable
func makeCounterPair() (func(), func() int) {
	count := 0
	increment := func() {
		count++
	}
	get := func() int {
		return count
	}
	return increment, get
}

// Closure with parameter — makes a configurable adder
func makeAdder(x int) func(int) int {
	return func(y int) int {
		return x + y // x is captured from makeAdder's scope
	}
}

// =============================================================================
// DEFER — Deep dive
// =============================================================================
// defer delays execution until the surrounding function returns.
// Defers run in LIFO (stack) order.
// Arguments to deferred functions are evaluated IMMEDIATELY at defer time.
// Common uses: cleanup, closing files, unlocking mutexes, timing

func withDefer() {
	fmt.Println("start")
	defer fmt.Println("end (deferred)")
	fmt.Println("middle")
	// Output: start → middle → end
}

// Defer for cleanup pattern — very common in Go
func readFile(filename string) {
	fmt.Println("Opening", filename)
	// In real code: f, _ := os.Open(filename)
	defer fmt.Println("Closing", filename) // guaranteed to run even if panic
	fmt.Println("Reading", filename)
}

// Defer with loop — each defer captures the VALUE of i at that point
func deferInLoop() {
	for i := 0; i < 3; i++ {
		// Arguments are evaluated NOW
		defer fmt.Println("deferred i:", i) // prints 2, 1, 0
	}
}

// Defer modifying named return values — a subtle but important behavior
func withDeferReturn() (result int) {
	defer func() {
		result++ // can modify named return value!
	}()
	return 10 // sets result=10, then defer runs: result=11
}

// =============================================================================
// PANIC AND RECOVER
// =============================================================================
// panic: stops normal execution, unwinds the stack, runs deferred functions
// recover: only useful inside a deferred function — stops the panic

func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
		}
	}()
	result = a / b // panics if b == 0
	return
}

// =============================================================================
// RECURSION
// =============================================================================

func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

// Tail-recursive style (Go doesn't optimize tail calls, but it's cleaner)
func factTail(n, acc int) int {
	if n <= 1 {
		return acc
	}
	return factTail(n-1, n*acc)
}

// =============================================================================
// INIT FUNCTION — Special function
// =============================================================================
// init() runs automatically before main().
// Each file can have multiple init() functions.
// They run in the order the files are initialized.
// Cannot be called manually.

func init() {
	fmt.Println("[init] Module 02 initializing...")
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 02: FUNCTIONS ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Basic functions
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Basic Functions ---")

	greet("Achyut")
	fmt.Println("add:", add(3, 4))
	fmt.Println("multiply:", multiply(6, 7))

	// -------------------------------------------------------------------------
	// SECTION 2: Multiple return values
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Multiple Return Values ---")

	result, err := divide(10, 3)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("10 / 3 = %.4f\n", result)
	}

	_, err2 := divide(5, 0) // use _ to discard result
	if err2 != nil {
		fmt.Println("Error:", err2)
	}

	nums := []int{5, 2, 8, 1, 9, 3}
	min, max := minMax(nums)
	fmt.Println("Min:", min, "Max:", max)

	// -------------------------------------------------------------------------
	// SECTION 3: Named return values
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Named Returns ---")

	area, circ := circle(5)
	fmt.Printf("Circle r=5: area=%.2f, circumference=%.2f\n", area, circ)

	// -------------------------------------------------------------------------
	// SECTION 4: Variadic functions
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Variadic Functions ---")

	fmt.Println("sum():", sum())            // 0 — works with zero args
	fmt.Println("sum(1,2):", sum(1, 2))     // 3
	fmt.Println("sum(1..5):", sum(1, 2, 3, 4, 5)) // 15

	// Spread a slice into variadic function
	numbers := []int{10, 20, 30, 40}
	fmt.Println("sum(slice...):", sum(numbers...)) // spread operator

	printAll(", ", "apple", "banana", "cherry")
	printAll(" | ", "Go", "Python", "Rust")

	// -------------------------------------------------------------------------
	// SECTION 5: Functions as values
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Functions as Values ---")

	// Assign function to variable
	addFn := add
	fmt.Println("addFn(5, 6):", addFn(5, 6))

	// Pass function as argument
	result2 := applyOperation(10, 5, multiply)
	fmt.Println("applyOperation(10, 5, multiply):", result2)

	// Anonymous function passed directly
	result3 := applyOperation(10, 5, func(a, b int) int { return a - b })
	fmt.Println("applyOperation subtract:", result3)

	// Function stored in map
	ops := map[string]func(int, int) int{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
	}
	for opName, opFn := range ops {
		fmt.Printf("ops[%s](8, 3) = %d\n", opName, opFn(8, 3))
	}

	// -------------------------------------------------------------------------
	// SECTION 6: Anonymous functions (IIFE)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Anonymous Functions (IIFE) ---")

	// Immediately Invoked Function Expression
	result4 := func(x, y int) int {
		return x * y
	}(6, 7) // called immediately
	fmt.Println("IIFE result:", result4)

	// Useful for creating a scope or for goroutines
	func() {
		secret := "I only exist in this block"
		fmt.Println(secret)
	}()

	// -------------------------------------------------------------------------
	// SECTION 7: Closures
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Closures ---")

	// Each call to makeCounter() creates an independent counter
	counter1 := makeCounter()
	counter2 := makeCounter()

	fmt.Println("counter1:", counter1()) // 1
	fmt.Println("counter1:", counter1()) // 2
	fmt.Println("counter1:", counter1()) // 3
	fmt.Println("counter2:", counter2()) // 1 — independent!
	fmt.Println("counter2:", counter2()) // 2

	// Shared variable between two closures
	inc, get := makeCounterPair()
	inc()
	inc()
	inc()
	fmt.Println("shared counter:", get()) // 3

	// Configurable adder
	add5 := makeAdder(5)
	add10 := makeAdder(10)
	fmt.Println("add5(3):", add5(3))   // 8
	fmt.Println("add10(3):", add10(3)) // 13

	// CLOSURE GOTCHA: Loop variable capture
	fmt.Println("\n--- Closure Gotcha ---")

	// WRONG: all closures capture the SAME loop variable
	funcs := make([]func(), 5)
	for i := 0; i < 5; i++ {
		i := i // SHADOW i — creates a NEW i for each iteration!
		funcs[i] = func() {
			fmt.Print(i, " ")
		}
	}
	for _, f := range funcs {
		f()
	}
	fmt.Println()

	// NOTE: In Go 1.22+, loop variables have per-iteration scope by default
	// so this is no longer a problem in modern Go. But knowing it exists matters.

	// Multiplier factory
	double := makeMultiplier(2)
	triple := makeMultiplier(3)
	fmt.Println("double(7):", double(7)) // 14
	fmt.Println("triple(7):", triple(7)) // 21

	// -------------------------------------------------------------------------
	// SECTION 8: Defer — detailed behavior
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Defer ---")

	withDefer()
	readFile("data.txt")

	// Defer argument evaluation time
	x := 10
	defer fmt.Println("deferred x:", x) // captures x=10 NOW
	x = 20
	fmt.Println("current x:", x) // 20
	// deferred will print: deferred x: 10 (original value!)

	// Named return + defer
	r := withDeferReturn()
	fmt.Println("withDeferReturn:", r) // 11, not 10!

	// -------------------------------------------------------------------------
	// SECTION 9: Panic and Recover
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Panic and Recover ---")

	res, err3 := safeDiv(10, 2)
	fmt.Println("safeDiv(10, 2):", res, err3) // 5, nil

	res2, err4 := safeDiv(10, 0)
	fmt.Println("safeDiv(10, 0):", res2, err4) // 0, error

	// -------------------------------------------------------------------------
	// SECTION 10: Recursion
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Recursion ---")

	for i := 0; i <= 10; i++ {
		fmt.Printf("factorial(%d) = %d\n", i, factorial(i))
	}

	fmt.Println("fibonacci sequence:")
	for i := 0; i < 10; i++ {
		fmt.Print(fibonacci(i), " ")
	}
	fmt.Println()

	fmt.Println("factTail(10):", factTail(10, 1))

	// -------------------------------------------------------------------------
	// SECTION 11: Function types and interfaces
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Function Types ---")

	// Using the custom type MathOperation
	var op MathOperation = add
	fmt.Println("op(3,4):", op(3, 4))

	op = multiply
	fmt.Println("op(3,4):", op(3, 4))

	// Function slice
	pipeline := []func(int) int{
		func(n int) int { return n + 1 },
		func(n int) int { return n * 2 },
		func(n int) int { return n - 3 },
	}

	val := 5
	for _, fn := range pipeline {
		val = fn(val)
	}
	fmt.Println("pipeline result:", val) // ((5+1)*2)-3 = 9

	// -------------------------------------------------------------------------
	// SECTION 12: Method values and expressions
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Method Values ---")

	// A method bound to a specific receiver becomes a function value
	s := strings.Builder{}
	writeFn := s.WriteString // method value — bound to s
	writeFn("Hello")
	writeFn(", World!")
	fmt.Println(s.String())

	fmt.Println("\n=== MODULE 02 COMPLETE ===")
}
