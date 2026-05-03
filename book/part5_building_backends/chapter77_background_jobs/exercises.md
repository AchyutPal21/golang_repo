# Chapter 77 Exercises — Background Jobs and Schedulers

## Exercise 1 — Job Processor (`exercises/01_job_processor`)

Build an order processing system with typed jobs, a middleware chain, retry with DLQ, idempotency, and job observability.

### Job types

```go
type OrderPayload   struct { OrderID, CustomerID string; Total int }
type EmailPayload   struct { To, Subject, Body string }
type ShipmentPayload struct { OrderID, Address string }
```

### Processor interface

```go
func (p *Processor) Register(jobType string, h Handler, mws ...Middleware)
func (p *Processor) Enqueue(jobType string, maxRetries int, payload any) *Job
func (p *Processor) RunWorkers(ctx context.Context, n int) *sync.WaitGroup
func (p *Processor) DLQ() []*Job
```

### Middleware

**`LoggingMW`**: prints job ID, type, and attempt number before running; prints error after on failure.

**`TimeoutMW(d time.Duration)`**: wraps the handler with `context.WithTimeout`; handler returns error if it exceeds the deadline.

### Idempotency

Build an `IdempotencyStore` with:
```go
func (s *IdempotencyStore) Check(key string) bool  // true = already processed
```

Use it in the `process-order` handler with key `"process-order:" + orderID` to deduplicate duplicate enqueues.

### Demonstration

1. Register 4 handlers: `process-order`, `send-email`, `create-shipment`, `charge-card`
2. `send-email` fails transiently for first 2 attempts, succeeds on 3rd
3. `charge-card` always fails → DLQ after maxRetries=2
4. Enqueue 5 jobs including a duplicate `process-order` for the same order
5. Run 2 workers; wait for queue to drain
6. Print: enqueued, done, failed_attempts, DLQ count
7. Print DLQ entries with job ID, type, attempts, and error messages
8. Compute and print average job duration

### Hints

- `Job.StartedAt` and `Job.FinishedAt` allow per-job duration tracking; average = sum(durations) / done_count
- `Chain` applies middleware right-to-left: outermost middleware is first in the slice
- The processor does not need a separate goroutine — use `sync.Mutex` + a polling loop in each worker
- Worker polling loop: `if pending empty → sleep(2ms); else pop and run`
- Use `atomic.Int64` for all stats fields to avoid lock contention during tracking
