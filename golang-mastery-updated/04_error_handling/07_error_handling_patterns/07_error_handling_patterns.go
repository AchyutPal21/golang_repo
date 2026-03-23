// 07_error_handling_patterns.go
//
// REAL-WORLD ERROR HANDLING PATTERNS
// ====================================
// Writing correct error handling is easy. Writing CLEAN, non-repetitive,
// readable error handling is an art. This file covers patterns that senior
// Go engineers use to keep error-heavy code from becoming a mess.
//
// ROB PIKE — "ERRORS ARE VALUES"
// --------------------------------
// In his 2015 blog post "Errors are values", Rob Pike argues that because
// errors are ordinary values, you can PROGRAM with them. You are not
// limited to the if-err-nil check. You can use all of Go's abstraction
// tools — structs, methods, functions — to manage errors elegantly.
//
// The key insight: repetitive error checking is a code smell. If you find
// yourself copy-pasting the same error check pattern, factor it out.
//
// PATTERNS COVERED
// ----------------
// 1. errWriter — the canonical "errors are values" technique
// 2. Functional error accumulator
// 3. Sentinel at function level (defer + named return)
// 4. Error wrapping helper (reducing boilerplate)
// 5. "Errors in the middle" — deferred error handling
// 6. Operation result type (error in a wrapper)
// 7. Error annotation table (adding context to a set of calls)

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 1: errWriter — "errors are values" applied to sequential writes
// ─────────────────────────────────────────────────────────────────────────────
// Rob Pike's original example from the bufio.Writer pattern.
//
// PROBLEM: When writing multiple fields, the naive approach requires an
// if err != nil check after every Write call — extremely repetitive.
//
// SOLUTION: Embed the error in a struct. The struct's Write method checks
// the embedded error first; if it is non-nil, it skips the actual work.
// This means you can chain many Write calls and check the error once at the end.
//
// This pattern works BECAUSE errors are values — you can store them.

// errWriter wraps a strings.Builder and captures the first error.
// After the first failure, all subsequent writes are no-ops.
type errWriter struct {
	b   strings.Builder
	err error
}

// write appends s only if no previous error has occurred.
func (ew *errWriter) write(s string) {
	if ew.err != nil {
		return // silently skip — first error already captured
	}
	if s == "FAIL" {
		ew.err = errors.New("write: simulated write failure")
		return
	}
	ew.b.WriteString(s)
}

// result returns the accumulated string and any error.
func (ew *errWriter) result() (string, error) {
	return ew.b.String(), ew.err
}

func demonstrateErrWriter() {
	fmt.Println("── Pattern 1: errWriter ──")

	// Naive approach (commented out to avoid repetition, shown for contrast):
	// _, err := w.Write(field1)
	// if err != nil { return err }
	// _, err = w.Write(field2)
	// if err != nil { return err }
	// ... 10 more times ...

	// errWriter approach: no error check needed between writes.
	w := &errWriter{}
	w.write("Name: Alice\n")
	w.write("Age: 30\n")
	w.write("Email: alice@example.com\n")
	// All checks happen at the end:
	output, err := w.result()
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  output:\n%s", output)
	}

	// Same pattern but with a failure in the middle:
	w2 := &errWriter{}
	w2.write("Name: Bob\n")
	w2.write("FAIL") // triggers the error
	w2.write("Email: bob@example.com\n") // skipped
	_, err2 := w2.result()
	fmt.Println("  w2 error:", err2)
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 2: Functional error accumulator
// ─────────────────────────────────────────────────────────────────────────────
// When you have a sequence of functions each returning error, you can
// use a helper that skips subsequent calls once an error occurs.
// This is the functional equivalent of errWriter for non-method calls.

// run executes fns in sequence, stopping at the first error.
// It returns the index of the failing function for debugging.
func run(fns ...func() error) (int, error) {
	for i, fn := range fns {
		if err := fn(); err != nil {
			return i, fmt.Errorf("step %d: %w", i, err)
		}
	}
	return -1, nil
}

func demonstrateFunctionalAccumulator() {
	fmt.Println("── Pattern 2: Functional accumulator (run) ──")

	stepA := func() error { fmt.Println("  stepA: ok"); return nil }
	stepB := func() error { fmt.Println("  stepB: ok"); return nil }
	stepC := func() error { return errors.New("stepC: something broke") }
	stepD := func() error { fmt.Println("  stepD: ok"); return nil }

	idx, err := run(stepA, stepB, stepC, stepD)
	if err != nil {
		fmt.Printf("  failed at step %d: %v\n", idx, err)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 3: Named return + defer for consistent error annotation
// ─────────────────────────────────────────────────────────────────────────────
// Named return values let a deferred function access and MODIFY the return
// value. This allows you to add a function-name prefix to any error that
// exits the function — without repeating the prefix at every return site.
//
// WARNING: Named returns can make control flow confusing if overused.
// Reserve this pattern specifically for error annotation in medium-to-large
// functions with multiple return paths.

func loadConfig(path string) (cfg string, err error) {
	// This deferred function runs when loadConfig returns (for any reason).
	// If err is non-nil, it wraps it with the function name.
	// Because err is a named return, the deferred function can read and
	// write it directly — like a closure over the return variable.
	defer func() {
		if err != nil {
			err = fmt.Errorf("loadConfig %q: %w", path, err)
		}
	}()

	if path == "" {
		return "", errors.New("path must not be empty") // annotated by defer
	}
	if path == "/missing" {
		return "", errors.New("file not found") // annotated by defer
	}
	return "db_host=localhost\nport=5432", nil // defer does nothing
}

func demonstrateNamedReturn() {
	fmt.Println("── Pattern 3: Named return + defer annotation ──")

	for _, p := range []string{"", "/missing", "/valid/config.toml"} {
		cfg, err := loadConfig(p)
		if err != nil {
			fmt.Printf("  loadConfig(%q): %v\n", p, err)
		} else {
			fmt.Printf("  loadConfig(%q): ok, %d bytes\n", p, len(cfg))
		}
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 4: Wrap helper — reducing wrapping boilerplate
// ─────────────────────────────────────────────────────────────────────────────
// If you wrap errors with the same format in many places, factor the
// wrapping logic into a helper.

// wrapf wraps err with a formatted message, but returns nil if err is nil.
// This lets you write: return wrapf(err, "doThing %s", id)
// without the surrounding if err != nil.
func wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

func doStep(name string, fail bool) error {
	if fail {
		return errors.New("underlying failure")
	}
	return nil
}

func demonstrateWrapHelper() {
	fmt.Println("── Pattern 4: wrapf helper ──")

	err := wrapf(doStep("fetch", true), "fetchUser id=%d", 42)
	fmt.Println("  wrapf result:", err)

	nilErr := wrapf(doStep("fetch", false), "fetchUser id=%d", 42)
	fmt.Println("  wrapf nil:  ", nilErr)
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 5: Errors in the middle — deferred cleanup with error aggregation
// ─────────────────────────────────────────────────────────────────────────────
// When performing an operation that opens a resource and must close it,
// you often want to return BOTH the operation error AND the close error.
// A naive defer close() silently drops the close error.
//
// Pattern: capture the close error in a deferred function and combine
// it with the operation error using errors.Join.

// openResource simulates opening a resource (e.g., a file or DB connection).
type resource struct {
	name string
}

func (r *resource) use(fail bool) error {
	if fail {
		return fmt.Errorf("use %s: simulated failure", r.name)
	}
	return nil
}

func (r *resource) close(fail bool) error {
	if fail {
		return fmt.Errorf("close %s: flush failed", r.name)
	}
	return nil
}

// processResource correctly handles BOTH the use error and the close error.
func processResource(useFail, closeFail bool) (err error) {
	r := &resource{name: "db-connection"}

	defer func() {
		closeErr := r.close(closeFail)
		// Combine: if both fail, caller sees both errors.
		// If only close fails, caller sees that too.
		err = errors.Join(err, closeErr)
	}()

	return r.use(useFail)
}

func demonstrateErrorsInMiddle() {
	fmt.Println("── Pattern 5: Errors in the middle (use + close) ──")

	cases := []struct{ useFail, closeFail bool }{
		{false, false},
		{true, false},
		{false, true},
		{true, true},
	}
	for _, c := range cases {
		err := processResource(c.useFail, c.closeFail)
		fmt.Printf("  use_fail=%v close_fail=%v → err=%v\n", c.useFail, c.closeFail, err)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 6: Operation result type
// ─────────────────────────────────────────────────────────────────────────────
// For fan-out patterns (multiple goroutines doing the same kind of work),
// bundle the result and error together in a struct. This is cleaner than
// sending them on separate channels.

type OpResult struct {
	Input  string
	Output string
	Err    error
}

func (r OpResult) OK() bool   { return r.Err == nil }
func (r OpResult) Failed() bool { return r.Err != nil }

func processItem(item string) OpResult {
	if strings.HasPrefix(item, "bad_") {
		return OpResult{Input: item, Err: fmt.Errorf("processItem %q: rejected", item)}
	}
	return OpResult{Input: item, Output: strings.ToUpper(item)}
}

func demonstrateOpResult() {
	fmt.Println("── Pattern 6: OpResult struct ──")

	items := []string{"hello", "bad_data", "world", "bad_input", "foo"}
	results := make([]OpResult, len(items))
	for i, item := range items {
		results[i] = processItem(item)
	}

	var okCount, failCount int
	for _, r := range results {
		if r.OK() {
			fmt.Printf("  [OK]   %q → %q\n", r.Input, r.Output)
			okCount++
		} else {
			fmt.Printf("  [FAIL] %q → %v\n", r.Input, r.Err)
			failCount++
		}
	}
	fmt.Printf("  Summary: %d ok, %d failed\n", okCount, failCount)
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 7: Reducing repetition with a step runner that annotates errors
// ─────────────────────────────────────────────────────────────────────────────
// A named-step runner: each step has a name, and the runner wraps the
// error with the step name automatically. Clean and eliminates repeated
// fmt.Errorf("stepName: %w", err) at every call site.

type Step struct {
	Name string
	Run  func() error
}

func runSteps(steps []Step) error {
	for _, s := range steps {
		if err := s.Run(); err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
	}
	return nil
}

func demonstrateStepRunner() {
	fmt.Println("── Pattern 7: Named step runner ──")

	db := ""
	steps := []Step{
		{
			Name: "connect to database",
			Run: func() error {
				db = "connected"
				return nil
			},
		},
		{
			Name: "run migrations",
			Run: func() error {
				if db == "" {
					return errors.New("no connection")
				}
				return nil
			},
		},
		{
			Name: "load seed data",
			Run: func() error {
				return errors.New("seed file missing") // simulated failure
			},
		},
		{
			Name: "start HTTP server",
			Run: func() error {
				fmt.Println("  (this step never runs)")
				return nil
			},
		},
	}

	if err := runSteps(steps); err != nil {
		fmt.Printf("  startup failed: %v\n", err)
		fmt.Printf("  root cause: %v\n", errors.Unwrap(err))
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 07: Error Handling Patterns ===")
	fmt.Println()

	demonstrateErrWriter()
	demonstrateFunctionalAccumulator()
	demonstrateNamedReturn()
	demonstrateWrapHelper()
	demonstrateErrorsInMiddle()
	demonstrateOpResult()
	demonstrateStepRunner()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. errWriter: embed error in a struct → check once at the end")
	fmt.Println("  2. run() helper: chain functions, stop at first error")
	fmt.Println("  3. Named return + defer: annotate ALL exits with function name once")
	fmt.Println("  4. wrapf: skip wrapping when err is nil, reducing boilerplate")
	fmt.Println("  5. Errors in the middle: errors.Join(opErr, closeErr) in defer")
	fmt.Println("  6. OpResult: bundle result + error for fan-out / batch patterns")
	fmt.Println("  7. Named step runner: automatic error annotation without repetition")
}
