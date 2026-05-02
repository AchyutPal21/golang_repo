# Chapter 51 — Revision Checkpoint

## Questions

1. What is a data race, and why does it not always produce a crash or obviously wrong result?
2. How does the race detector work mechanically? What does it instrument, and what does it track?
3. Why does Go's `map` panic on concurrent writes even without `-race`, while a plain `int` with concurrent writes only silently loses updates?
4. What is the difference between `sync.Mutex` and `atomic.Int64` as a fix for a counter race, and when would you prefer each?
5. Why does `go test -race ./...` belong in CI even though the race detector only catches races that actually occur during the run?

## Answers

1. A data race occurs when two goroutines access the same memory location concurrently, and at least one of the accesses is a write, without any synchronisation between them. It does not always crash because modern CPUs and compilers reorder instructions and buffer writes to cache lines; the incorrect value may happen to be the right one on a given run, or the "wrong" value may still produce observable output that looks plausible. The bug manifests non-deterministically and only under specific scheduling conditions — making it very hard to reproduce without tooling.

2. The race detector is implemented by the `ThreadSanitizer` (TSan) library integrated into the Go toolchain. `go build -race` instruments every memory access at compile time, inserting calls to the shadow memory tracking library. At runtime, TSan maintains a **happens-before graph** using vector clocks: each goroutine carries a logical clock; every synchronisation event (channel send/receive, mutex lock/unlock, `sync.WaitGroup.Done/Wait`, atomic operations) advances the clocks and records an edge in the graph. When two accesses to the same memory address are observed with no happens-before relationship between them, TSan reports a race — printing the goroutine stacks, the conflicting accesses, and their creation sites.

3. Go's `map` implementation has a **write flag** (a single word) that is set whenever a mutation begins and checked at the start of every map operation. This check is in the runtime itself and uses a plain (non-atomic) read — it fires even without TSan instrumentation because the map intentionally detects concurrent mutations to provide a clear error rather than silent corruption. A plain `int` has no such guard: the processor silently overwrites the value in a non-atomic read-modify-write sequence, and the lost update is invisible unless you're comparing the final value to the expected total.

4. `atomic.Int64` is cheaper: a single CPU instruction (LOCK XADD or similar), no syscall, no contention structure. It is ideal when the protected data is a single numeric value and operations are Add/Load/Store/CAS. `sync.Mutex` is more expensive (involves OS futex under contention) but is appropriate when you need to protect a composite update (multiple fields that must change together atomically) or when you hold the lock across several statements. For a single counter: prefer `atomic.Int64`. For a struct with multiple fields that must be read/written together: use `sync.Mutex`.

5. Tests exercise more goroutines, more code paths, and more concurrent interactions than a typical `main()` run. Many races are latent — they only manifest when two goroutines are scheduled at exactly the right relative timing. Running with `-race` under tests dramatically increases the probability of triggering the race because (a) tests are structured to exercise boundary conditions, (b) goroutines are created at higher density, and (c) the race detector's shadow memory tracking observes all accesses regardless of timing. Even so, `-race` is a dynamic tool and will miss races that don't execute on a given run — but catching 80–90% of races in CI is far better than finding them in production.
