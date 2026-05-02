# Chapter 44 — Exercises

## 44.1 — Retry with timeout

Run [`exercises/01_retry_with_timeout`](exercises/01_retry_with_timeout/main.go).

A `retryWithTimeout` function with per-attempt timeout, overall deadline, exponential back-off, and four test scenarios.

Try:
- Add `ErrPermanent` handling: if the operation returns `ErrPermanent`, stop retrying immediately and return the error without back-off.
- Add a `jitter float64` field to `RetryConfig` that randomises back-off by ±jitter fraction (e.g., 0.2 means ±20%). This avoids thundering herd when many callers retry simultaneously.
- Add a `OnAttempt func(attempt int, err error, backoff time.Duration)` callback so the caller can log each retry event.

## 44.2 ★ — Multiplexed fan-in with timeout

Implement `fanInWithTimeout[T any](timeout time.Duration, sources ...<-chan T) <-chan T` that:
- Merges all sources into a single output channel
- Closes the output channel when **all** sources are closed OR the timeout fires
- Uses a single goroutine per source plus one coordinator

Test with 3 sources that close at different times and a timeout that fires before the slowest source.

## 44.3 ★★ — Supervised ticker

Build a `SupervisedTicker` that:
- Calls a `work() error` function every `interval`
- If the work function doesn't return within `workTimeout`, logs a "slow work" warning and cancels the call (via a done channel passed to work)
- If work returns an error N consecutive times, calls an `onFatal(err)` callback and stops the ticker
- Exposes `Stop()` and `Stats() (calls, errors, timeouts int)`
