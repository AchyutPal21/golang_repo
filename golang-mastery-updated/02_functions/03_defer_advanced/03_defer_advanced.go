// 03_defer_advanced.go
//
// TOPIC: defer — Real-World Patterns, Pitfalls, and Named Return Interactions
//
// defer schedules a function call to run when the SURROUNDING FUNCTION returns,
// regardless of whether the return is normal, via panic, or via os.Exit (well,
// os.Exit doesn't run defers — that's an important gotcha).
//
// HOW defer WORKS INTERNALLY:
//   Go maintains a per-goroutine defer stack (LIFO). Each defer statement pushes
//   a record onto this stack. When the function returns (any return path), the
//   runtime pops and executes each deferred call in reverse order.
//
//   ARGUMENT EVALUATION is EAGER: the arguments to the deferred function are
//   evaluated at the defer statement, not when the deferred function runs.
//   But if you defer an anonymous function with no args, it captures variables
//   by reference (like a closure) — so it sees the current values at execution time.
//
// WHY defer?
//   Before defer, cleanup code had to be duplicated at every return path:
//     if err != nil { mu.Unlock(); return err }
//     ...
//     mu.Unlock()   // forget this and you have a deadlock
//   defer centralizes cleanup — you write it once, right next to the resource
//   acquisition, and it runs no matter how the function exits.

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─── 1. THE CLASSIC PATTERNS ──────────────────────────────────────────────────

// 1a. Mutex unlock — the most common real-world defer usage.
// The defer goes RIGHT AFTER Lock() so you can never forget to Unlock().
// Without defer you'd need to call Unlock() at every return path.
type SafeCounter struct {
	mu    fakeRWMutex // using a fake mutex to keep this file standalone
	count int
}

type fakeRWMutex struct{ locked bool }

func (m *fakeRWMutex) Lock()   { m.locked = true }
func (m *fakeRWMutex) Unlock() { m.locked = false }

func (c *SafeCounter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock() // will ALWAYS run, even if a panic happens below
	// Now we can write complex logic with early returns and the mutex will
	// always be released. Without defer, every early return is a potential
	// deadlock bug.
	c.count++
}

// 1b. File close — defer ensures the file is closed even if processing errors.
// In real code: f, err := os.Open(...); defer f.Close()
func processFileMock(name string) error {
	fmt.Printf("  opening %q\n", name)
	// Imagine: f, err := os.Open(name)
	// defer f.Close()  ← placed immediately after the open+nil-check

	defer fmt.Printf("  closed %q (via defer)\n", name) // simulating f.Close()
	fmt.Printf("  processing %q\n", name)
	// If processing failed and returned early, the defer still fires.
	return nil
}

// 1c. HTTP body close — in real HTTP clients you always defer resp.Body.Close().
//
//	resp, err := http.Get(url)
//	if err != nil { return err }
//	defer resp.Body.Close()  ← right here, before reading
//	body, err := io.ReadAll(resp.Body)
//
// WHY immediately after the nil check?
//   Because if you defer before checking err, you might call Close() on a nil
//   resp.Body, which panics. Always: check err, THEN defer.
func httpBodyClosePattern() {
	fmt.Println("  [simulated] defer resp.Body.Close() pattern:")
	defer fmt.Println("  [simulated] Body closed")
	fmt.Println("  [simulated] Reading response body...")
	// In real code: io.ReadAll(resp.Body) would be here
}

// ─── 2. DEFER FOR TIMING / TRACING ────────────────────────────────────────────
//
// A common pattern is to defer the "end" of a measurement right after recording
// the start. The closure captures the start time by reference — but since start
// is set before the defer, and the defer's closure reads start when it RUNS
// (after the function completes), it sees the correct start time.

func timeTrack(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("  %s took %v\n", name, time.Since(start))
	}
}

// Usage: defer timeTrack("operationName")()
// Note the double parentheses:
//   timeTrack("foo")  → calls timeTrack, returns a func()
//   ()                → defers THAT returned func() (the stop part)
// The outer call (timeTrack) runs IMMEDIATELY (recording start time).
// The inner func() runs at the end of the surrounding function.
func expensiveOperation() {
	defer timeTrack("expensiveOperation")()
	// simulate work
	time.Sleep(1 * time.Millisecond)
	fmt.Println("  doing expensive work...")
}

// ─── 3. DEFER WITH NAMED RETURNS — CLEANUP THAT MODIFIES RETURN VALUES ────────
//
// This is an advanced but important pattern.
// When named return values are used, a deferred function can read AND MODIFY
// the values that will be returned to the caller.
//
// This enables: "begin a transaction, and if something goes wrong, roll it back
// inside the defer — all from a single code path."

type fakeDB struct{ log []string }

func (db *fakeDB) Begin() { db.log = append(db.log, "BEGIN") }
func (db *fakeDB) Commit() { db.log = append(db.log, "COMMIT") }
func (db *fakeDB) Rollback() { db.log = append(db.log, "ROLLBACK") }
func (db *fakeDB) Exec(sql string) error {
	db.log = append(db.log, "EXEC: "+sql)
	if strings.Contains(sql, "FAIL") {
		return fmt.Errorf("SQL error: %s", sql)
	}
	return nil
}

// withTransaction demonstrates: use named return 'err' so that the defer can
// inspect whether the transaction should be committed or rolled back.
func withTransaction(db *fakeDB, ops func(*fakeDB) error) (err error) {
	db.Begin()

	defer func() {
		// At this point 'err' holds whatever value the function is returning.
		// If err != nil, something went wrong → roll back.
		// If err == nil, everything succeeded → commit.
		if err != nil {
			db.Rollback()
			fmt.Println("  defer: rolled back transaction due to error:", err)
		} else {
			db.Commit()
			fmt.Println("  defer: committed transaction")
		}
	}()

	// Run the operations. If any fails, err is set and we return immediately.
	// The defer fires and sees the non-nil err → rollback.
	err = ops(db)
	return // naked return: returns current value of 'err'
}

// ─── 4. ARGUMENT EVALUATION IS EAGER ─────────────────────────────────────────
//
// CRITICAL subtlety: the ARGUMENTS of the deferred function are evaluated
// IMMEDIATELY when defer is executed, not when the deferred function runs.
//
// This trips up even experienced Go developers.

func deferArgEvaluation() {
	x := 1
	// CASE A: Argument evaluated NOW (x=1 is captured immediately).
	defer fmt.Println("  deferred with arg evaluated at defer time: x =", x)

	// CASE B: Anonymous func with no args captures x by REFERENCE.
	// It will print the value of x at the time it runs (after return).
	defer func() {
		fmt.Println("  deferred closure sees x at run time: x =", x)
	}()

	x = 100 // change x after the defers are registered
	fmt.Println("  deferArgEvaluation returning, x =", x)
	// Output order (defers run LIFO):
	//   1. "deferArgEvaluation returning, x = 100"  (normal return)
	//   2. Case B closure: x = 100  (captured by reference, sees 100)
	//   3. Case A fmt.Println: x = 1  (argument was evaluated when defer ran, x was 1 then)
}

// ─── 5. DEFER LOOP MISTAKE ────────────────────────────────────────────────────
//
// Never defer inside a loop if you mean "close each resource after each
// iteration". Defers only run when the FUNCTION returns, not each loop iteration.
// If you open 1000 files in a loop and defer Close() each, you hold 1000 open
// file descriptors until the function exits — a resource leak.
//
// The fix: wrap the loop body in an anonymous function and defer inside IT,
// so the defer fires when the anonymous function returns (each iteration).

func deferInLoopMistake() {
	items := []string{"a", "b", "c"}

	fmt.Println("  WRONG: defers run only when outer function returns:")
	// In real code this would open files; we simulate with prints.
	// All "closed X" messages appear AFTER the loop, not per-iteration.
	for _, item := range items {
		item := item // Go 1.22 range would handle this, but explicit for clarity
		defer fmt.Printf("    closed %q (end of function)\n", item)
	}

	fmt.Println("  (loop finished, now function returns, defers fire...)")
}

func deferInLoopFixed() {
	items := []string{"a", "b", "c"}

	fmt.Println("  CORRECT: wrap loop body in anonymous func:")
	for _, item := range items {
		func(name string) {
			// "open" the resource
			fmt.Printf("    opened %q\n", name)
			defer fmt.Printf("    closed %q (end of inner func)\n", name)
			// process the resource
			fmt.Printf("    processed %q\n", name)
		}(item) // immediately invoked — defer fires when this func returns
	}
}

// ─── 6. DEFER FOR TRANSACTION ROLLBACK — STACKING PATTERN ────────────────────
//
// Multiple defers stack (LIFO). This is useful for multi-step operations where
// each step has a corresponding cleanup and cleanups must run in reverse order
// (like releasing locks or closing nested resources).

type Step struct{ name string }

func (s Step) do() error {
	fmt.Printf("  executing step: %s\n", s.name)
	return nil
}
func (s Step) undo() {
	fmt.Printf("  undoing step: %s\n", s.name)
}

func multiStepOperation() (err error) {
	steps := []Step{
		{name: "allocate resource A"},
		{name: "allocate resource B"},
		{name: "allocate resource C"},
	}

	for _, step := range steps {
		s := step // capture
		if err = s.do(); err != nil {
			return err // defers for already-done steps will fire
		}
		// Each successful step registers its own undo.
		// If a later step fails, all registered undos run in LIFO order.
		defer s.undo()
	}

	fmt.Println("  all steps succeeded")
	return nil
	// Defers fire here in LIFO: undo C, undo B, undo A
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  DEFER — ADVANCED PATTERNS")
	fmt.Println(sep)

	// 1a. Mutex unlock
	fmt.Println("\n── 1a. Mutex Unlock via defer ──")
	c := &SafeCounter{}
	c.Increment(); c.Increment(); c.Increment()
	fmt.Printf("  counter = %d (mutex always released via defer)\n", c.count)

	// 1b. File close
	fmt.Println("\n── 1b. File Close via defer ──")
	_ = processFileMock("data.txt")

	// 1c. HTTP body close
	fmt.Println("\n── 1c. HTTP Body Close Pattern ──")
	httpBodyClosePattern()

	// 2. Timing/tracing
	fmt.Println("\n── 2. Timing with defer ──")
	expensiveOperation()

	// 3. Named returns + defer for transactions
	fmt.Println("\n── 3. Named Returns + defer (Transaction Rollback) ──")
	db := &fakeDB{}

	err := withTransaction(db, func(db *fakeDB) error {
		return db.Exec("INSERT INTO users VALUES (1, 'Alice')")
	})
	fmt.Println("  success transaction log:", db.log, "err:", err)

	db2 := &fakeDB{}
	err = withTransaction(db2, func(db *fakeDB) error {
		_ = db.Exec("INSERT INTO users VALUES (2, 'Bob')")
		return db.Exec("INSERT FAIL INTO broken") // this fails
	})
	fmt.Println("  failed transaction log:", db2.log, "err:", err)

	// 4. Argument evaluation timing
	fmt.Println("\n── 4. Eager Argument Evaluation ──")
	deferArgEvaluation()

	// 5. Defer in loop — the mistake
	fmt.Println("\n── 5a. Defer in Loop (WRONG) ──")
	deferInLoopMistake() // defers fire at the END, not per-iteration

	fmt.Println("\n── 5b. Defer in Loop (CORRECT — wrap in func) ──")
	deferInLoopFixed()

	// 6. Stacking defers for rollback
	fmt.Println("\n── 6. Stacking Defers (LIFO Rollback) ──")
	_ = multiStepOperation()

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • defer runs when the surrounding FUNCTION returns (not block)")
	fmt.Println("  • Arguments to deferred func are evaluated EAGERLY at defer time")
	fmt.Println("  • Closures in defer capture variables by reference (lazy evaluation)")
	fmt.Println("  • Named returns + defer: deferred func can modify return values")
	fmt.Println("  • Multiple defers = LIFO stack — great for ordered cleanup")
	fmt.Println("  • defer in loop = defers pile up; wrap body in func() to fix")
	fmt.Println("  • os.Exit() does NOT run defers — only log.Fatal triggers exit")
	fmt.Println(sep)
}
