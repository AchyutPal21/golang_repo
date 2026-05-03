// FILE: book/part5_building_backends/chapter77_background_jobs/exercises/01_job_processor/main.go
// CHAPTER: 77 — Background Jobs and Schedulers
// TOPIC: Order processing system — typed jobs, multi-stage pipelines,
//        retry with exponential backoff, job observability, and idempotency.
//
// Run (from the chapter folder):
//   go run ./exercises/01_job_processor

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// JOB TYPES
// ─────────────────────────────────────────────────────────────────────────────

type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusRunning JobStatus = "running"
	StatusDone    JobStatus = "done"
	StatusFailed  JobStatus = "failed"
	StatusDead    JobStatus = "dead"
)

type Job struct {
	ID         string
	Type       string
	Payload    any
	MaxRetries int
	Attempts   int
	Status     JobStatus
	Errors     []string
	EnqueuedAt time.Time
	StartedAt  time.Time
	FinishedAt time.Time
}

func (j *Job) Duration() time.Duration {
	if j.FinishedAt.IsZero() {
		return 0
	}
	return j.FinishedAt.Sub(j.StartedAt)
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB REGISTRY + HANDLER
// ─────────────────────────────────────────────────────────────────────────────

type Handler func(ctx context.Context, job *Job) error

type Middleware func(next Handler) Handler

func Chain(h Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func LoggingMW(next Handler) Handler {
	return func(ctx context.Context, job *Job) error {
		fmt.Printf("  [log] job %s type=%s attempt=%d\n", job.ID, job.Type, job.Attempts+1)
		err := next(ctx, job)
		if err != nil {
			fmt.Printf("  [log] job %s failed: %v\n", job.ID, err)
		}
		return err
	}
}

func TimeoutMW(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, job *Job) error {
			tCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(tCtx, job)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PROCESSOR
// ─────────────────────────────────────────────────────────────────────────────

type Processor struct {
	mu       sync.Mutex
	pending  []*Job
	dlq      []*Job
	handlers map[string]Handler
	seq      atomic.Int64
	Stats    struct {
		Enqueued  atomic.Int64
		Done      atomic.Int64
		Failed    atomic.Int64
		DLQed     atomic.Int64
		TotalTime atomic.Int64 // nanoseconds
	}
}

func NewProcessor() *Processor {
	return &Processor{handlers: make(map[string]Handler)}
}

func (p *Processor) Register(jobType string, h Handler, mws ...Middleware) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[jobType] = Chain(h, mws...)
}

func (p *Processor) Enqueue(jobType string, maxRetries int, payload any) *Job {
	p.mu.Lock()
	defer p.mu.Unlock()
	j := &Job{
		ID:         fmt.Sprintf("j-%d", p.seq.Add(1)),
		Type:       jobType,
		Payload:    payload,
		MaxRetries: maxRetries,
		Status:     StatusPending,
		EnqueuedAt: time.Now(),
	}
	p.pending = append(p.pending, j)
	p.Stats.Enqueued.Add(1)
	return j
}

func (p *Processor) pop() *Job {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) == 0 {
		return nil
	}
	j := p.pending[0]
	p.pending = p.pending[1:]
	j.Status = StatusRunning
	j.StartedAt = time.Now()
	return j
}

func (p *Processor) finish(j *Job, err error) {
	j.FinishedAt = time.Now()
	p.Stats.TotalTime.Add(int64(j.Duration()))
	if err == nil {
		j.Status = StatusDone
		p.Stats.Done.Add(1)
		return
	}
	j.Attempts++
	j.Errors = append(j.Errors, err.Error())
	p.Stats.Failed.Add(1)
	if j.Attempts >= j.MaxRetries {
		j.Status = StatusDead
		p.mu.Lock()
		p.dlq = append(p.dlq, j)
		p.mu.Unlock()
		p.Stats.DLQed.Add(1)
		return
	}
	j.Status = StatusPending
	p.mu.Lock()
	p.pending = append(p.pending, j)
	p.mu.Unlock()
}

func (p *Processor) RunWorkers(ctx context.Context, n int) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				j := p.pop()
				if j == nil {
					time.Sleep(2 * time.Millisecond)
					continue
				}
				h, ok := p.handlers[j.Type]
				if !ok {
					p.finish(j, fmt.Errorf("no handler"))
					continue
				}
				err := h(ctx, j)
				p.finish(j, err)
			}
		}(i)
	}
	return &wg
}

func (p *Processor) DLQ() []*Job {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*Job, len(p.dlq))
	copy(out, p.dlq)
	return out
}

func (p *Processor) Pending() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pending)
}

// ─────────────────────────────────────────────────────────────────────────────
// IDEMPOTENCY STORE
// ─────────────────────────────────────────────────────────────────────────────

type IdempotencyStore struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{seen: make(map[string]bool)}
}

func (s *IdempotencyStore) Check(key string) (alreadyProcessed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return true
	}
	s.seen[key] = true
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER PROCESSING DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type OrderPayload struct {
	OrderID    string
	CustomerID string
	Total      int
}

type EmailPayload struct {
	To      string
	Subject string
	Body    string
}

type ShipmentPayload struct {
	OrderID string
	Address string
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Job Processor Exercise ===")
	fmt.Println()

	p := NewProcessor()
	idempotency := NewIdempotencyStore()
	var failCount atomic.Int32

	// ── REGISTER HANDLERS ────────────────────────────────────────────────────
	p.Register("process-order", func(ctx context.Context, j *Job) error {
		op := j.Payload.(OrderPayload)
		key := "process-order:" + op.OrderID
		if idempotency.Check(key) {
			fmt.Printf("  [order] duplicate job for %s, skipping\n", op.OrderID)
			return nil
		}
		fmt.Printf("  [order] processing %s total=%d\n", op.OrderID, op.Total)
		return nil
	}, LoggingMW)

	p.Register("send-email", func(ctx context.Context, j *Job) error {
		ep := j.Payload.(EmailPayload)
		// Simulate 50% transient failure.
		if j.Attempts < 2 && failCount.Add(1) <= 2 {
			return fmt.Errorf("SMTP connection timeout")
		}
		fmt.Printf("  [email] sent to %s: %s\n", ep.To, ep.Subject)
		return nil
	}, LoggingMW, TimeoutMW(50*time.Millisecond))

	p.Register("create-shipment", func(ctx context.Context, j *Job) error {
		sp := j.Payload.(ShipmentPayload)
		fmt.Printf("  [shipment] created for order %s addr=%s\n", sp.OrderID, sp.Address)
		return nil
	}, LoggingMW)

	// Always-failing job to hit DLQ.
	p.Register("charge-card", func(ctx context.Context, j *Job) error {
		return fmt.Errorf("card declined")
	}, LoggingMW)

	// ── ENQUEUE JOBS ─────────────────────────────────────────────────────────
	fmt.Println("--- Enqueueing jobs ---")
	p.Enqueue("process-order", 3, OrderPayload{"ord-1", "c-1", 9999})
	p.Enqueue("send-email", 4, EmailPayload{"alice@example.com", "Order Confirmation", "Your order ord-1 is confirmed."})
	p.Enqueue("create-shipment", 3, ShipmentPayload{"ord-1", "123 Main St"})
	p.Enqueue("charge-card", 2, nil) // will DLQ
	// Duplicate order job (idempotency guard).
	p.Enqueue("process-order", 3, OrderPayload{"ord-1", "c-1", 9999})
	fmt.Printf("  queued %d jobs\n", p.Pending())

	// ── RUN WORKERS ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Running workers ---")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	wg := p.RunWorkers(ctx, 2)

	// Wait until queue drains or timeout.
	deadline := time.After(280 * time.Millisecond)
	for {
		select {
		case <-deadline:
			goto done
		default:
			if p.Pending() == 0 {
				time.Sleep(20 * time.Millisecond) // let in-flight jobs finish
				goto done
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
done:
	cancel()
	wg.Wait()

	// ── RESULTS ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  stats: enqueued=%d done=%d failed_attempts=%d dlq=%d\n",
		p.Stats.Enqueued.Load(), p.Stats.Done.Load(),
		p.Stats.Failed.Load(), p.Stats.DLQed.Load())

	fmt.Println()
	fmt.Println("--- Dead Letter Queue ---")
	for _, j := range p.DLQ() {
		fmt.Printf("  DLQ: %s type=%s attempts=%d errors=%v\n",
			j.ID, j.Type, j.Attempts, j.Errors)
	}

	// ── OBSERVABILITY ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Job metrics ---")
	total := p.Stats.TotalTime.Load()
	done := p.Stats.Done.Load()
	if done > 0 {
		avg := time.Duration(total / done)
		fmt.Printf("  avg job duration: %s\n", avg.Round(time.Microsecond))
	}
}
