# Chapter 46 — Revision Checkpoint

## Questions

1. What is the difference between `atomic.Int64.Add` and a plain `n++` under concurrent access?
2. Describe the CAS (CompareAndSwap) semantic and write a CAS-loop pattern for a lock-free increment.
3. What rule must all `Store` calls on an `atomic.Value` obey, and what happens if it is violated?
4. When should you choose `atomic.Value` over `sync.RWMutex` for a shared read-heavy value?
5. Why can you not use atomic operations to protect an invariant that spans two separate variables?

## Answers

1. `atomic.Int64.Add` compiles to a single hardware atomic instruction (e.g., `LOCK XADD` on x86) that reads, modifies, and writes the memory location as one indivisible operation. No other goroutine can observe a partial state. A plain `n++` compiles to a read-modify-write sequence of separate instructions; the scheduler can switch goroutines between any two of them, causing lost updates (two goroutines both read the same old value, both add 1, both write the same new value — net effect: only one increment instead of two). Under concurrent access, `n++` is a data race; `atomic.Add` is correct.

2. CAS atomically checks whether a variable equals an expected value and, only if it does, sets it to a new value, returning whether the swap occurred. Semantic: `if *addr == old { *addr = new; return true } else { return false }`. CAS-loop for increment: `for { old := n.Load(); if n.CompareAndSwap(old, old+1) { break } }`. If another goroutine changed `n` between `Load` and `CompareAndSwap`, the CAS fails and the loop retries with the freshly-read value. This is correct because the loop eventually succeeds when no contention occurs on a given iteration.

3. All `Store` calls on an `atomic.Value` must store a value of exactly the same concrete type. The first `Store` call sets the type; subsequent calls that use a different type cause a panic at runtime. This constraint exists because `atomic.Value` uses a single pointer under the hood and stores type information alongside the data — mixing types would corrupt the type metadata and make `Load` return an incorrect type. If you need to store values of different types, use `atomic.Pointer[interface{}]` with an interface type.

4. Prefer `atomic.Value` over `sync.RWMutex` when: the shared value is replaced wholesale (not modified in place), the value is read on every request (very high read frequency), and the value fits in a pointer-sized store. `atomic.Value.Load` is a single atomic pointer read — no lock acquisition, no cache-line contention from the mutex's internal state, and no risk of priority inversion. `sync.RWMutex` is better when the critical section reads multiple fields that must be consistent together, or when the value cannot be treated as an immutable snapshot.

5. Atomic operations are atomic with respect to a single memory location. If an invariant requires two variables to be updated together (e.g., a balance and a transaction count that must agree), atomically updating each one separately does not prevent another goroutine from seeing the first update without the second. Between the two atomic writes, the system is in a temporarily inconsistent state. A mutex protects the entire critical section — the lock prevents any other goroutine from observing the state between the two writes. For multi-variable invariants, a mutex is required.
