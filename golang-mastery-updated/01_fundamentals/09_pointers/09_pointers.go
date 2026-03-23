// FILE: 01_fundamentals/09_pointers.go
// TOPIC: Pointers — What they are, value vs pointer semantics, when to use each
//
// Run: go run 01_fundamentals/09_pointers.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Pointer vs value semantics is THE most important decision in every Go
//   function signature. Get it wrong and you either get unexpected mutations,
//   or you waste memory copying large structs. Understanding nil pointers
//   prevents the most common runtime panic in Go programs.
//   Unlike C, Go pointers cannot do arithmetic — this is a safety design choice.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// WHAT IS A POINTER?
// ─────────────────────────────────────────────────────────────────────────────
//
// A pointer is a variable that holds the MEMORY ADDRESS of another variable.
// Think of memory as a huge array of bytes, each with an address (like a
// house number on a street). A pointer is a house number, not the house.
//
// int  x = 42      → x holds the VALUE 42, lives at some address, say 0x100
// *int p = &x      → p holds the ADDRESS 0x100
//
// *p = 99          → go to address 0x100, store 99 there → x is now 99
// fmt.Println(*p)  → go to address 0x100, read the value → prints 99
//
// In Go, pointers are SAFE:
//   - No pointer arithmetic (you cannot do p+1 to get the next element)
//   - The garbage collector tracks pointers, so no dangling pointers
//     (as long as you don't use the unsafe package)
//   - The zero value of any pointer type is nil

type Person struct {
	Name string
	Age  int
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Pointers")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// BASIC POINTER MECHANICS
	// ─────────────────────────────────────────────────────────────────────

	x := 42
	p := &x  // & = "address of" operator. p is type *int

	fmt.Printf("\n── Basic pointer mechanics ──\n")
	fmt.Printf("  x   = %d  (value)\n", x)
	fmt.Printf("  &x  = %p  (address of x, type *int)\n", &x)
	fmt.Printf("  p   = %p  (p stores the address of x)\n", p)
	fmt.Printf("  *p  = %d  (* = dereference: get value AT the address)\n", *p)

	// Modifying through pointer:
	*p = 100  // go to the address p holds, store 100 there
	fmt.Printf("\n  After *p = 100:\n")
	fmt.Printf("  x = %d  (x changed because p points to x)\n", x)
	fmt.Printf("  *p = %d\n", *p)

	// ─────────────────────────────────────────────────────────────────────
	// VALUE SEMANTICS: Functions get a COPY by default
	// ─────────────────────────────────────────────────────────────────────
	//
	// By default, Go passes arguments BY VALUE — a copy is made.
	// The function works on the copy. The original is unchanged.
	//
	// This is SAFE: callers know their data won't be modified.
	// This is COSTLY for large data: copying a 100-field struct is expensive.

	n := 10
	fmt.Printf("\n── Value semantics (pass by copy) ──\n")
	fmt.Printf("  Before doubleByValue(n): n = %d\n", n)
	doubleByValue(n)
	fmt.Printf("  After  doubleByValue(n): n = %d  (unchanged!)\n", n)

	// ─────────────────────────────────────────────────────────────────────
	// POINTER SEMANTICS: Pass the address so function can modify original
	// ─────────────────────────────────────────────────────────────────────
	//
	// Pass &n (address of n) → function receives *int (pointer to int).
	// Function modifies the value AT that address → original changes.

	fmt.Printf("\n── Pointer semantics (pass address) ──\n")
	fmt.Printf("  Before doubleByPointer(&n): n = %d\n", n)
	doubleByPointer(&n)
	fmt.Printf("  After  doubleByPointer(&n): n = %d  (changed!)\n", n)

	// ─────────────────────────────────────────────────────────────────────
	// VALUE vs POINTER FOR STRUCTS — The big practical question
	// ─────────────────────────────────────────────────────────────────────
	//
	// When passing a struct to a function, you choose:
	//
	// VALUE (copy):
	//   func f(p Person) { }      → function gets a full copy of Person
	//   PRO: caller's Person is safe from modification
	//   CON: entire struct copied, expensive for large structs
	//   USE WHEN: small struct, you want immutability, pure function
	//
	// POINTER:
	//   func f(p *Person) { }    → function gets the address
	//   PRO: no copy, function can modify the original
	//   CON: caller's data can be modified (requires documentation/discipline)
	//   USE WHEN: large struct (avoid copy), you NEED to modify the original,
	//             or you want nil to represent "no value"

	alice := Person{Name: "Alice", Age: 30}
	fmt.Printf("\n── Struct: value vs pointer ──\n")
	fmt.Printf("  alice before updateByValue:   %+v\n", alice)
	updateByValue(alice)
	fmt.Printf("  alice after  updateByValue:   %+v (unchanged)\n", alice)
	updateByPointer(&alice)
	fmt.Printf("  alice after  updateByPointer: %+v (modified!)\n", alice)

	// ─────────────────────────────────────────────────────────────────────
	// new() FUNCTION — Allocate a zero-value and return its pointer
	// ─────────────────────────────────────────────────────────────────────
	//
	// new(T) allocates memory for a T, initializes it to its zero value,
	// and returns *T (a pointer to it).
	//
	// This is rarely used in Go because struct literals (&Person{}) are
	// more expressive. But new() is useful for primitive types.

	pi := new(int)      // allocates int(0), returns *int
	ps := new(string)   // allocates string(""), returns *string

	fmt.Printf("\n── new() function ──\n")
	fmt.Printf("  new(int)    = %p, value = %d\n", pi, *pi)
	fmt.Printf("  new(string) = %p, value = %q\n", ps, *ps)

	*pi = 42
	fmt.Printf("  After *pi=42: %d\n", *pi)

	// Using & with struct literal (more idiomatic than new()):
	pPerson := &Person{Name: "Bob", Age: 25}  // allocates Person, returns *Person
	fmt.Printf("  &Person{...} = %+v\n", *pPerson)

	// ─────────────────────────────────────────────────────────────────────
	// NIL POINTER — The zero value of any pointer type
	// ─────────────────────────────────────────────────────────────────────
	//
	// Every pointer type has a zero value of nil.
	// nil means "this pointer points to nothing".
	//
	// DEREFERENCING nil → PANIC
	//   var p *int = nil
	//   *p = 5  → panic: runtime error: invalid memory address or nil pointer dereference
	//
	// ALWAYS check for nil before dereferencing a pointer you didn't control:
	//   if p != nil { *p = 5 }
	//
	// nil pointer panics are Go's equivalent of NullPointerException in Java.
	// They are the most common runtime error in Go programs.

	var nilPtr *int  // zero value = nil
	fmt.Printf("\n── nil pointer ──\n")
	fmt.Printf("  var nilPtr *int = %v  (zero value is nil)\n", nilPtr)
	fmt.Printf("  nilPtr == nil : %v\n", nilPtr == nil)

	// Safe dereference pattern:
	safeDeref(nilPtr)
	nonNil := 42
	safeDeref(&nonNil)

	// ─────────────────────────────────────────────────────────────────────
	// POINTER TO POINTER — Rare but valid
	// ─────────────────────────────────────────────────────────────────────

	v := 10
	ptr1 := &v      // *int points to v
	ptr2 := &ptr1   // **int points to ptr1

	fmt.Printf("\n── Pointer to pointer ──\n")
	fmt.Printf("  v     = %d\n", v)
	fmt.Printf("  *ptr1 = %d\n", *ptr1)
	fmt.Printf("  **ptr2= %d\n", **ptr2)

	**ptr2 = 99  // modify v through ptr2
	fmt.Printf("  After **ptr2=99: v=%d\n", v)

	// ─────────────────────────────────────────────────────────────────────
	// WHEN TO USE POINTER vs VALUE — Decision Guide
	// ─────────────────────────────────────────────────────────────────────

	fmt.Println("\n── When to use pointer vs value ──")
	fmt.Println(`
  USE POINTER when:
    1. The function needs to MODIFY the argument
    2. The struct is LARGE (copying is expensive) — rough threshold: >100 bytes
    3. You need nil to represent "optional" / "no value"
    4. Maintaining consistency: if some methods use pointer receiver,
       use pointer everywhere for that type (covered in Module 03)

  USE VALUE when:
    1. Small types (int, float, small structs)
    2. The function should NOT modify the argument (safety)
    3. You want pure/functional behavior (no side effects)
    4. When working with immutable data

  RULE OF THUMB from Go team:
    - Structs with > 3-4 fields: use pointer
    - Structs with 1-2 primitive fields: use value
    - When in doubt: use pointer (a little overhead is worth the flexibility)
`)

	fmt.Println("─── SUMMARY ────────────────────────────────")
	fmt.Println("  &x  → address of x (type: *T)")
	fmt.Println("  *p  → value at address p (dereference)")
	fmt.Println("  new(T) → allocate zero T, return *T")
	fmt.Println("  nil  → zero value for any pointer type")
	fmt.Println("  ALWAYS check nil before dereferencing")
	fmt.Println("  Go pointers: safe (no arithmetic), GC-managed")
}

// Value receiver — gets a COPY of n
func doubleByValue(n int) {
	n = n * 2  // only the copy is modified
	fmt.Printf("    inside doubleByValue: n=%d (copy modified)\n", n)
}

// Pointer receiver — can modify the original
func doubleByPointer(n *int) {
	*n = *n * 2  // modify the value at the address
	fmt.Printf("    inside doubleByPointer: *n=%d (original modified)\n", *n)
}

func updateByValue(p Person) {
	p.Age = 999  // only the copy is modified
}

func updateByPointer(p *Person) {
	p.Age = 999  // modifies the original Person's Age field
}

func safeDeref(p *int) {
	if p == nil {
		fmt.Println("    safeDeref: pointer is nil, skipping")
		return
	}
	fmt.Printf("    safeDeref: value = %d\n", *p)
}
