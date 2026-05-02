# Chapter 45 — Revision Checkpoint

## Questions

1. What is the difference between `sync.Mutex` and `sync.RWMutex` and when should you choose each?
2. Why must `sync.Cond.Wait` be called inside a `for` loop rather than an `if` statement?
3. What is `sync.Once` and what guarantee does it provide? Can it be reset?
4. What is the risk of calling `wg.Add(1)` inside the launched goroutine rather than before launching it?
5. When is `sync.Pool` appropriate, and what must you always do with a value you get from a pool?

## Answers

1. `sync.Mutex` is an exclusive lock — exactly one goroutine can hold it at a time, for both reads and writes. `sync.RWMutex` allows multiple concurrent readers (`RLock`/`RUnlock`) or one exclusive writer (`Lock`/`Unlock`). Choose `Mutex` for write-heavy workloads or when the critical section is very short (the overhead of `RWMutex` is higher than `Mutex`). Choose `RWMutex` when reads are much more frequent than writes and reads take long enough that allowing them to proceed concurrently meaningfully reduces contention.

2. `sync.Cond.Wait` releases the mutex and suspends the goroutine, then reacquires the mutex before returning. When Wait returns, it does NOT guarantee the condition is true — another goroutine may have changed state between the Signal and the goroutine actually waking up ("spurious wakeup"), or multiple goroutines may have been woken by `Broadcast` but only one can act on the condition. The `for` loop re-checks the condition after every wakeup and goes back to waiting if it is still false. An `if` would proceed on the assumption the condition is true, which is incorrect.

3. `sync.Once` is a primitive that runs a provided function exactly once, regardless of how many goroutines call `Do` concurrently. The first goroutine to call `Do` runs the function; all subsequent callers block until the function returns, then proceed without running it again. It provides the guarantee: "this code runs at most once, and any goroutine that calls Do after the first invocation sees the result of that invocation." `Once` cannot be reset — its `done` flag is set permanently. To re-run initialisation, create a new `sync.Once`.

4. If `wg.Add(1)` is called inside the goroutine, there is a race between the `wg.Wait()` call in the parent and the `Add` in the goroutine. If `Wait` is called before the goroutine runs its `Add`, the WaitGroup's counter is still zero, `Wait` returns immediately (thinking all goroutines are done), and the parent proceeds before the goroutine has actually started or finished. The goroutine then later calls `Add(1)` on a WaitGroup that the parent has already stopped watching, and the subsequent `Done()` may bring the counter below zero and panic.

5. `sync.Pool` is appropriate for short-lived objects that are allocated frequently in a hot path and are expensive to allocate or that create GC pressure — for example, `bytes.Buffer`, JSON encoders/decoders, or scratch arrays used within a single request handler. The critical rule: **always reset the object before use**. A pooled object may have been previously used and contain stale data. Calling `Reset()` (or the equivalent for that type) before using the object ensures you start with a clean state. Forgetting to reset is a subtle bug where old data from a previous caller leaks into the current operation.
