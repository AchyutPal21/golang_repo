# Chapter 49 — Exercises

## 49.1 — CSV pipeline

Run [`exercises/01_csv_pipeline`](exercises/01_csv_pipeline/main.go).

Five-stage pipeline processing 12 simulated CSV rows: source → parse → validate → enrich (4 parallel workers, 10ms simulated tax-lookup API) → aggregate. Invalid rows (negative amount, missing category) are reported inline; the final summary shows totals and per-category revenue.

Try:
- Add a `timeout` field to the config and cancel the context mid-pipeline — observe how many rows complete.
- Add a sixth stage that writes enriched rows to a channel consumed by two sink goroutines (fan-out the sink).
- Run with `-race` flag: `go run -race ./exercises/01_csv_pipeline` — confirm no data races.

## 49.2 ★ — Merge with priority

Write a `mergePriority(ctx, high, low <-chan int) <-chan int` that always drains `high` before `low` when both are ready, using the nested-select priority pattern from Ch44.

Verify: generate 100 high-priority items and 100 low-priority items; confirm that all high-priority items appear before any low-priority items in the output (or at worst interleaved at the boundary when the high channel empties).

## 49.3 ★★ — Rate-limited pipeline stage

Write a `rateLimit[T any](ctx context.Context, in <-chan T, rps int) <-chan T` stage that passes items through at most `rps` items per second. The stage should not drop items — it must block upstream when the rate limit is reached.

Use a `time.Ticker` with period `1s/rps` to gate emission. Test by pumping 20 items through a 5-RPS limiter and verifying the total duration is ≥ 4 seconds.
