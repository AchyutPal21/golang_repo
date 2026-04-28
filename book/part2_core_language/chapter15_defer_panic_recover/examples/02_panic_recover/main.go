// FILE: book/part2_core_language/chapter15_defer_panic_recover/examples/02_panic_recover/main.go
// CHAPTER: 15 — defer, panic, recover
// TOPIC: panic mechanics, recover contract, safe wrapper, re-panic,
//        panic vs error return.
//
// Run (from the chapter folder):
//   go run ./examples/02_panic_recover

package main

import (
	"fmt"
	"runtime/debug"
)

// panicUnwinding shows the stack unwinding: deferred functions run
// during a panic before the program crashes.
func panicUnwinding() {
	defer fmt.Println("  [defer A] runs during panic unwind")
	defer fmt.Println("  [defer B] runs during panic unwind")
	fmt.Println("  about to panic")
	panic("something went wrong")
}

// safeDiv wraps division in a recover, converting a panic to an error.
// recover() only works inside a deferred function called during unwinding.
func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered panic: %v", r)
		}
	}()
	result = a / b // panics if b == 0
	return
}

// safeCall wraps any func() to catch panics, returning an error.
// This is the general-purpose boundary pattern.
func safeCall(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	f()
	return nil
}

// safeCallWithStack recovers and captures the stack trace.
func safeCallWithStack(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = fmt.Errorf("panic: %v\n%s", r, stack)
		}
	}()
	f()
	return nil
}

// mustGetKey panics with a descriptive message when an expected key
// is absent — useful in init/startup code where missing config is fatal.
func mustGetKey(m map[string]string, key string) string {
	v, ok := m[key]
	if !ok {
		panic(fmt.Sprintf("required key %q not found in config", key))
	}
	return v
}

// rePanic demonstrates re-panicking: recover, inspect, re-panic
// if not the expected type.
type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }

func handleRequest(fail bool) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		// Only recover our own error type; re-panic for unknown panics.
		if ae, ok := r.(*authError); ok {
			err = ae
			return
		}
		panic(r) // re-panic: not our problem
	}()

	if fail {
		panic(&authError{"token expired"})
	}
	return nil
}

func main() {
	// --- panic unwind (catch it with safeCall) ---
	fmt.Println("=== panic unwind ===")
	if err := safeCall(panicUnwinding); err != nil {
		fmt.Println("  caught:", err)
	}

	fmt.Println()

	// --- safeDiv ---
	fmt.Println("=== safeDiv ===")
	r, err := safeDiv(10, 2)
	fmt.Printf("10/2 = %d, err=%v\n", r, err)
	r, err = safeDiv(10, 0)
	fmt.Printf("10/0 = %d, err=%v\n", r, err)

	fmt.Println()

	// --- mustGetKey ---
	fmt.Println("=== mustGetKey ===")
	cfg := map[string]string{"host": "localhost"}
	fmt.Println("host:", mustGetKey(cfg, "host"))
	if err := safeCall(func() { mustGetKey(cfg, "port") }); err != nil {
		fmt.Println("missing key:", err)
	}

	fmt.Println()

	// --- rePanic / selective recovery ---
	fmt.Println("=== selective recovery ===")
	err = handleRequest(false)
	fmt.Println("no fail:", err)

	err = handleRequest(true)
	fmt.Println("auth fail:", err)

	// Unknown panic is re-panicked — catch it at the outer boundary.
	if err := safeCall(func() {
		defer func() {
			if r := recover(); r != nil {
				panic(r) // simulate re-panic
			}
		}()
		panic("mystery crash")
	}); err != nil {
		fmt.Println("unknown panic:", err)
	}
}
