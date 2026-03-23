// FILE: 01_fundamentals/02_variables/02_variables.go
// TOPIC: Variables — All Declaration Styles, Zero Values, Scope, Shadowing
//
// Run: go run 01_fundamentals/02_variables/02_variables.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Go has 4 ways to declare variables. Each exists for a reason.
//   Knowing WHICH to use and WHEN separates readable Go from noisy Go.
//   Zero values and scope rules are sources of subtle bugs — knowing them
//   cold prevents hours of debugging.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// PACKAGE-LEVEL VARIABLES
// ─────────────────────────────────────────────────────────────────────────────
//
// These are declared outside any function. They:
//   - Exist for the entire lifetime of the program
//   - Are accessible from any function in this package
//   - Are initialized before main() runs
//   - CANNOT use := (short declaration is only allowed inside functions)
//
// Use package-level vars sparingly. They create implicit dependencies
// and make functions harder to test (functions that read global state are
// harder to reason about than functions that take parameters).
// Common legitimate uses: configuration, loggers, DB connection pools.

var appName string = "GoMastery" // explicit type + value
var buildID = "abc123"           // type inferred from value (string)
var debugMode bool               // zero value = false (no value given)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Variables")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// THE 4 WAYS TO DECLARE VARIABLES
	// ─────────────────────────────────────────────────────────────────────

	// WAY 1: var with explicit type and value
	// Use when: you want to be crystal clear about the type,
	// especially when the right-hand side value doesn't obviously convey it.
	// Example: var timeout time.Duration = 30 * time.Second
	var age int = 25
	fmt.Printf("Way 1 — explicit type + value: age = %d (type: %T)\n", age, age)

	// WAY 2: var with type inference
	// Go infers the type from the right-hand side expression.
	// The inferred type follows Go's DEFAULT type rules:
	//   untyped integer constant → int
	//   untyped float constant   → float64
	//   untyped string literal   → string
	//   untyped bool             → bool
	var name = "Achyut" // inferred: string
	var score = 98.5    // inferred: float64  (NOT float32!)
	var passed = true   // inferred: bool
	fmt.Printf("Way 2 — type inference: name=%q score=%.1f passed=%v\n", name, score, passed)

	// WAY 3: var with zero value (no assignment)
	// Every type in Go has a ZERO VALUE — the default value when no
	// initializer is given. This is one of Go's most important design decisions:
	// there is NO concept of "uninitialized memory" in Go.
	//
	//   int, int8..int64, uint*  → 0
	//   float32, float64         → 0.0
	//   bool                     → false
	//   string                   → "" (empty string, not nil!)
	//   pointer                  → nil
	//   slice                    → nil  (but len=0, cap=0, safe to append)
	//   map                      → nil  (reading returns zero value, writing PANICS)
	//   channel                  → nil
	//   interface                → nil
	//   struct                   → each field gets its zero value
	//   array                    → each element gets its zero value
	//
	// WHY ZERO VALUES MATTER:
	//   In C/C++, uninitialized variables have garbage values → undefined behavior.
	//   In Go, every variable is always in a valid, predictable state.
	//   This eliminates a whole class of bugs.
	var counter int   // 0
	var ratio float64 // 0.0
	var active bool   // false
	var label string  // ""
	fmt.Printf("Way 3 — zero values: counter=%d ratio=%f active=%v label=%q\n",
		counter, ratio, active, label)

	// WAY 4: Short variable declaration :=
	// This is the MOST COMMON style inside functions.
	// := is syntactic sugar for: infer type + declare + assign in one step.
	//
	// RULES:
	//   - Only valid INSIDE functions (not at package level)
	//   - At least ONE variable on the left must be NEW (not already declared)
	//   - If some vars on the left already exist, they are REASSIGNED, not re-declared
	city := "Kolkata" // new variable, type string
	count := 10       // new variable, type int
	pi := 3.14159     // new variable, type float64
	fmt.Printf("Way 4 — short decl: city=%q count=%d pi=%.5f\n", city, count, pi)

	// := with multiple variables — common for function returns
	// Here 'count' already exists, so it's reassigned; 'err' is new.
	// This is VALID because at least one (err) is new:
	count, ok := 42, true
	fmt.Printf("  Re-assignment + new: count=%d ok=%v\n", count, ok)

	// ─────────────────────────────────────────────────────────────────────
	// BLOCK DECLARATIONS — Clean style for multiple related vars
	// ─────────────────────────────────────────────────────────────────────
	//
	// When declaring several variables together, the var() block is cleaner
	// and signals "these variables belong together conceptually".
	// Often used at package level for grouped configuration constants/vars.
	var (
		firstName  = "Achyut"
		lastName   = "Pal"
		rollNo     = 42
		isEnrolled = true
	)
	fmt.Printf("\nBlock var: %s %s, roll=%d, enrolled=%v\n",
		firstName, lastName, rollNo, isEnrolled)

	// ─────────────────────────────────────────────────────────────────────
	// MULTIPLE ASSIGNMENT — swap without a temp variable
	// ─────────────────────────────────────────────────────────────────────
	//
	// Go evaluates the RIGHT side completely before assigning to the left.
	// This makes swapping trivial — no temp variable needed.
	a, b := 10, 20
	fmt.Printf("\nBefore swap: a=%d b=%d\n", a, b)
	a, b = b, a // simultaneous assignment
	fmt.Printf("After swap:  a=%d b=%d\n", a, b)

	// ─────────────────────────────────────────────────────────────────────
	// THE BLANK IDENTIFIER _
	// ─────────────────────────────────────────────────────────────────────
	//
	// Go's compiler ERRORS if you declare a variable and don't use it.
	// This prevents dead code and forces you to think about what you need.
	//
	// When you DON'T need a value (e.g., one return value of a multi-return func),
	// use _ (blank identifier) to explicitly discard it.
	// _ is not actually a variable — it's a write-only sink.
	x, _ := divide(10, 3) // we only need the quotient, not the remainder
	fmt.Printf("\nBlank identifier: 10/3 quotient = %d (remainder discarded)\n", x)

	// ─────────────────────────────────────────────────────────────────────
	// VARIABLE SCOPE — Where a variable is visible
	// ─────────────────────────────────────────────────────────────────────
	//
	// Go uses LEXICAL SCOPING (also called block scoping).
	// A variable is visible from its declaration until the end of the
	// enclosing {} block.
	//
	// Scopes nest: inner blocks can see outer variables, but outer blocks
	// CANNOT see inner variables.
	{
		// This variable only exists inside this block
		innerVar := "I only exist in this block"
		fmt.Println("\nInner scope:", innerVar)
	}
	// fmt.Println(innerVar) ← compile error: undefined: innerVar

	// ─────────────────────────────────────────────────────────────────────
	// SHADOWING — The #1 subtle bug in Go variable scoping
	// ─────────────────────────────────────────────────────────────────────
	//
	// When you declare a variable in an inner scope with the same name as
	// an outer variable, the inner one SHADOWS the outer one.
	// Inside the inner scope, references to that name hit the INNER variable.
	// The outer variable is unchanged.
	//
	// This is valid Go but often causes bugs, especially with := in if blocks.

	result := "outer"
	fmt.Printf("\nBefore shadow block: result = %q\n", result)
	{
		result := "inner" // NEW variable, shadows the outer 'result'
		fmt.Printf("Inside shadow block: result = %q\n", result)
	}
	fmt.Printf("After shadow block:  result = %q (outer unchanged!)\n", result)

	// COMMON BUG with := in if/else:
	//
	//   err := doSomething()    // outer err
	//   if condition {
	//       result, err := doOther()  // BUG: := creates a NEW err in this block
	//       _ = result                // outer err is never set!
	//   }
	//
	// FIX: declare result before the if, use = inside:
	//   var result string
	//   err := doSomething()
	//   if condition {
	//       result, err = doOther()  // = assigns to the OUTER err and result
	//   }

	// ─────────────────────────────────────────────────────────────────────
	// PACKAGE-LEVEL VARS (declared at top of file)
	// ─────────────────────────────────────────────────────────────────────
	fmt.Printf("\nPackage-level vars: appName=%q buildID=%q debugMode=%v\n",
		appName, buildID, debugMode)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  var x T = v    → explicit, verbose, use at package level")
	fmt.Println("  var x = v      → inferred type, verbose")
	fmt.Println("  var x T        → zero value, signals intentional init-later")
	fmt.Println("  x := v         → idiomatic inside functions")
	fmt.Println("  _ = v          → discard unwanted values")
	fmt.Println("  Watch out for SHADOWING with := inside nested blocks!")
}

// helper used above
func divide(a, b int) (int, int) {
	return a / b, a % b
}
