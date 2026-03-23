// 01_why_generics.go
//
// WHY GENERICS?
// =============
// Before Go 1.18 (released March 2022), Go had no generics.
// Every algorithm that needed to work across types required duplication,
// interface{} gymnastics, or code generation. Generics solve this elegantly.
//
// This file walks through the PROBLEM first, then the PRE-GENERICS solutions,
// then explains what generics ARE and ARE NOT in Go.

package main

import (
	"fmt"
	"strings"
)

// =============================================================================
// PART 1: THE PROBLEM — CODE DUPLICATION
// =============================================================================
//
// Suppose you want a "Sum" function. Easy for int:

func sumInt(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Now you need it for float64 too. You MUST duplicate:

func sumFloat64(nums []float64) float64 {
	total := 0.0
	for _, n := range nums {
		total += n
	}
	return total
}

// And for int64, int32, uint, float32... you write them all.
// This is NOT a hypothetical: the Go standard library itself had to do this.
// Look at math/bits — you'll find functions repeated for every integer width.
//
// Same problem with collections. A "Max" function:

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxString(a, b string) string {
	if a > b {
		return a
	}
	return b
}

// Three identical bodies. Only the types differ.
// Multiply this by every algorithm you write. The duplication is enormous.
//
// A "Contains" function for slices:

func containsInt(slice []int, target int) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

func containsString(slice []string, target string) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// Identical structure. The compiler could figure this out. But pre-1.18, it couldn't.

// =============================================================================
// PART 2: PRE-GENERICS SOLUTION #1 — interface{} / any
// =============================================================================
//
// The "any" type (alias for interface{}) can hold any value.
// You can write one function that accepts anything — but with steep costs.

func sumAny(nums []any) any {
	// Problem 1: What type is the result? We must pick one.
	// Problem 2: We don't know what operation to perform without type switching.
	// Problem 3: Type assertions can PANIC at runtime if wrong.

	// Let's try:
	if len(nums) == 0 {
		return 0
	}

	switch nums[0].(type) {
	case int:
		total := 0
		for _, v := range nums {
			// Each element requires a type assertion — verbose and error-prone.
			// If any element is NOT int, this panics at runtime.
			total += v.(int)
		}
		return total
	case float64:
		total := 0.0
		for _, v := range nums {
			total += v.(float64)
		}
		return total
	default:
		panic("unsupported type")
	}
	// We're back to writing type-specific code — but NOW without compile-time safety.
}

// The caller's experience is also terrible:
//
//   result := sumAny([]any{1, 2, 3})
//   // result is type "any" — caller must assert AGAIN
//   total := result.(int)  // panic if wrong
//
// Problems with interface{}:
// 1. No compile-time type checking — errors surface at runtime
// 2. Performance: every value must be "boxed" into an interface (heap allocation)
// 3. Verbose: type assertions everywhere
// 4. Loss of type information: []int → []any is NOT automatic; you must copy
// 5. The function signature lies: "sumAny([]any)" accepts garbage

// =============================================================================
// PART 2B: TYPE ASSERTION VERBOSITY DEMONSTRATED
// =============================================================================

// A "Map" function pre-generics using interface{}:
// Apply function f to each element of slice, return new slice of results.

func mapAny(slice []any, f func(any) any) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}

// Caller must write:
//
//   doubled := mapAny([]any{1, 2, 3}, func(v any) any {
//       return v.(int) * 2  // type assert inside — panic if wrong element type
//   })
//   // doubled is []any{2, 4, 6}
//   // To use it as []int, must convert element by element
//
// This is so painful that most Go programmers just duplicated code instead.

// =============================================================================
// PART 3: PRE-GENERICS SOLUTION #2 — CODE GENERATION
// =============================================================================
//
// Go has "go generate" + tools like stringer, mockgen, or the now-obsolete
// "genny" library. The idea: write a template in a special comment syntax,
// then run a tool that generates type-specific versions.
//
// Example with the "genny" tool (now mostly abandoned):
//
//   // template:
//   //go:generate genny -in=$GOFILE -out=gen-$GOFILE gen "SomeType=int,float64"
//
//   func Sum_SomeType_(nums []SomeType) SomeType {
//       var total SomeType
//       for _, n := range nums {
//           total += n
//       }
//       return total
//   }
//
// The tool would generate sumInt.go and sumFloat64.go with the types substituted.
//
// Problems with code generation:
// 1. Complex build pipeline — must run go generate before go build
// 2. Generated files bloat the repository
// 3. Error messages point to generated files, not the template
// 4. Templates are non-standard Go, IDE support is poor
// 5. Debugging is harder — breakpoints in generated code
// 6. Tool fragmentation — every team uses different generators

// =============================================================================
// PART 4: WHY GO WAITED UNTIL 1.18
// =============================================================================
//
// Go was designed in 2007. Generics were discussed from the very beginning.
// Rob Pike, Ken Thompson, and Robert Griesemer intentionally left them out.
// WHY?
//
// 1. COMPLEXITY RISK
//    C++ templates are notoriously complex. Error messages for template failures
//    are notoriously hard to read. Java generics have type erasure, leading to
//    surprising runtime behavior. The Go team didn't want to inherit those sins.
//
// 2. FINDING THE RIGHT DESIGN
//    Over a decade, multiple proposals were made and rejected:
//    - 2010: "generics" proposal — too template-like
//    - 2013: "generalized types" by Griesemer — progress, but not quite right
//    - 2018: contract-based proposal — promising but contracts were confusing
//    - 2019: contracts replaced by interfaces as constraints — breakthrough
//    - 2021: "type parameters" proposal accepted, implemented in 1.18 (2022)
//
// 3. THE INSIGHT: INTERFACES AS CONSTRAINTS
//    The 2019 breakthrough: what if constraints ARE interfaces?
//    Go already had interfaces. Type sets (what types satisfy an interface)
//    already existed conceptually. The generics proposal extended interfaces
//    to describe not just method sets but TYPE sets.
//    This reused existing mental models instead of introducing a new concept.
//
// 4. SIMPLICITY PRESERVED
//    Go generics are deliberately less powerful than C++ templates.
//    There is no template specialization (different implementation per type).
//    There is no variadic templates.
//    There are no non-type template parameters (no Stack<int, 10>).
//    These omissions are intentional — complexity was rejected.
//
// The result: a generics system that's expressive enough for the common cases,
// simple enough to understand in an afternoon, and consistent with existing Go idioms.

// =============================================================================
// PART 5: WHAT GENERICS ARE (in Go)
// =============================================================================
//
// Generics in Go = TYPE PARAMETERS
//
// A function or type can declare that it works with "any type T that satisfies
// some constraint." The constraint is expressed as an interface.
//
// Syntax preview (details in 02_type_parameters.go):
//
//   func Sum[T int | float64](nums []T) T {
//       var total T
//       for _, n := range nums {
//           total += n
//       }
//       return total
//   }
//
// When you call Sum[int](...) or Sum[float64](...), the compiler generates
// (or monomorphizes) a version of Sum specifically for that type.
// The type is checked at COMPILE TIME — no runtime surprises.
//
// Key properties:
// - Type safety: the compiler verifies constraints at the call site
// - Single source of truth: one function body, not N duplicated ones
// - No runtime type assertions needed
// - The function signature is honest about what it accepts

// =============================================================================
// PART 6: WHAT GENERICS ARE NOT (in Go)
// =============================================================================
//
// NOT TEMPLATES (like C++):
//   C++ templates are processed textually — the template body is copied and
//   substituted, then compiled. Errors surface late, messages are cryptic.
//   Go generics are compiled with constraints checked BEFORE instantiation.
//   The constraint defines exactly what operations T supports. If your constraint
//   doesn't include "+", you cannot write "a + b" — compiler error immediately.
//
// NOT MACROS:
//   Macros (C #define, Rust macros) operate at the syntactic level.
//   They can do arbitrary code transformations. Go has no macro system.
//   Generics work purely within the type system — they cannot generate new
//   method names, manipulate syntax trees, or do compile-time reflection.
//
// NOT TEMPLATES WITH SPECIALIZATION:
//   In C++, you can write a special version of a template for a specific type.
//   In Go, you cannot. One constraint, one body. If you need different behavior
//   for different types, you use interfaces (method dispatch) not generics.
//
// NOT A RUNTIME FEATURE:
//   Unlike Java generics (which use type erasure and work with Object at runtime),
//   Go generics are fully resolved at compile time. The type parameter T is NOT
//   present at runtime through reflection in the way Java's generics are.
//   (Go uses GC shapes for code sharing — but that's an implementation detail.)
//
// NOT A REPLACEMENT FOR INTERFACES:
//   Interfaces express: "this value responds to these method calls."
//   Use interfaces when you need runtime polymorphism — when the TYPE of a value
//   isn't known until the program runs (e.g., from config, user input, plugins).
//
//   Generics express: "this algorithm works on any type satisfying this constraint."
//   Use generics for algorithms and data structures where the TYPE IS KNOWN
//   at compile time but you want to avoid duplication.
//
//   Rule of thumb:
//     - If you're writing an algorithm → generics
//     - If you're writing a plugin/extension point → interfaces
//     - If types are determined at runtime → interfaces

// =============================================================================
// PART 7: THE PROMISE OF GENERICS — A PREVIEW
// =============================================================================
//
// With generics, here's what the Sum problem looks like:

// NumberConstraint allows any numeric type that supports addition.
// (We'll explore constraint syntax deeply in 03_constraints.go)
type NumberConstraint interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Sum works for ALL numeric types. One body. Compile-time safe.
func Sum[T NumberConstraint](nums []T) T {
	var total T // zero value of T — 0 for ints, 0.0 for floats
	for _, n := range nums {
		total += n // the constraint guarantees + is valid on T
	}
	return total
}

// Max works for any ordered type. One body.
func Max[T interface{ ~int | ~float64 | ~string }](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Contains works for any comparable type. One body.
func Contains[T comparable](slice []T, target T) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// =============================================================================
// MAIN — demonstrating everything
// =============================================================================

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("WHY GENERICS — Demonstrating the problem and solution")
	fmt.Println(strings.Repeat("=", 60))

	// --- PART 1: Pre-generics duplication ---
	fmt.Println("\n--- Pre-generics: type-specific functions ---")
	fmt.Println("sumInt([1,2,3,4,5])   =", sumInt([]int{1, 2, 3, 4, 5}))
	fmt.Println("sumFloat64([1.1,2.2]) =", sumFloat64([]float64{1.1, 2.2, 3.3}))
	fmt.Println("maxInt(7, 3)          =", maxInt(7, 3))
	fmt.Println("maxFloat64(7.5, 3.5)  =", maxFloat64(7.5, 3.5))
	fmt.Println("maxString(\"go\",\"rust\")=", maxString("go", "rust"))
	fmt.Println("containsInt([1,2,3],2)=", containsInt([]int{1, 2, 3}, 2))
	fmt.Println("containsString([a,b],c)=", containsString([]string{"a", "b"}, "c"))

	// --- PART 2: interface{} approach —  dangerous ---
	fmt.Println("\n--- interface{}/any approach: type assertions required ---")
	resultAny := sumAny([]any{10, 20, 30})
	fmt.Printf("sumAny result: %v (type: %T) — caller must assert to use\n", resultAny, resultAny)

	// Demonstrate the boxing overhead conceptually:
	// []int{1,2,3} cannot be passed as []any — must copy each element
	ints := []int{1, 2, 3}
	anySlice := make([]any, len(ints))
	for i, v := range ints {
		anySlice[i] = v // each int is "boxed" into an interface — heap allocation
	}
	fmt.Printf("Converting []int to []any requires manual copy: %v\n", anySlice)

	// --- PART 3: The generic solution ---
	fmt.Println("\n--- Generic solution: one body, full type safety ---")
	fmt.Println("Sum[int]([1,2,3,4,5])       =", Sum([]int{1, 2, 3, 4, 5}))         // type inferred
	fmt.Println("Sum[float64]([1.1,2.2,3.3]) =", Sum([]float64{1.1, 2.2, 3.3}))     // explicit
	fmt.Println("Sum[int64]([10,20,30])       =", Sum([]int64{10, 20, 30}))           // new type, no new function
	fmt.Println("Max(7, 3)                    =", Max(7, 3))                          // int, inferred
	fmt.Println("Max(7.5, 3.5)               =", Max(7.5, 3.5))                      // float64, inferred
	fmt.Println("Max(\"go\", \"rust\")          =", Max("go", "rust"))                // string, inferred
	fmt.Println("Contains([1,2,3], 2)         =", Contains([]int{1, 2, 3}, 2))       // int
	fmt.Println("Contains([a,b,c], d)         =", Contains([]string{"a", "b"}, "d")) // string

	// --- What the compiler catches ---
	fmt.Println("\n--- What generics catch at compile time ---")
	fmt.Println("The following would NOT compile (try uncommenting):")
	fmt.Println("  Sum([]string{\"a\",\"b\"})  — string does not satisfy NumberConstraint")
	fmt.Println("  Max([]int{1}, 2)         — mismatched types between arguments")
	fmt.Println("  Contains([][]int{}, ...) — slices are not comparable")

	// --- Key takeaway ---
	fmt.Println("\n--- Key Takeaway ---")
	fmt.Println("Pre-generics: 3 types × 5 algorithms = 15 functions")
	fmt.Println("With generics: 5 generic functions = 5 functions (+ compile-time safety)")
	fmt.Println()
	fmt.Println("Generics are NOT about runtime flexibility (that's interfaces).")
	fmt.Println("Generics are about COMPILE-TIME REUSE without duplication.")
	fmt.Println()
	fmt.Println("The rule:")
	fmt.Println("  - Algorithm over many known types? → Generic function")
	fmt.Println("  - Runtime polymorphism / plugin point? → Interface")
}
