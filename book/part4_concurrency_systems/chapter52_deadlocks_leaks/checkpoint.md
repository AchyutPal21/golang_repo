# Chapter 52 — Revision Checkpoint

## Questions

1. How does the Go runtime detect deadlocks, and why can it only detect total deadlocks (all goroutines blocked) and not partial ones?
2. What is the lock-ordering rule, and why does it prevent circular waits?
3. What is the difference between a deadlock and a livelock, and which is harder to detect with tooling?
4. Give two concrete signs that a goroutine is leaking, visible at the OS level and at the Go runtime level.
5. Why must every blocking operation in a goroutine have a context-cancellation escape hatch?

## Answers

1. The Go runtime detects total deadlocks by checking whether **all goroutines** (every goroutine in the program) are blocked simultaneously — on a channel receive/send, a mutex, a `select` with no ready cases, etc. When this condition is true, no goroutine can ever become runnable again, so the runtime prints `fatal error: all goroutines are asleep - deadlock!` and exits. It cannot detect **partial** deadlocks (a subset of goroutines stuck) because the scheduler cannot tell whether those goroutines are permanently stuck or just waiting for an event that will eventually arrive. Partial deadlocks appear as goroutine leaks or infinite hangs rather than a runtime panic.

2. The lock-ordering rule states: **always acquire multiple mutexes in a globally consistent order** (e.g., by ascending ID, or by memory address). This prevents circular waits because if every goroutine follows the same ordering, no goroutine A can hold lock X while waiting for lock Y while goroutine B holds lock Y and waits for lock X — the lower-ordered lock must always be acquired first, so neither goroutine can be blocked waiting for a lock the other holds before it acquires the prerequisite lower-ordered lock.

3. A **deadlock** means goroutines are blocked — they are not consuming CPU. A **livelock** means goroutines are running (consuming CPU) but making no useful progress — they keep detecting conflict and backing off, then retrying, indefinitely. Deadlocks are easier to detect: the Go runtime detects total deadlocks and exits; partial deadlocks appear as a goroutine stuck in a block state in `pprof` goroutine dumps. Livelocks are harder to detect because the goroutines appear "active" in CPU profiles and only the absence of output or progress signals the problem.

4. OS-level signs: the process's RSS memory grows continuously over time (leaked goroutines hold stack memory); `go tool pprof` goroutine profile shows the count monotonically increasing. Go runtime level: `runtime.NumGoroutine()` returns a number that grows and never decreases after handling many requests; a goroutine dump (`SIGQUIT` or `kill -3`) shows goroutines blocked in `chan receive` or `chan send` at lines that should have returned long ago.

5. A goroutine that blocks without an escape hatch is a goroutine that can only exit when the blocking operation completes. If the caller times out, is cancelled, or the program shuts down, the goroutine continues to hold its stack, any captured heap allocations, and any file descriptors or locks it holds. Over the lifetime of a server that handles thousands of requests, leaked goroutines accumulate, exhaust memory, and eventually cause out-of-memory crashes. The context-cancellation escape hatch (`case <-ctx.Done(): return`) guarantees that no goroutine outlives the logical operation it was created to serve — it is the single most important pattern for safe concurrent programs in Go.
