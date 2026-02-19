package main

// =============================================================================
// MODULE 01: GO FUNDAMENTALS — Every tiny detail, from scratch
// =============================================================================
// Run this file:  go run 01_fundamentals/main.go
// Uncomment one section at a time, read it, run it, understand it.
// =============================================================================

import (
	"fmt"
	"math"
	"math/cmplx"
	"unsafe"
)

func main() {

	// =========================================================================
	// SECTION 1: HOW GO RUNS — The entry point
	// =========================================================================
	// Every Go program starts from func main() inside package main.
	// Go compiles to a single binary. No VM, no interpreter.
	// Execution order: package-level var init → init() functions → main()
	fmt.Println("=== MODULE 01: FUNDAMENTALS ===")

	// =========================================================================
	// SECTION 2: VARIABLES — All 4 ways to declare
	// =========================================================================

	// WAY 1: var with explicit type
	var age int = 25
	fmt.Println("age:", age)

	// WAY 2: var with type inference (Go infers type from value)
	var name = "Achyut"
	fmt.Println("name:", name)

	// WAY 3: var zero value (no value given — gets zero value of the type)
	// Zero values: int=0, float=0.0, bool=false, string="", pointer=nil
	var score int    // 0
	var gpa float64  // 0.0
	var passed bool  // false
	var label string // ""
	fmt.Println("zero values:", score, gpa, passed, label)

	// WAY 4: Short declaration := (ONLY inside functions)
	city := "Kolkata"   // type inferred as string
	count := 10         // type inferred as int
	pi := 3.14          // type inferred as float64
	active := true      // type inferred as bool
	fmt.Println(city, count, pi, active)

	// MULTIPLE variables at once
	x, y, z := 1, 2, 3
	fmt.Println("x y z:", x, y, z)

	// BLOCK declaration (clean way for multiple vars)
	var (
		firstName string = "Achyut"
		lastName  string = "Pal"
		rollNo    int    = 42
	)
	fmt.Println(firstName, lastName, rollNo)

	// KEY RULES:
	// - := only inside functions
	// - var can be at package level or inside functions
	// - Every declared variable MUST be used (Go enforces this — no dead variables)
	// - _ (blank identifier) is used to discard a value

	_ = "this value is intentionally discarded"

	// =========================================================================
	// SECTION 3: TYPES — Every built-in type in Go
	// =========================================================================

	// --- INTEGERS ---
	// Signed (can be negative or positive):
	var i8 int8 = 127          // -128 to 127         (8 bits)
	var i16 int16 = 32767      // -32768 to 32767      (16 bits)
	var i32 int32 = 2147483647 // -2^31 to 2^31-1      (32 bits)
	var i64 int64 = 9223372036854775807 // -2^63 to 2^63-1 (64 bits)
	var i int = 100            // platform: 32 or 64 bit (use this by default)

	fmt.Println("Signed ints:", i8, i16, i32, i64, i)

	// Unsigned (only positive):
	var u8 uint8 = 255       // 0 to 255
	var u16 uint16 = 65535   // 0 to 65535
	var u32 uint32 = 4294967295
	var u64 uint64 = 18446744073709551615
	var u uint = 100         // platform: 32 or 64 bit

	fmt.Println("Unsigned ints:", u8, u16, u32, u64, u)

	// byte = alias for uint8 (used for raw bytes / ASCII)
	var b byte = 'A' // 65
	fmt.Println("byte:", b, string(b)) // prints 65 and A

	// rune = alias for int32 (used for Unicode code points)
	var r rune = '🚀' // Unicode emoji
	fmt.Println("rune:", r, string(r)) // prints number and 🚀

	// uintptr: large enough to hold a pointer value (used in unsafe code)
	var ptr uintptr = 0
	_ = ptr

	// --- FLOATS ---
	var f32 float32 = 3.14159265358979 // 6-7 decimal precision
	var f64 float64 = 3.14159265358979 // 15-16 decimal precision (use this by default)
	fmt.Printf("float32: %.10f\n", f32) // notice precision loss
	fmt.Printf("float64: %.15f\n", f64) // full precision

	// --- COMPLEX ---
	var c64 complex64 = 3 + 4i
	var c128 complex128 = cmplx.Sqrt(-5 + 12i)
	fmt.Println("complex64:", c64)
	fmt.Println("complex128:", c128)
	fmt.Println("real part:", real(c64), "imaginary:", imag(c64))

	// --- BOOL ---
	var t bool = true
	var f bool = false
	fmt.Println("bool:", t, f)
	fmt.Println("AND:", t && f, "OR:", t || f, "NOT:", !t)

	// --- STRING ---
	var s string = "Hello, Go!"
	fmt.Println("string:", s)
	fmt.Println("length in bytes:", len(s)) // len gives bytes, not characters
	fmt.Println("first byte:", s[0])        // 72 = 'H' in ASCII
	fmt.Println("first char:", string(s[0])) // "H"

	// String is immutable — you cannot do s[0] = 'h' (compile error)
	// String is a sequence of bytes (UTF-8 encoded)

	// Raw string literals (backtick) — no escape processing
	raw := `This is line 1
This is line 2
Backslash: \n is NOT a newline here`
	fmt.Println(raw)

	// =========================================================================
	// SECTION 4: TYPE SIZES — How much memory each type uses
	// =========================================================================
	fmt.Println("\n--- Type Sizes ---")
	fmt.Println("int8:", unsafe.Sizeof(i8), "bytes")
	fmt.Println("int16:", unsafe.Sizeof(i16), "bytes")
	fmt.Println("int32:", unsafe.Sizeof(i32), "bytes")
	fmt.Println("int64:", unsafe.Sizeof(i64), "bytes")
	fmt.Println("float32:", unsafe.Sizeof(f32), "bytes")
	fmt.Println("float64:", unsafe.Sizeof(f64), "bytes")
	fmt.Println("bool:", unsafe.Sizeof(t), "bytes")

	// =========================================================================
	// SECTION 5: TYPE CONVERSION — Go is STRICTLY typed, NO implicit conversion
	// =========================================================================
	fmt.Println("\n--- Type Conversion ---")

	var myInt int = 42
	var myFloat float64 = float64(myInt) // EXPLICIT conversion required
	var myUint uint = uint(myFloat)

	fmt.Println(myInt, myFloat, myUint)

	// String conversion
	num := 65
	letter := string(rune(num))  // int → rune → string gives the character
	fmt.Println("int to char:", letter) // "A"

	// To convert number to its text: use fmt.Sprintf or strconv (covered later)
	numText := fmt.Sprintf("%d", num)
	fmt.Println("int to text:", numText, "type:", fmt.Sprintf("%T", numText))

	// =========================================================================
	// SECTION 6: CONSTANTS — Compile-time values
	// =========================================================================
	fmt.Println("\n--- Constants ---")

	const MaxRetries = 3         // untyped constant
	const AppName string = "GoMastery" // typed constant
	const Gravity = 9.8

	fmt.Println(MaxRetries, AppName, Gravity)

	// Constants CANNOT use :=
	// Constants CANNOT be changed after declaration
	// Untyped constants have arbitrary precision (very useful)

	// IOTA — auto-incrementing constant generator (used in enums)
	const (
		Sunday    = iota // 0
		Monday           // 1
		Tuesday          // 2
		Wednesday        // 3
		Thursday         // 4
		Friday           // 5
		Saturday         // 6
	)
	fmt.Println("Days:", Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday)

	// iota with expressions
	const (
		_  = iota             // 0 — skip
		KB = 1 << (10 * iota) // 1 << 10 = 1024
		MB                    // 1 << 20 = 1048576
		GB                    // 1 << 30 = 1073741824
		TB                    // 1 << 40
	)
	fmt.Printf("KB=%d MB=%d GB=%d TB=%d\n", KB, MB, GB, TB)

	// =========================================================================
	// SECTION 7: OPERATORS — Every operator in Go
	// =========================================================================
	fmt.Println("\n--- Operators ---")

	a, bb := 10, 3

	// Arithmetic
	fmt.Println("+ :", a+bb)  // 13
	fmt.Println("- :", a-bb)  // 7
	fmt.Println("* :", a*bb)  // 30
	fmt.Println("/ :", a/bb)  // 3  (integer division, truncates)
	fmt.Println("% :", a%bb)  // 1  (remainder/modulo)

	// Float division
	fa, fb := float64(a), float64(bb)
	fmt.Println("float /:", fa/fb) // 3.333...

	// Increment / Decrement (these are STATEMENTS in Go, not expressions)
	counter := 0
	counter++ // counter = counter + 1
	counter++ // counter = 2
	counter-- // counter = 1
	fmt.Println("counter:", counter)
	// NOTE: Go has NO ++counter or --counter (pre-increment doesn't exist)
	// NOTE: You cannot do: x = counter++ (it's a statement, not expression)

	// Comparison operators (return bool)
	fmt.Println("==:", a == bb) // false
	fmt.Println("!=:", a != bb) // true
	fmt.Println("> :", a > bb)  // true
	fmt.Println("< :", a < bb)  // false
	fmt.Println(">=:", a >= bb) // true
	fmt.Println("<=:", a <= bb) // false

	// Logical operators
	p, q := true, false
	fmt.Println("&& (AND):", p && q) // false
	fmt.Println("|| (OR):", p || q)  // true
	fmt.Println("!  (NOT):", !p)     // false

	// Short-circuit evaluation:
	// In p && q: if p is false, q is NOT evaluated
	// In p || q: if p is true, q is NOT evaluated

	// Bitwise operators (operate on individual bits)
	m, n := 12, 10 // 12=1100, 10=1010 in binary
	fmt.Printf("& (AND): %d & %d = %d\n", m, n, m&n)   // 1000 = 8
	fmt.Printf("| (OR):  %d | %d = %d\n", m, n, m|n)   // 1110 = 14
	fmt.Printf("^ (XOR): %d ^ %d = %d\n", m, n, m^n)   // 0110 = 6
	fmt.Printf("&^ (AND NOT): %d &^ %d = %d\n", m, n, m&^n) // 0100 = 4
	fmt.Printf("<< (left shift):  %d << 2 = %d\n", m, m<<2)  // 48
	fmt.Printf(">> (right shift): %d >> 1 = %d\n", m, m>>1)  // 6

	// Assignment operators
	val := 10
	val += 5  // val = val + 5  → 15
	val -= 3  // val = val - 3  → 12
	val *= 2  // val = val * 2  → 24
	val /= 4  // val = val / 4  → 6
	val %= 4  // val = val % 4  → 2
	fmt.Println("val after assignments:", val)

	// =========================================================================
	// SECTION 8: CONTROL FLOW — if, for, switch, defer
	// =========================================================================
	fmt.Println("\n--- Control Flow ---")

	// --- IF ---
	temperature := 35

	// Basic if
	if temperature > 30 {
		fmt.Println("It's hot!")
	}

	// if-else
	if temperature > 30 {
		fmt.Println("Hot")
	} else {
		fmt.Println("Cool")
	}

	// if-else if-else
	if temperature > 40 {
		fmt.Println("Extreme heat")
	} else if temperature > 30 {
		fmt.Println("Hot")
	} else if temperature > 20 {
		fmt.Println("Warm")
	} else {
		fmt.Println("Cool")
	}

	// if with init statement — variable scoped to the if block
	if temp := 37.5; temp > 37.0 {
		fmt.Println("Fever detected:", temp)
	}
	// temp is not accessible here — it died with the if block

	// --- FOR --- (Go's ONLY loop — no while, no do-while)

	// Classic C-style for
	for i := 0; i < 5; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// While-style (omit init and post)
	j := 0
	for j < 5 {
		fmt.Print(j, " ")
		j++
	}
	fmt.Println()

	// Infinite loop (omit everything)
	k := 0
	for {
		if k >= 5 {
			break // break exits the loop
		}
		fmt.Print(k, " ")
		k++
	}
	fmt.Println()

	// continue — skips to next iteration
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			continue // skip even numbers
		}
		fmt.Print(i, " ") // prints: 1 3 5 7 9
	}
	fmt.Println()

	// Range-based for (used with arrays, slices, maps, strings, channels)
	fruits := []string{"apple", "banana", "cherry"}
	for index, value := range fruits {
		fmt.Printf("index=%d value=%s\n", index, value)
	}

	// Discard index with _
	for _, fruit := range fruits {
		fmt.Println(fruit)
	}

	// Range over string gives runes (Unicode code points), not bytes!
	word := "Hello🚀"
	for i, ch := range word {
		fmt.Printf("index=%d rune=%c\n", i, ch)
	}

	// Range over map
	scores := map[string]int{"Alice": 95, "Bob": 87}
	for name2, score2 := range scores {
		fmt.Printf("%s: %d\n", name2, score2)
	}

	// Labeled break — break out of nested loops
outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == 1 && j == 1 {
				break outer // breaks BOTH loops
			}
			fmt.Printf("i=%d j=%d\n", i, j)
		}
	}

	// --- SWITCH ---
	// Go switch: NO fallthrough by default, NO break needed
	day := "Monday"
	switch day {
	case "Saturday", "Sunday": // multiple values per case
		fmt.Println("Weekend!")
	case "Monday":
		fmt.Println("Start of work week")
	case "Friday":
		fmt.Println("TGIF!")
	default:
		fmt.Println("Midweek")
	}

	// Switch with init statement
	switch os := "linux"; os {
	case "darwin":
		fmt.Println("macOS")
	case "linux":
		fmt.Println("Linux")
	default:
		fmt.Println("Unknown OS")
	}

	// Switch without expression = cleaner if-else chain
	score2 := 85
	switch {
	case score2 >= 90:
		fmt.Println("A grade")
	case score2 >= 80:
		fmt.Println("B grade")
	case score2 >= 70:
		fmt.Println("C grade")
	default:
		fmt.Println("F grade")
	}

	// Explicit fallthrough — rare, but exists
	num2 := 2
	switch num2 {
	case 1:
		fmt.Println("one")
		fallthrough // EXPLICITLY falls to next case
	case 2:
		fmt.Println("two")
		fallthrough
	case 3:
		fmt.Println("three") // this runs too!
	case 4:
		fmt.Println("four") // this does NOT run
	}

	// --- DEFER ---
	// defer: delays execution of a function until the surrounding function returns
	// Arguments are evaluated IMMEDIATELY, but call happens LAST
	fmt.Println("\n--- Defer ---")
	defer fmt.Println("I run last (deferred)")
	fmt.Println("I run first")
	fmt.Println("I run second")

	// Multiple defers — execute in LIFO (Last In, First Out) order
	for i := 0; i < 3; i++ {
		defer fmt.Println("deferred:", i) // prints 2, 1, 0
	}
	fmt.Println("After defer loop")

	// =========================================================================
	// SECTION 9: fmt PACKAGE — Printf formatting verbs
	// =========================================================================
	fmt.Println("\n--- fmt formatting ---")

	type Person struct {
		Name string
		Age  int
	}
	p2 := Person{"Alice", 30}

	fmt.Printf("%v\n", p2)   // {Alice 30}         — default format
	fmt.Printf("%+v\n", p2)  // {Name:Alice Age:30} — with field names
	fmt.Printf("%#v\n", p2)  // main.Person{Name:"Alice", Age:30} — Go syntax
	fmt.Printf("%T\n", p2)   // main.Person         — type
	fmt.Printf("%d\n", 42)   // 42                  — decimal integer
	fmt.Printf("%b\n", 42)   // 101010              — binary
	fmt.Printf("%o\n", 42)   // 52                  — octal
	fmt.Printf("%x\n", 42)   // 2a                  — hex lowercase
	fmt.Printf("%X\n", 42)   // 2A                  — hex uppercase
	fmt.Printf("%f\n", 3.14) // 3.140000            — float
	fmt.Printf("%.2f\n", 3.14159) // 3.14           — 2 decimal places
	fmt.Printf("%e\n", 3.14) // 3.140000e+00        — scientific notation
	fmt.Printf("%s\n", "go") // go                  — string
	fmt.Printf("%q\n", "go") // "go"                — quoted string
	fmt.Printf("%t\n", true) // true                — boolean
	fmt.Printf("%c\n", 65)   // A                   — character from int
	fmt.Printf("%p\n", &a)   // memory address      — pointer

	// Width and padding
	fmt.Printf("|%10d|\n", 42)   // |        42|  — right aligned, width 10
	fmt.Printf("|%-10d|\n", 42)  // |42        |  — left aligned
	fmt.Printf("|%010d|\n", 42)  // |0000000042|  — zero padded

	// Sprintf — format to a string (doesn't print)
	result := fmt.Sprintf("Name: %s, Age: %d", "Achyut", 25)
	fmt.Println(result)

	// =========================================================================
	// SECTION 10: SCOPE — Where variables live
	// =========================================================================
	// Package level: visible to all files in the package
	// Function level: visible within the function
	// Block level: visible within {} braces (if, for, switch, etc.)
	// Go uses LEXICAL scoping

	outer2 := "I am outer"
	{
		inner := "I am inner"
		fmt.Println(outer2, inner) // both visible here
		outer2 = "outer modified"
	}
	// inner is NOT accessible here
	fmt.Println(outer2) // "outer modified"

	// Variable shadowing — inner declaration hides outer
	shadowVar := "original"
	{
		shadowVar := "shadowed" // NEW variable, hides outer
		fmt.Println(shadowVar)  // "shadowed"
	}
	fmt.Println(shadowVar) // "original" — original is unchanged

	// =========================================================================
	// SECTION 11: POINTERS — Deep dive
	// =========================================================================
	fmt.Println("\n--- Pointers ---")

	// A pointer holds a MEMORY ADDRESS
	// *T = pointer to type T
	// & = address-of operator
	// * = dereference operator (get value at address)

	val2 := 42
	ptr2 := &val2 // ptr2 is of type *int

	fmt.Println("val2:", val2)        // 42
	fmt.Println("ptr2:", ptr2)        // 0xc000... (memory address)
	fmt.Println("*ptr2:", *ptr2)      // 42 (value AT that address)
	fmt.Printf("type of ptr2: %T\n", ptr2) // *int

	// Modify through pointer
	*ptr2 = 100 // changes val2 through the pointer
	fmt.Println("val2 after *ptr2 = 100:", val2) // 100

	// new() — allocates zeroed memory and returns a pointer
	p3 := new(int) // *int pointing to a zero-initialized int
	fmt.Println("new(int):", *p3)     // 0
	*p3 = 999
	fmt.Println("after *p3 = 999:", *p3) // 999

	// Pointer to pointer
	val3 := 10
	ptr3 := &val3
	ptr4 := &ptr3 // pointer to pointer (**)

	fmt.Println("val3:", val3)
	fmt.Println("ptr3:", ptr3, "*ptr3:", *ptr3)
	fmt.Println("ptr4:", ptr4, "*ptr4:", *ptr4, "**ptr4:", **ptr4)

	// nil pointer — zero value of a pointer
	var nilPtr *int
	fmt.Println("nil pointer:", nilPtr) // <nil>
	// NEVER dereference a nil pointer — it causes a panic (runtime crash)
	// if nilPtr != nil { fmt.Println(*nilPtr) } // safe pattern

	// Why pointers matter:
	// 1. Modify a variable inside a function (pass by reference)
	// 2. Avoid copying large data structures
	// 3. Signal optional/missing values (nil)

	doubleMe(&val2)
	fmt.Println("val2 after doubleMe:", val2) // 200

	// Go does NOT have pointer arithmetic (unlike C)
	// You CANNOT do: ptr2++ or ptr2 + 1

	// =========================================================================
	// SECTION 12: TYPE INFERENCE — How Go figures out types
	// =========================================================================
	fmt.Println("\n--- Type Inference ---")

	a2 := 42          // int
	b2 := 3.14        // float64
	c2 := "hello"     // string
	d2 := true        // bool
	e2 := 1 + 2i      // complex128

	fmt.Printf("a2: %T = %v\n", a2, a2)
	fmt.Printf("b2: %T = %v\n", b2, b2)
	fmt.Printf("c2: %T = %v\n", c2, c2)
	fmt.Printf("d2: %T = %v\n", d2, d2)
	fmt.Printf("e2: %T = %v\n", e2, e2)

	// Untyped constants adapt to context
	const bigNum = 1000000000000 // could be int64, float64, etc.
	var result2 int64 = bigNum   // works — bigNum adapts to int64
	var result3 float64 = bigNum // works — bigNum adapts to float64
	fmt.Println(result2, result3)

	// =========================================================================
	// SECTION 13: MATH PACKAGE — Common operations
	// =========================================================================
	fmt.Println("\n--- Math ---")

	fmt.Println("Abs(-5):", math.Abs(-5))
	fmt.Println("Ceil(4.1):", math.Ceil(4.1))   // 5
	fmt.Println("Floor(4.9):", math.Floor(4.9)) // 4
	fmt.Println("Round(4.5):", math.Round(4.5)) // 5
	fmt.Println("Sqrt(16):", math.Sqrt(16))     // 4
	fmt.Println("Pow(2,10):", math.Pow(2, 10))  // 1024
	fmt.Println("Max(3,7):", math.Max(3, 7))    // 7
	fmt.Println("Min(3,7):", math.Min(3, 7))    // 3
	fmt.Println("Pi:", math.Pi)
	fmt.Println("MaxInt:", math.MaxInt64)
	fmt.Println("MinInt:", math.MinInt64)

	fmt.Println("\n=== MODULE 01 COMPLETE ===")
}

// Function to demonstrate pointer usage
func doubleMe(n *int) {
	*n = *n * 2
}
