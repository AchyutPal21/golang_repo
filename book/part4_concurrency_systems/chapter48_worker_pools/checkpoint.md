# Chapter 48 — Revision Checkpoint

## Questions

1. In the N-worker pool pattern, what are the two shutdown signals a worker must handle, and how do they differ?
2. Why must you close `results` in a separate goroutine rather than immediately after `wg.Wait()`?
3. What does `errgroup.WithContext` return, and what happens when the first goroutine returns a non-nil error?
4. How does a buffered channel implement a counting semaphore, and what prevents the semaphore count from exceeding the limit?
5. In the scatter-gather pattern, why is it safe for goroutines to write to distinct slice indices without a mutex?

## Answers

1. Workers must handle two shutdown signals: (a) **closed jobs channel** (`ok == false` from `case job, ok := <-jobs`) — the graceful path where all jobs drain normally before workers exit; (b) **context cancellation** (`case <-ctx.Done()`) — the immediate-stop path where the caller decides to abort before the job queue is empty. The difference is intent: channel close means "no more work," context cancellation means "stop regardless of remaining work."

2. `wg.Wait()` blocks until all workers exit. If you call it from the same goroutine that is draining `results`, you deadlock: workers cannot send to a full `results` channel, so they never finish, so `wg.Wait()` never returns. Calling `wg.Wait()` in a separate goroutine lets the sink loop keep consuming from `results`, freeing space for workers to complete. Only after all workers finish does the goroutine close `results`, which terminates the range loop.

3. `errgroup.WithContext(parent)` returns a `*Group` and a derived `context.Context`. The context is created with `context.WithCancelCause`. When any goroutine passed to `g.Go(fn)` returns a non-nil error, the group records the first error via `sync.Once` and calls its internal cancel function with that error. This cancels the derived context, signalling all other goroutines (which should observe `ctx.Done()`) to stop early. `g.Wait()` blocks until all goroutines return and then returns the first recorded error.

4. A buffered channel `sem := make(chan struct{}, N)` implements a counting semaphore because sending blocks when the channel is full. To acquire the semaphore, a goroutine sends `sem <- struct{}{}` — if N goroutines are already holding slots (channel is full), the send blocks. To release, the goroutine receives `<-sem`, freeing one slot. The channel capacity N is the maximum count: the runtime enforces it by blocking any additional sender until a receiver creates space.

5. Each goroutine is assigned a unique index `i` before launch (via the loop variable capture `i := i` or Go 1.22 per-iteration semantics). Each goroutine reads from and writes to only `results[i]` — a distinct memory location from every other goroutine. The Go memory model guarantees that two goroutines that never touch the same memory word do not require synchronization; writes to non-overlapping array elements are independent. `wg.Wait()` provides the happens-before relationship that ensures the parent goroutine sees all written values after the wait returns.
