// FILE: 01_fundamentals/08_control_flow.go
// TOPIC: Control Flow — if/else, switch, for (all 5 forms), goto, labels
//
// Run: go run 01_fundamentals/08_control_flow.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Go deliberately has ONE loop construct (for) instead of for/while/do-while.
//   Its switch is more powerful than C (no fallthrough by default, can match
//   anything, not just integers). Knowing all these forms lets you write
//   idiomatic Go instead of translating patterns from other languages.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math/rand"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Control Flow")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// IF / ELSE IF / ELSE
	// ─────────────────────────────────────────────────────────────────────
	//
	// Differences from C/Java:
	//   - No parentheses around condition: if x > 0 { } (not if (x > 0) {})
	//   - Braces are REQUIRED, even for single-line bodies
	//   - if can have an INITIALIZATION STATEMENT before the condition
	//
	// The init-statement form is VERY idiomatic Go:
	//   if err := doSomething(); err != nil { ... }
	//
	// WHY init-statement?
	//   The variable (err, value, etc.) is scoped to the if/else block only.
	//   It doesn't pollute the surrounding function scope.
	//   This is the #1 pattern you'll see in real Go codebases.

	score := 75

	fmt.Printf("\n── if/else if/else ──\n")
	if score >= 90 {
		fmt.Println("  Grade: A")
	} else if score >= 80 {
		fmt.Println("  Grade: B")
	} else if score >= 70 {
		fmt.Println("  Grade: C")
	} else {
		fmt.Println("  Grade: F")
	}

	// Init-statement pattern — the idiomatic Go way:
	fmt.Println("\n  Init-statement form:")
	if n := rand.Intn(10); n%2 == 0 {
		fmt.Printf("  n=%d is even (n scoped to this if block)\n", n)
	} else {
		fmt.Printf("  n=%d is odd (n scoped to this if/else block)\n", n)
	}
	// 'n' is NOT accessible here — it was scoped to the if block above
	// fmt.Println(n)  ← compile error

	// ─────────────────────────────────────────────────────────────────────
	// SWITCH STATEMENT
	// ─────────────────────────────────────────────────────────────────────
	//
	// Go's switch is more powerful than C's:
	//
	// 1. NO IMPLICIT FALLTHROUGH:
	//    In C/Java, execution falls through to the next case unless you break.
	//    In Go, each case AUTOMATICALLY breaks. You must explicitly say 'fallthrough'
	//    to continue to the next case (and you rarely want this).
	//
	// 2. CASES CAN HAVE MULTIPLE VALUES:
	//    case 1, 2, 3:  (no need for multiple case lines)
	//
	// 3. CASES CAN BE EXPRESSIONS:
	//    case x > 0:  (not just constants/literals)
	//
	// 4. EXPRESSION-LESS SWITCH (switch without a value):
	//    Equivalent to switch true — each case is a boolean expression.
	//    This replaces long if/else if chains nicely.
	//
	// 5. TYPE SWITCH:
	//    switch v := x.(type) { case int: ... case string: ... }
	//    (Covered in Module 03)

	day := "Wednesday"
	fmt.Printf("\n── switch (day=%q) ──\n", day)
	switch day {
	case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
		fmt.Println("  Weekday")
	case "Saturday", "Sunday":
		fmt.Println("  Weekend")
	default:
		fmt.Println("  Unknown day")
	}

	// Switch with init statement:
	fmt.Println("\n  Switch with init statement:")
	switch x := rand.Intn(5); {
	case x == 0:
		fmt.Println("  Got zero")
	case x < 3:
		fmt.Printf("  Got small value: %d\n", x)
	default:
		fmt.Printf("  Got large value: %d\n", x)
	}

	// Expression-less switch (replaces if/else chain):
	temp := 35  // degrees celsius
	fmt.Printf("\n  Expression-less switch (temp=%d°C):\n", temp)
	switch {
	case temp < 0:
		fmt.Println("  Freezing")
	case temp < 15:
		fmt.Println("  Cold")
	case temp < 25:
		fmt.Println("  Comfortable")
	case temp < 35:
		fmt.Println("  Warm")
	default:
		fmt.Println("  Hot")
	}

	// fallthrough: explicitly continue to next case
	fmt.Println("\n  Explicit fallthrough:")
	switch 2 {
	case 1:
		fmt.Println("  case 1")
		fallthrough
	case 2:
		fmt.Println("  case 2")
		fallthrough  // continues to case 3 regardless of 3's condition
	case 3:
		fmt.Println("  case 3 (fell through)")
	case 4:
		fmt.Println("  case 4 (NOT reached, fallthrough stops at case 3)")
	}

	// ─────────────────────────────────────────────────────────────────────
	// FOR LOOP — Go's ONLY looping construct (replaces for/while/do-while)
	// ─────────────────────────────────────────────────────────────────────
	//
	// FORM 1: Classic C-style for loop
	//   for init; condition; post { }
	//   All three parts are optional.

	fmt.Printf("\n── for loops ──\n")
	fmt.Print("  Form 1 (classic):  ")
	for i := 0; i < 5; i++ {
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// FORM 2: While-style (condition only)
	//   for condition { }
	//   When init and post are omitted, it looks like a while loop.
	fmt.Print("  Form 2 (while):    ")
	i := 0
	for i < 5 {
		fmt.Printf("%d ", i)
		i++
	}
	fmt.Println()

	// FORM 3: Infinite loop
	//   for { }
	//   Runs forever. Use break to exit, or return.
	//   Use case: event loops, servers, background workers.
	fmt.Print("  Form 3 (infinite): ")
	j := 0
	for {
		if j >= 5 {
			break  // exit the infinite loop
		}
		fmt.Printf("%d ", j)
		j++
	}
	fmt.Println()

	// FORM 4: Range over slice/array
	//   for index, value := range collection { }
	//   Returns index AND value. Use _ to discard either.
	nums := []int{10, 20, 30, 40, 50}
	fmt.Print("  Form 4 (range slice):  ")
	for i, v := range nums {
		fmt.Printf("[%d]=%d ", i, v)
	}
	fmt.Println()

	// Range: index only (discard value)
	fmt.Print("  Range indices only:    ")
	for i := range nums {
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// Range: value only (discard index)
	fmt.Print("  Range values only:     ")
	for _, v := range nums {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// FORM 5: Range over string (gives runes, not bytes)
	fmt.Print("  Form 5 (range string): ")
	for i, r := range "Hello" {
		fmt.Printf("[%d]=%c ", i, r)
	}
	fmt.Println()

	// Range over map (order is RANDOM every time — Go randomizes for security):
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Print("  Range over map (random order): ")
	for k, v := range m {
		fmt.Printf("%s:%d ", k, v)
	}
	fmt.Println()

	// Range over channel (discussed in Module 06):
	// for v := range ch { }  ← reads until channel is closed

	// ─────────────────────────────────────────────────────────────────────
	// BREAK AND CONTINUE
	// ─────────────────────────────────────────────────────────────────────
	//
	// break:    exits the innermost for/switch/select
	// continue: skips the rest of the current iteration, goes to next

	fmt.Println("\n── break and continue ──")
	fmt.Print("  Skip evens (continue): ")
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			continue  // skip even numbers
		}
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	fmt.Print("  Stop at 5 (break):     ")
	for i := 0; i < 10; i++ {
		if i == 5 {
			break  // stop at 5
		}
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────
	// LABELED BREAK / CONTINUE — Breaking out of NESTED loops
	// ─────────────────────────────────────────────────────────────────────
	//
	// In nested loops, plain break/continue only affects the INNERMOST loop.
	// Labels let you break/continue an OUTER loop.
	//
	// Syntax: put a label before the loop, then use break Label / continue Label.
	//
	// USE CASE: searching a 2D grid, exit as soon as you find what you need.

	fmt.Println("\n── labeled break (exit nested loops) ──")
	target := 6
	fmt.Printf("  Searching for %d in 3x3 grid:\n", target)

outer:
	for row := 1; row <= 3; row++ {
		for col := 1; col <= 3; col++ {
			val := row*col
			fmt.Printf("    row=%d col=%d val=%d\n", row, col, val)
			if val == target {
				fmt.Printf("  Found %d! Breaking outer loop.\n", target)
				break outer  // exits BOTH loops
			}
		}
	}
	fmt.Println("  After labeled break (execution continues here)")

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  if: no parens needed, init-statement is idiomatic")
	fmt.Println("  switch: no implicit fallthrough, cases can be expressions")
	fmt.Println("  for: Go's ONLY loop — 5 forms cover every use case")
	fmt.Println("    classic | while-style | infinite | range slice | range string")
	fmt.Println("  break/continue: affect innermost loop")
	fmt.Println("  Label break/continue: exit nested loops cleanly")
}
