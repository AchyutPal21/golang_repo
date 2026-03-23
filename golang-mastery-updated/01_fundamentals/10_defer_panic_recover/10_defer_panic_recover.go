// FILE: 01_fundamentals/10_defer_panic_recover.go
// TOPIC: defer, panic, recover — Go's unique error/cleanup mechanism
//
// Run: go run 01_fundamentals/10_defer_panic_recover.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   defer is one of Go's most elegant features — it guarantees cleanup code
//   runs regardless of how a function exits (return, panic, even os.Exit? No —
//   defer does NOT run on os.Exit). It completely replaces try/finally.
//   panic/recover is Go's emergency error mechanism — very different from
//   exceptions in Java/Python. Knowing WHEN to use (and NOT use) panic is
//   critical to writing idiomatic Go.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: defer, panic, recover")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// DEFER — Schedule a function call to run when the surrounding function returns
	// ─────────────────────────────────────────────────────────────────────
	//
	// defer statement: defers execution of a function until the surrounding
	// function returns — whether it returns normally, panics, or via any path.
	//
	// KEY RULES:
	//   1. Deferred calls run in LIFO order (last deferred = first to run)
	//   2. Arguments to the deferred function are EVALUATED IMMEDIATELY
	//      (at the time defer is called), not when it actually runs
	//   3. Deferred functions CAN read and modify named return values
	//      (powerful and subtle — used for cleanup and post-processing)
	//
	// PRIMARY USE CASE: resource cleanup
	//   file, err := os.Open("file.txt")
	//   if err != nil { return err }
	//   defer file.Close()  ← runs when function returns, guaranteed
	//   // ... work with file ...
	//
	// This pattern is MUCH cleaner than try/finally and ensures you never
	// forget to close/unlock/clean up even if you add early returns.

	demonstrateDefer()

	// ─────────────────────────────────────────────────────────────────────
	// DEFER LIFO ORDER
	// ─────────────────────────────────────────────────────────────────────

	deferLIFO()

	// ─────────────────────────────────────────────────────────────────────
	// DEFER: ARGUMENT EVALUATION IS IMMEDIATE
	// ─────────────────────────────────────────────────────────────────────

	deferArgEvaluation()

	// ─────────────────────────────────────────────────────────────────────
	// PANIC — Runtime emergency: unrecoverable situation
	// ─────────────────────────────────────────────────────────────────────
	//
	// panic() stops normal execution of the current goroutine:
	//   1. Stops execution at the panic call
	//   2. Runs deferred functions in LIFO order (going up the call stack)
	//   3. If no recover() catches it: prints stack trace and exits the program
	//
	// WHEN TO USE panic:
	//   - Programmer errors: nil pointer you didn't expect, out-of-bounds,
	//     type assertion on wrong type (with single-value form)
	//   - Invariant violations: "this should never happen" situations
	//   - Initialization failures: if your program can't start correctly
	//     (like can't parse required config, can't connect to DB at boot)
	//
	// WHEN NOT TO USE panic:
	//   - Normal error conditions (file not found, invalid input, network error)
	//   - Anything the caller might want to handle
	//   - In library code (almost never — libraries should return errors)
	//
	// PANIC vs RETURN ERROR:
	//   Go's idiom is to RETURN errors, not panic.
	//   panic is for exceptional/unexpected situations.
	//   If in doubt, return an error.

	fmt.Println("\n── panic and recover ──")
	demonstratePanicRecover()

	// ─────────────────────────────────────────────────────────────────────
	// RECOVER — Catch a panic (only works inside a deferred function)
	// ─────────────────────────────────────────────────────────────────────
	//
	// recover() stops the panic and returns the value passed to panic().
	// It ONLY works when called DIRECTLY inside a deferred function.
	// Calling recover() outside a deferred function returns nil.
	//
	// PATTERN: wrap risky code in a function, defer a recovery function.
	//
	// USE CASES:
	//   1. Server: one goroutine panicking shouldn't crash the whole server.
	//      Recover in a middleware, log the panic, return 500.
	//   2. Plugin/untrusted code: if calling external code that might panic.
	//   3. Converting panic to error: library code that might panic internally.

	demonstrateSafeDivide()

	// ─────────────────────────────────────────────────────────────────────
	// DEFERRED FUNCTION MODIFYING NAMED RETURN VALUE
	// ─────────────────────────────────────────────────────────────────────
	//
	// This is advanced and subtle. If a function uses NAMED return values,
	// a deferred function can read AND MODIFY them.
	// This is used to implement "cleanup that depends on whether there was an error".

	fmt.Println("\n── defer modifying named return values ──")
	result, err := riskyOperation(false)
	fmt.Printf("  riskyOperation(false): result=%d, err=%v\n", result, err)
	result, err = riskyOperation(true)
	fmt.Printf("  riskyOperation(true):  result=%d, err=%v\n", result, err)

	fmt.Println("\n── defer in a loop (common mistake!) ──")
	deferInLoop()

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  defer: run at function return, LIFO, args evaluated immediately")
	fmt.Println("  Main use: resource cleanup (Close, Unlock, conn.Done)")
	fmt.Println("  panic: for unrecoverable programmer errors or invariant violations")
	fmt.Println("  DO NOT panic for normal errors — return error values")
	fmt.Println("  recover: catch panic ONLY inside a deferred function")
	fmt.Println("  recover use case: servers (don't crash on one goroutine panic)")
}

func demonstrateDefer() {
	fmt.Println("\n── defer: basic usage ──")
	fmt.Println("  defer guarantees cleanup regardless of how we exit:")

	simulateFileOperation()
}

func simulateFileOperation() {
	fmt.Println("  Opening file...")
	// defer registered immediately when we "open" the resource
	defer fmt.Println("  File closed (deferred cleanup ran)")

	fmt.Println("  Reading from file...")
	fmt.Println("  Processing data...")
	fmt.Println("  Function about to return...")
	// When this function returns, the deferred call runs
}

func deferLIFO() {
	fmt.Println("\n── defer: LIFO order (last deferred = first to run) ──")
	defer fmt.Println("  defer 1: First deferred, runs LAST")
	defer fmt.Println("  defer 2: Second deferred, runs SECOND")
	defer fmt.Println("  defer 3: Third deferred, runs FIRST")
	fmt.Println("  Function body executing...")
}

func deferArgEvaluation() {
	fmt.Println("\n── defer: arguments evaluated IMMEDIATELY ──")
	i := 1
	// The argument 'i' is evaluated NOW (i=1), not when defer runs
	defer fmt.Printf("  deferred i value = %d  (captured at defer time, not at return time)\n", i)
	i = 100  // this change does NOT affect the deferred call
	fmt.Printf("  current i = %d\n", i)
	// When function returns, deferred call prints i=1, not i=100
}

// demonstratePanicRecover shows panic + deferred cleanup + recovery
func demonstratePanicRecover() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  Recovered from panic: %v\n", r)
			fmt.Println("  Program continues after recover")
		}
	}()

	fmt.Println("  About to panic...")
	defer fmt.Println("  This deferred call DOES run (before recover)")
	panic("something went badly wrong!")
	// Code below panic is unreachable:
	// fmt.Println("this never runs")
}

// safeDivide uses recover to convert a panic into an error return
func safeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered panic: %v", r)
		}
	}()
	return a / b, nil  // panics if b == 0 (integer division by zero)
}

func demonstrateSafeDivide() {
	fmt.Println("\n── recover: convert panic to error ──")

	result, err := safeDivide(10, 2)
	fmt.Printf("  safeDivide(10, 2) = %d, err=%v\n", result, err)

	result, err = safeDivide(10, 0)
	fmt.Printf("  safeDivide(10, 0) = %d, err=%v  ← panic recovered!\n", result, err)
}

// riskyOperation demonstrates defer modifying named return values
func riskyOperation(fail bool) (result int, err error) {
	// This defer can SEE and MODIFY 'result' and 'err' because they're named.
	defer func() {
		if err != nil {
			// If there was an error, also zero out result for cleanliness
			result = 0
			fmt.Println("  defer: error detected, cleaning up result")
		}
	}()

	if fail {
		result = 42  // set before failing
		err = fmt.Errorf("operation failed intentionally")
		return
	}
	result = 100
	return
}

// deferInLoop demonstrates a common mistake: defer inside a loop
func deferInLoop() {
	// MISTAKE: defer inside a loop defers ALL calls until the FUNCTION returns
	// They don't run at the end of each loop iteration!
	// For 1000 iterations, you'd have 1000 pending defers — memory issue.
	//
	// BAD pattern (if this were real resource cleanup):
	// for _, f := range files {
	//     file, _ := os.Open(f)
	//     defer file.Close()  // all Close() calls pile up until function returns!
	// }
	//
	// GOOD pattern: wrap the body in a function:
	// for _, f := range files {
	//     func() {
	//         file, _ := os.Open(f)
	//         defer file.Close()  // runs at end of this anonymous func, not outer func
	//         // ... use file ...
	//     }()
	// }

	fmt.Println("  Correct pattern: defer inside anonymous func in loop")
	for i := 0; i < 3; i++ {
		func() {
			defer fmt.Printf("  cleanup for iteration %d\n", i)
			fmt.Printf("  working on iteration %d\n", i)
		}()  // called immediately
	}
}
