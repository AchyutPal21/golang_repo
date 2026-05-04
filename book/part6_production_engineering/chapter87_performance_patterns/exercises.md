# Chapter 87 Exercises — Performance Patterns

## Exercise 1 (provided): Slab Allocator

Location: `exercises/01_buffer_slab/main.go`

Implements a fixed-pool slab allocator with:
- 64 slots × 4096 bytes
- Channel-based lock-free acquire/release
- Slot zeroing on release
- Concurrent packet processor using the slab
- Allocation comparison vs per-call `make`

## Exercise 2 (self-directed): Ring-Buffer Logger

Build a zero-allocation ring-buffer logger that:
- Stores the last N log entries in a pre-allocated `[N][256]byte` array
- Uses `atomic.Int64` for the write cursor (no mutex)
- Writes log messages using `strconv.AppendInt` and direct byte copies
- Prints the ring buffer contents in order (newest-last)
- Measures: zero heap allocations after the initial setup

Acceptance criteria:
- `go test -race` passes
- Logging 10 000 entries allocates 0 bytes on the heap (after warm-up)

## Exercise 3 (self-directed): HTTP Response Formatter Pool

Build a `ResponseFormatter` that:
- Is pooled via `sync.Pool`
- Formats JSON responses into a caller-supplied `[]byte` sink
- Uses `strconv.AppendInt`, `strconv.AppendBool`, `strconv.AppendQuote`
- Never calls `fmt.Sprintf` or `json.Marshal`
- Benchmarks at least 5× fewer allocations than a `json.Marshal` approach

## Stretch Goal: Concurrent Slab with Waiting

Extend the slab from Exercise 1 to support a `AcquireWait(ctx context.Context)`
method that blocks until a slot is available or the context is cancelled,
instead of returning `ErrSlabFull` immediately. Implement a fairness queue so
waiting goroutines are served in FIFO order.
