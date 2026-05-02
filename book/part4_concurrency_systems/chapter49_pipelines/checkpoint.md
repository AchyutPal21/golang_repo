# Chapter 49 — Revision Checkpoint

## Questions

1. What is the canonical function signature for a pipeline stage, and what two things must it always do?
2. How does context cancellation propagate through a chain of pipeline stages without the stages knowing about each other?
3. What is back-pressure in the context of channels, and how does it prevent unbounded memory growth?
4. Why does fan-out (N workers draining one channel) not require a mutex to distribute work safely?
5. When you fan-out and fan-in through multiple workers, why does the output order differ from the input order, and how do you restore it?

## Answers

1. A pipeline stage takes `(ctx context.Context, in <-chan T)` and returns `<-chan U`. It must: (a) close its output channel when done — `defer close(out)` — so the downstream stage's range or `ok` check terminates; (b) respect both the upstream's closure (`!ok`) and context cancellation (`ctx.Done()`) so the pipeline tears down cleanly on either signal. Without `defer close(out)`, the next stage would block forever waiting for more items.

2. Each stage holds a reference to the same `ctx`. When the context is cancelled (by a timeout, manual cancel, or errgroup error), every stage's `case <-ctx.Done()` fires independently and returns, closing its output channel. The stage downstream sees the channel close (`!ok`) and also returns, propagating the shutdown in a cascade. Stages don't call each other's cancel functions — the shared context is the only coordination point.

3. Back-pressure occurs when the channel between two stages fills to capacity and the sender blocks. A buffered channel of size N means the producer can get at most N items ahead of the consumer; once the buffer is full, the producer goroutine parks on the send until the consumer reads. This bounds memory use to at most `cap(channel) * sizeof(item)` instead of growing without limit. It is the natural throttle built into the channel model — no explicit rate-limit code is needed to prevent a fast producer from overwhelming a slow consumer.

4. A channel is safe for concurrent use by multiple goroutines. When N workers all do `case job := <-jobs`, the Go runtime's channel implementation uses an internal mutex to ensure exactly one receiver wins each item — the same way only one send wins in a select race. No user-level mutex is needed because the channel itself serializes access. Each job is delivered to exactly one worker; workers never see the same item twice.

5. Workers have different execution speeds (due to OS scheduling, varying item cost, or simulated delays). Worker 3 might finish its item before worker 1 even starts, so results arrive in completion order, not dispatch order. To restore input order: (a) attach the original index to each item before dispatch: `type indexed[T] struct{ i int; v T }`; (b) write results into a pre-allocated slice at position `r.i`; (c) wait for all goroutines to finish before reading the slice. Because each goroutine writes to a unique index, no mutex is required and the final slice holds results in the original order.
