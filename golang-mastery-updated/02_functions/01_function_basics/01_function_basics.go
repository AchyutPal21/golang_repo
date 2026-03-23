// 01_function_basics.go
//
// TOPIC: Function Declaration, Multiple Returns, Named Returns,
//        Variadic Functions, and First-Class Functions
//
// Go's function system is deceptively simple on the surface but has several
// features that distinguish it from languages like Java, Python, or C++.
// The key ideas:
//   1. Multiple return values — no need for out-parameters or Result<T,E> wrappers
//   2. Named return values    — documentation + naked returns (use sparingly)
//   3. Variadic functions     — clean API for variable argument lists
//   4. First-class functions  — functions are values, enabling functional patterns
//
// WHY does Go have these?
//   Go was designed for clarity and explicitness. Multiple returns eliminate the
//   awkward C pattern of returning error codes via pointer parameters. First-class
//   functions let you write clean callbacks, middleware, and strategy patterns
//   without Java-style single-method interface boilerplate.

package main

import (
	"fmt"
	"math"
	"strings"
)

// ─── 1. BASIC FUNCTION DECLARATION ────────────────────────────────────────────
//
// Syntax: func <name>(<params>) <returnType> { ... }
//
// Go is explicit: you MUST declare parameter types. Unlike Python there is no
// duck typing; unlike C++ there are no default parameter values.
// If multiple consecutive params share a type, you can group them:
//   func add(x, y int) int   — same as func add(x int, y int) int

func add(x, y int) int {
	// The return type appears AFTER the parameter list — this is the opposite of
	// Java/C. Rob Pike has said it reads more naturally: "function add, takes
	// two ints, returns int."
	return x + y
}

// Functions can return any type, including structs, slices, maps, interfaces.
func greet(name string) string {
	return "Hello, " + name + "!"
}

// A function with no return value. Go uses no keyword for void — just omit it.
// (C has void, Java has void, Go just... doesn't have it.)
func printDivider(char string, length int) {
	fmt.Println(strings.Repeat(char, length))
}

// ─── 2. MULTIPLE RETURN VALUES ────────────────────────────────────────────────
//
// This is one of Go's most impactful features. In most languages, returning two
// values requires either:
//   - A struct/tuple wrapper (Python, Rust, Swift)
//   - An out-parameter passed by pointer (C, C++)
//   - Throwing an exception (Java, C#)
//
// Go chose explicit multiple returns. The dominant idiom is (result, error):
//   value, err := someOperation()
//   if err != nil { ... }
//
// This makes error handling visible at every call site — no hidden exceptions.

func divide(a, b float64) (float64, error) {
	if b == 0 {
		// We return the zero value for float64 (0.0) as the result when there's
		// an error. This is idiomatic — callers check err first, and the zero
		// value signals "no meaningful result".
		return 0, fmt.Errorf("divide: cannot divide %.2f by zero", a)
	}
	return a / b, nil
}

// Multiple returns aren't only for (value, error) — you can return any combo.
// Here we return (min, max) from a single pass through data.
func minMax(nums []int) (int, int) {
	if len(nums) == 0 {
		return 0, 0
	}
	min, max := nums[0], nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return min, max
}

// ─── 3. NAMED RETURN VALUES ───────────────────────────────────────────────────
//
// You can name the return values in the signature. This does two things:
//   a) Documents what each return value means (like self-documenting code)
//   b) Pre-declares those variables, initialized to their zero values
//
// When to use named returns:
//   - When the function body is complex and naming clarifies meaning
//   - With defer + named return for cleanup (covered in 03_defer_advanced.go)
//   - Short functions where naked returns improve readability
//
// When NOT to use named returns:
//   - Large functions where "naked return" is confusing (Go vet warns about this)
//   - When the names add no clarity over the types alone
//
// The Go standard library uses named returns selectively — not everywhere.

func circleStats(radius float64) (area, circumference float64) {
	// 'area' and 'circumference' are pre-declared and zero-initialized.
	// We can assign to them directly without := inside the function body.
	area = math.Pi * radius * radius
	circumference = 2 * math.Pi * radius
	// A "naked return" returns the current values of named return variables.
	// This is fine for SHORT functions. For long functions, naked returns
	// make code hard to read — you can't tell what's being returned without
	// scrolling back to the signature.
	return
}

// Named returns + defer pattern (preview — full detail in 03_defer_advanced.go).
// This demonstrates WHY named returns exist: the deferred function can read and
// modify the named return values, allowing cleanup that also changes the result.
func readConfig(path string) (result string, err error) {
	// Imagine opening a file here...
	defer func() {
		// Because 'result' and 'err' are named, this defer can inspect them.
		// This is impossible with anonymous return values.
		if err != nil {
			result = "<default config>"
			// We intentionally don't clear err — caller still knows it failed
		}
	}()
	if path == "" {
		err = fmt.Errorf("readConfig: path cannot be empty")
		return // naked return: returns result="" and err=<the error>
	}
	result = "config from " + path
	return
}

// ─── 4. VARIADIC FUNCTIONS ────────────────────────────────────────────────────
//
// Variadic functions accept a variable number of arguments of one type.
// Syntax: func name(fixed params, rest ...T)
//
// Inside the function, 'rest' is a []T (a slice). This is NOT magic — it's
// just syntactic sugar. The compiler packs the args into a slice for you.
//
// Rules:
//   - Only the LAST parameter can be variadic
//   - You can pass zero arguments to a variadic parameter (gives empty slice)
//   - fmt.Println, fmt.Printf are variadic: func Println(a ...any) (int, error)

func sum(nums ...int) int {
	// nums is []int inside this function — iterate normally.
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Mixed: fixed param + variadic. The fixed param MUST come first.
func joinStrings(sep string, parts ...string) string {
	return strings.Join(parts, sep)
}

// ─── 4a. THE SPREAD OPERATOR ──────────────────────────────────────────────────
//
// When you HAVE a slice and want to pass it to a variadic function, use `...`
// after the slice to "spread" it. This is the opposite of packing.
//
// WITHOUT spread: sum(mySlice)     — COMPILE ERROR (passing []int where int expected)
// WITH spread:    sum(mySlice...)  — correct, unpacks the slice as individual args
//
// The spread operator does NOT copy the slice — the function receives the same
// underlying array. This matters for performance in tight loops.

func spreadDemo() {
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println("sum via spread:", sum(nums...)) // equivalent to sum(1,2,3,4,5)

	// You can mix a leading fixed slice with a spread — but only if the variadic
	// param is alone. You cannot do: sum(0, nums...) — that would be a syntax error
	// because 0 is not a slice. Instead:
	withExtra := append([]int{0}, nums...)
	fmt.Println("sum with extra:", sum(withExtra...))
}

// ─── 5. FUNCTIONS AS FIRST-CLASS VALUES ──────────────────────────────────────
//
// In Go, functions are values just like int, string, or struct.
// You can:
//   - Assign a function to a variable
//   - Pass a function as an argument
//   - Return a function from a function
//   - Store functions in slices or maps
//
// This is the foundation for:
//   - Callbacks (pass behavior, not data)
//   - Higher-order functions (filter, map, reduce)
//   - Closures (functions that capture their environment)
//   - Middleware chains (e.g., HTTP handlers)
//
// In Java you'd need a functional interface (Runnable, Function<T,R>).
// In Python every function is already first-class. Go is similar to Python here.

// A function that takes another function as an argument.
// 'transform' has type func(int) int — this is the function type.
func applyToEach(nums []int, transform func(int) int) []int {
	result := make([]int, len(nums))
	for i, n := range nums {
		result[i] = transform(n)
	}
	return result
}

// A function that RETURNS a function.
// The returned function is a "multiplier factory" — we'll explore this more
// deeply in 02_closures.go, but the mechanics start here.
func makeMultiplier(factor int) func(int) int {
	// We return an anonymous function (a function literal).
	// It captures 'factor' from the enclosing scope — this is a closure.
	return func(n int) int {
		return n * factor
	}
}

// Assigning a named function to a variable.
// The variable 'fn' has type func(int, int) int — same signature as 'add'.
// In Go, functions have structural types — any function with the same
// signature is assignable, regardless of its name.
func assignFunctionToVar() {
	fn := add // no parentheses — we want the function itself, not its result
	fmt.Printf("Type of fn: %T\n", fn)
	fmt.Println("fn(3, 4) =", fn(3, 4))
}

// Storing functions in a map — useful for command dispatch tables.
func functionMap() {
	ops := map[string]func(int, int) int{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
	}

	for name, op := range ops {
		fmt.Printf("%-3s(10, 3) = %d\n", name, op(10, 3))
	}
}

// ─── MAIN: DEMONSTRATE EVERYTHING ────────────────────────────────────────────

func main() {
	printDivider("═", 55)
	fmt.Println("  FUNCTION BASICS IN GO")
	printDivider("═", 55)

	// 1. Basic functions
	fmt.Println("\n── 1. Basic Functions ──")
	fmt.Println("add(3, 4)      =", add(3, 4))
	fmt.Println("greet(\"Alice\") =", greet("Alice"))

	// 2. Multiple return values
	fmt.Println("\n── 2. Multiple Return Values ──")
	result, err := divide(10, 3)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("10 / 3 = %.4f\n", result)
	}

	_, err = divide(5, 0)
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Ignoring a return value with blank identifier _
	// In Go you MUST use every variable you declare. _ lets you discard one.
	min, max := minMax([]int{3, 1, 4, 1, 5, 9, 2, 6})
	fmt.Printf("min=%d, max=%d\n", min, max)

	// 3. Named return values
	fmt.Println("\n── 3. Named Return Values ──")
	area, circ := circleStats(5)
	fmt.Printf("Circle r=5: area=%.2f, circumference=%.2f\n", area, circ)

	cfg, err := readConfig("")
	fmt.Printf("readConfig(\"\") => result=%q, err=%v\n", cfg, err)
	cfg, err = readConfig("/etc/app.conf")
	fmt.Printf("readConfig(path) => result=%q, err=%v\n", cfg, err)

	// 4. Variadic functions
	fmt.Println("\n── 4. Variadic Functions ──")
	fmt.Println("sum()           =", sum())         // zero args: valid, returns 0
	fmt.Println("sum(1,2,3)      =", sum(1, 2, 3))
	fmt.Println("sum(1..10)      =", sum(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))
	fmt.Println("joinStrings     =", joinStrings("-", "a", "b", "c", "d"))

	// 4a. Spread operator
	fmt.Println("\n── 4a. Spread Operator ──")
	spreadDemo()

	// 5. Functions as first-class values
	fmt.Println("\n── 5. First-Class Functions ──")

	// Assign to variable
	assignFunctionToVar()

	// Pass as argument
	nums := []int{1, 2, 3, 4, 5}
	doubled := applyToEach(nums, func(n int) int { return n * 2 })
	fmt.Println("doubled:", doubled)

	squared := applyToEach(nums, func(n int) int { return n * n })
	fmt.Println("squared:", squared)

	// Return from function (factory)
	triple := makeMultiplier(3)
	fmt.Printf("triple(7) = %d\n", triple(7))

	tenX := makeMultiplier(10)
	fmt.Printf("tenX(7)   = %d\n", tenX(7))

	// Function map (dispatch table)
	fmt.Println("\n── Function Map (Dispatch Table) ──")
	functionMap()

	printDivider("═", 55)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • Multiple returns eliminate need for out-params/exceptions")
	fmt.Println("  • Named returns = documentation + naked return (use sparingly)")
	fmt.Println("  • Variadic ...T packs args into a slice; spread T... unpacks")
	fmt.Println("  • Functions are values: assign, pass, return, store in maps")
	printDivider("═", 55)
}
