// 04_panic_recover.go
//
// TOPIC: panic, recover — Internals, When to Use, Server Middleware Pattern,
//        Converting Panics to Errors, and Goroutine Limitations
//
// PANIC:
//   A panic is an abnormal termination of a goroutine. It triggers stack
//   unwinding — Go unwinds the call stack of the panicking goroutine,
//   running deferred functions at each frame as it goes. If no deferred
//   function calls recover(), the goroutine terminates, and the runtime
//   prints the panic value + stack trace, then exits the program.
//
// RECOVER:
//   recover() stops the unwinding and returns the panic value. It ONLY works
//   inside a deferred function — calling recover() outside a defer does nothing
//   and returns nil.
//
// WHEN TO PANIC vs RETURN ERROR:
//   Return error when:
//     - The failure is expected and the caller should handle it
//       (file not found, invalid input, network timeout)
//     - The function is part of a library — libraries should never panic
//       on bad input from the caller; return an error instead
//   Panic when:
//     - The program has reached an impossible/unrecoverable state
//       (programmer error, e.g., out-of-bounds on an internal data structure)
//     - During initialization if a required resource is unavailable and the
//       program cannot meaningfully continue (e.g., must-parse template at startup)
//     - In tests: t.Fatal is preferred, but a panic is acceptable
//
// The Go standard library panics for true programming errors (e.g., nil pointer
// dereference, index out of bounds) but returns errors for expected failures.

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─── 1. PANIC BASICS AND STACK UNWINDING ─────────────────────────────────────
//
// When panic(value) is called:
//   1. The current function stops executing immediately.
//   2. Any deferred functions in the CURRENT frame run.
//   3. Control returns to the caller, which also panics.
//   4. This "unwinds" the stack, running defers at each frame.
//   5. Continues until the top of the goroutine's stack (program terminates).
//
// panic() accepts any value: string, error, int, struct, anything.
// The runtime's built-in panics (nil pointer, index out of bounds) pass a
// runtime.Error value (which implements the error interface).

func level3() {
	fmt.Println("  level3: about to panic")
	defer fmt.Println("  level3: deferred cleanup (runs during unwind)")
	panic("something went terribly wrong in level3")
}

func level2() {
	defer fmt.Println("  level2: deferred cleanup (runs during unwind)")
	fmt.Println("  level2: calling level3")
	level3()
	fmt.Println("  level2: this line is NEVER reached") // unreachable
}

func level1WithRecover() {
	defer func() {
		// recover() MUST be called inside a defer to work.
		// It returns the panic value and stops the unwinding.
		if r := recover(); r != nil {
			fmt.Printf("  level1: recovered panic: %v\n", r)
		}
	}()
	defer fmt.Println("  level1: regular defer (runs after recover defer)")
	fmt.Println("  level1: calling level2")
	level2()
	fmt.Println("  level1: this line is NEVER reached")
}

// ─── 2. WHEN TO PANIC — PROGRAMMING ERRORS ───────────────────────────────────
//
// Good use of panic: catching programmer errors that indicate a bug in the code,
// not a failure in the environment.

// mustPositive panics if n <= 0 — it's a programming contract violation.
// Functions named "must..." conventionally panic on invalid input.
// They're used in initialization code where you want to fail fast.
func mustPositive(n int) int {
	if n <= 0 {
		panic(fmt.Sprintf("mustPositive: received %d, must be > 0 (programming error)", n))
	}
	return n
}

// mustParseTemplate simulates the common init-time panic pattern.
// Real use: template.Must(template.ParseFiles("tmpl.html"))
func mustParseTemplate(name string) string {
	if name == "" {
		panic("mustParseTemplate: template name cannot be empty")
	}
	return "parsed:" + name
}

// ─── 3. RECOVER PATTERN FOR SERVERS (MIDDLEWARE) ─────────────────────────────
//
// The canonical use case for recover() in production Go code is HTTP middleware.
// If a handler panics, you don't want the entire server to crash — you want to:
//   1. Recover the panic
//   2. Log the stack trace
//   3. Return a 500 response to the client
//   4. Continue serving other requests
//
// This is how packages like net/http, Gin, and Echo implement panic recovery.

// simulatedHandler represents a request handler that might panic.
type handlerFunc func(request string) string

// recoveryMiddleware wraps a handler and recovers from any panic.
// It converts the panic into an error response instead of crashing.
func recoveryMiddleware(handler handlerFunc) handlerFunc {
	return func(request string) (response string) {
		defer func() {
			if r := recover(); r != nil {
				// In real code: log.Printf("panic recovered: %v\n%s", r, debug.Stack())
				fmt.Printf("  [middleware] recovered panic: %v\n", r)
				// Modify the named return value to send an error response.
				response = fmt.Sprintf("500 Internal Server Error (panic: %v)", r)
			}
		}()
		return handler(request)
	}
}

func goodHandler(req string) string {
	return "200 OK: processed " + req
}

func panickyHandler(req string) string {
	if req == "bad" {
		panic("handler: cannot process 'bad' request")
	}
	return "200 OK: processed " + req
}

// ─── 4. CONVERTING PANIC TO ERROR ─────────────────────────────────────────────
//
// Sometimes you call a third-party function that might panic, and you want to
// present that as an error to YOUR callers (because your API should not panic).
// The pattern is: wrap in a function that defers a recover and assigns to a
// named error return.

// safeDiv wraps integer division and converts any divide-by-zero panic to an error.
func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			// r might be a runtime.Error or anything else.
			// We convert it to a proper error.
			err = fmt.Errorf("safeDiv: panic recovered: %v", r)
		}
	}()
	result = a / b // integer divide by zero causes a runtime panic
	return
}

// panicToError is a generic helper that runs fn and converts any panic to an error.
// Useful when integrating with panic-happy third-party code.
func panicToError(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v // if panic value is already an error, use it directly
			default:
				err = fmt.Errorf("panic: %v", v)
			}
		}
	}()
	fn()
	return nil
}

// ─── 5. PANIC WITH NON-ERROR TYPES ────────────────────────────────────────────
//
// panic() accepts any value — not just errors or strings.
// The recover() call returns interface{} (any), so you must type-assert to use it.
// This is unusual but occasionally seen in internal signaling patterns.

type panicPayload struct {
	Code    int
	Message string
}

func riskyOperation(fail bool) {
	if fail {
		// Panicking with a custom struct
		panic(panicPayload{Code: 42, Message: "custom panic payload"})
	}
	fmt.Println("  riskyOperation: success")
}

func handleStructPanic(fail bool) {
	defer func() {
		if r := recover(); r != nil {
			// Type switch to handle different panic value types
			switch p := r.(type) {
			case panicPayload:
				fmt.Printf("  recovered panicPayload: code=%d msg=%q\n", p.Code, p.Message)
			case error:
				fmt.Printf("  recovered error: %v\n", p)
			case string:
				fmt.Printf("  recovered string: %q\n", p)
			default:
				fmt.Printf("  recovered unknown type %T: %v\n", p, p)
			}
		}
	}()
	riskyOperation(fail)
}

// ─── 6. RE-PANICKING ──────────────────────────────────────────────────────────
//
// Sometimes you recover a panic, do some cleanup, and then re-panic because
// you can't handle it — you just wanted to run the cleanup.
// This preserves the original panic and lets it propagate up the stack.

func rePanicExample() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("  rePanicExample: doing cleanup before re-panic")
			// Re-panic with the SAME value to preserve original context.
			// If you panic(errors.New("new error")) you lose original info.
			panic(r)
		}
	}()
	panic("original panic message")
}

// ─── 7. PANICS IN GOROUTINES — CANNOT BE RECOVERED FROM A DIFFERENT GOROUTINE ──
//
// This is the most important limitation of panic/recover.
//
// If goroutine A panics, ONLY deferred functions in goroutine A can recover it.
// Goroutine B CANNOT recover goroutine A's panic.
// An unrecovered panic in ANY goroutine terminates the ENTIRE PROGRAM.
//
// This means:
//   - Every goroutine that might panic MUST have its own recover() defer.
//   - A common pattern: wrap all goroutine bodies in a recovery function.
//   - Worker pools often wrap each worker in a recover to prevent one bad
//     job from killing the entire pool.

func goroutineSafeWrapper(fn func()) {
	// Every goroutine you launch should be wrapped this way if they might panic.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  [goroutine wrapper] recovered: %v\n", r)
		}
	}()
	fn()
}

func demonstrateGoroutinePanic() {
	done := make(chan struct{})

	go func() {
		defer close(done)
		goroutineSafeWrapper(func() {
			fmt.Println("  goroutine: doing work")
			panic("goroutine panicked!")
		})
		fmt.Println("  goroutine: recovered and continuing")
	}()

	<-done
	fmt.Println("  main: program still alive after goroutine panic")
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  PANIC AND RECOVER")
	fmt.Println(sep)

	// 1. Stack unwinding with recover
	fmt.Println("\n── 1. Stack Unwinding + recover() ──")
	fmt.Println("  calling level1WithRecover:")
	level1WithRecover()
	fmt.Println("  program continues after level1WithRecover returns")

	// 2. must... pattern
	fmt.Println("\n── 2. must... Pattern (Programming Errors) ──")
	fmt.Println("  mustPositive(5) =", mustPositive(5))
	err := panicToError(func() { mustPositive(-1) })
	fmt.Println("  mustPositive(-1) converted to error:", err)

	tmpl := mustParseTemplate("index.html")
	fmt.Println("  mustParseTemplate:", tmpl)

	// 3. Server middleware recovery
	fmt.Println("\n── 3. Server Middleware Recovery ──")
	safeGoodHandler := recoveryMiddleware(goodHandler)
	safePanickyHandler := recoveryMiddleware(panickyHandler)

	fmt.Println(" ", safeGoodHandler("hello"))
	fmt.Println(" ", safePanickyHandler("hello"))
	fmt.Println(" ", safePanickyHandler("bad")) // this will panic and be recovered

	// 4. Converting panic to error
	fmt.Println("\n── 4. Converting Panic to Error ──")
	result, err := safeDiv(10, 2)
	fmt.Printf("  safeDiv(10, 2): result=%d, err=%v\n", result, err)

	result, err = safeDiv(10, 0)
	fmt.Printf("  safeDiv(10, 0): result=%d, err=%v\n", result, err)

	err = panicToError(func() { panic(errors.New("wrapped error panic")) })
	fmt.Println("  panicToError with error:", err)

	err = panicToError(func() { panic(123) })
	fmt.Println("  panicToError with int:", err)

	// 5. Non-error panic types
	fmt.Println("\n── 5. Panic with Non-Error Types ──")
	handleStructPanic(false) // no panic
	handleStructPanic(true)  // panics with panicPayload struct

	// 6. Re-panicking
	fmt.Println("\n── 6. Re-panicking ──")
	err = panicToError(func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("  inner recover: doing cleanup, then re-panicking")
				panic(r) // re-panic
			}
		}()
		panic("re-panic test")
	})
	fmt.Println("  outermost recovered:", err)

	// 7. Goroutine panic isolation
	fmt.Println("\n── 7. Panic Isolation in Goroutines ──")
	demonstrateGoroutinePanic()

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • panic unwinds the goroutine stack, running defers at each frame")
	fmt.Println("  • recover() only works inside a defer — outside it returns nil")
	fmt.Println("  • Use error returns for expected failures; panic for programmer bugs")
	fmt.Println("  • Libraries should never panic on bad user input — return error")
	fmt.Println("  • Server middleware: wrap handlers in recover defer to survive panics")
	fmt.Println("  • Convert panic→error: defer+recover+named return pattern")
	fmt.Println("  • panic values can be any type; type-switch in recover")
	fmt.Println("  • Goroutine A CANNOT recover goroutine B's panic — each needs its own")
	fmt.Println(sep)
}
