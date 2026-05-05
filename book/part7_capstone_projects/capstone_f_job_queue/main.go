// FILE: book/part7_capstone_projects/capstone_f_job_queue/main.go
// CAPSTONE F — Job Queue
// Self-contained simulation: priority queue, visibility timeout, dead-letter queue,
// worker pool, and job-handler dispatch — no external deps.
//
// Run:
//   go run ./part7_capstone_projects/capstone_f_job_queue

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PRIORITY TYPE
// ─────────────────────────────────────────────────────────────────────────────

// Priority controls the order in which jobs are dequeued.
// Lower numeric value == higher urgency.
type Priority int

const (
	PriorityHigh   Priority = 1
	PriorityNormal Priority = 2
	PriorityLow    Priority = 3
)

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "High"
	case PriorityNormal:
		return "Normal"
	case PriorityLow:
		return "Low"
	default:
		return "Unknown"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB
// ─────────────────────────────────────────────────────────────────────────────

// Job is the unit of work flowing through the queue.
type Job struct {
	ID                string
	Type              string        // used to look up the handler
	Payload           string        // arbitrary data; stringified for simplicity
	Priority          Priority
	Attempts          int           // how many times this job has been tried
	MaxAttempts       int           // attempts >= MaxAttempts → DLQ
	VisibilityTimeout time.Duration // how long a dequeued job stays invisible
	EnqueuedAt        time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// DEAD-LETTER QUEUE
// ─────────────────────────────────────────────────────────────────────────────

// DeadLetterQueue holds jobs that have exhausted their retry budget.
type DeadLetterQueue struct {
	mu   sync.Mutex
	jobs []Job
}

// Add appends a job to the DLQ.
func (dlq *DeadLetterQueue) Add(job Job) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	dlq.jobs = append(dlq.jobs, job)
}

// List returns a snapshot of all dead-lettered jobs.
func (dlq *DeadLetterQueue) List() []Job {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	out := make([]Job, len(dlq.jobs))
	copy(out, dlq.jobs)
	return out
}

// Requeue moves a job from the DLQ back to the main queue with reset attempts.
// Returns an error if the job ID is not found in the DLQ.
func (dlq *DeadLetterQueue) Requeue(jobID string, q *Queue) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	for i, j := range dlq.jobs {
		if j.ID == jobID {
			dlq.jobs = append(dlq.jobs[:i], dlq.jobs[i+1:]...)
			j.Attempts = 0
			j.EnqueuedAt = time.Now()
			q.Enqueue(j)
			return nil
		}
	}
	return fmt.Errorf("dlq: job %q not found", jobID)
}

// ─────────────────────────────────────────────────────────────────────────────
// QUEUE
// ─────────────────────────────────────────────────────────────────────────────

// Queue is a thread-safe, priority-ordered job queue with visibility timeouts.
//
// Design notes
//   - jobs slice is kept sorted by Priority (ascending numeric value = higher urgency).
//     Insertion is O(n) which is fine for this simulation; a real system would use
//     a heap or multiple FIFO lanes (see scaling_discussion.md).
//   - A job is "invisible" while it is being processed: it is removed from jobs
//     and placed in inFlight with a deadline.  If the worker crashes before
//     Ack/Nack, a background reaper returns the job to jobs once the deadline
//     passes (at-least-once delivery).
type Queue struct {
	mu            sync.Mutex
	jobs          []Job                    // pending jobs, sorted by Priority
	inFlight      map[string]Job           // jobID → job currently being processed
	invisDeadline map[string]time.Time     // jobID → visibility-timeout deadline
	dlq           *DeadLetterQueue
	stats         *Stats
}

// NewQueue creates an empty Queue backed by the given DLQ and stats.
func NewQueue(dlq *DeadLetterQueue, stats *Stats) *Queue {
	q := &Queue{
		inFlight:      make(map[string]Job),
		invisDeadline: make(map[string]time.Time),
		dlq:           dlq,
		stats:         stats,
	}
	return q
}

// Enqueue adds a job to the queue in priority order.
func (q *Queue) Enqueue(job Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// sorted insertion: find first job with strictly lower urgency (higher numeric value)
	insertAt := len(q.jobs)
	for i, existing := range q.jobs {
		if existing.Priority > job.Priority {
			insertAt = i
			break
		}
	}

	q.jobs = append(q.jobs, Job{}) // grow by one
	copy(q.jobs[insertAt+1:], q.jobs[insertAt:])
	q.jobs[insertAt] = job

	atomic.AddInt64(&q.stats.Enqueued, 1)
}

// reapExpired moves any timed-out in-flight jobs back to the pending queue.
// Caller must NOT hold q.mu.
func (q *Queue) reapExpired() {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	for id, deadline := range q.invisDeadline {
		if now.After(deadline) {
			job := q.inFlight[id]
			delete(q.inFlight, id)
			delete(q.invisDeadline, id)

			// re-insert without incrementing Attempts; the original Nack path
			// increments Attempts, so timeout reappearance is a free retry.
			insertAt := len(q.jobs)
			for i, existing := range q.jobs {
				if existing.Priority > job.Priority {
					insertAt = i
					break
				}
			}
			q.jobs = append(q.jobs, Job{})
			copy(q.jobs[insertAt+1:], q.jobs[insertAt:])
			q.jobs[insertAt] = job

			fmt.Printf("  [reaper] job %s reappeared after visibility timeout\n", job.ID)
		}
	}
}

// Dequeue returns the highest-priority available job and marks it in-flight.
// Returns false if the queue is empty.
func (q *Queue) Dequeue() (Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) == 0 {
		return Job{}, false
	}

	// jobs[0] is always the highest priority due to sorted insertion
	job := q.jobs[0]
	q.jobs = q.jobs[1:]

	// mark invisible until the visibility deadline
	deadline := time.Now().Add(job.VisibilityTimeout)
	q.inFlight[job.ID] = job
	q.invisDeadline[job.ID] = deadline

	return job, true
}

// Ack marks a job as successfully processed and removes it permanently.
func (q *Queue) Ack(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.inFlight[jobID]; !ok {
		return fmt.Errorf("ack: job %q not in-flight", jobID)
	}
	delete(q.inFlight, jobID)
	delete(q.invisDeadline, jobID)
	atomic.AddInt64(&q.stats.Acked, 1)
	return nil
}

// Nack signals that a job failed.  It increments Attempts; if the job has
// exhausted MaxAttempts it is moved to the DLQ, otherwise it is re-enqueued.
func (q *Queue) Nack(jobID string) error {
	q.mu.Lock()

	job, ok := q.inFlight[jobID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("nack: job %q not in-flight", jobID)
	}
	delete(q.inFlight, jobID)
	delete(q.invisDeadline, jobID)
	q.mu.Unlock() // release before calling Enqueue / DLQ.Add (which acquire their own locks)

	job.Attempts++
	atomic.AddInt64(&q.stats.Nacked, 1)

	if job.Attempts >= job.MaxAttempts {
		atomic.AddInt64(&q.stats.DeadLettered, 1)
		q.dlq.Add(job)
		fmt.Printf("  [dlq]    job %-12s type=%-12s sent to DLQ after %d attempts\n",
			job.ID, job.Type, job.Attempts)
		return nil
	}

	// re-enqueue without touching the stats.Enqueued counter (it was already counted)
	q.mu.Lock()
	insertAt := len(q.jobs)
	for i, existing := range q.jobs {
		if existing.Priority > job.Priority {
			insertAt = i
			break
		}
	}
	q.jobs = append(q.jobs, Job{})
	copy(q.jobs[insertAt+1:], q.jobs[insertAt:])
	q.jobs[insertAt] = job
	q.mu.Unlock()

	return nil
}

// Len returns the number of pending (not in-flight) jobs.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB HANDLER INTERFACE + REGISTRY
// ─────────────────────────────────────────────────────────────────────────────

// JobHandler is implemented by anything that knows how to process a specific
// job type.
type JobHandler interface {
	Handle(job Job) error
}

// HandlerRegistry maps job type strings to their handlers.
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

// NewHandlerRegistry creates an empty registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]JobHandler)}
}

// Register associates a handler with a job type.
func (r *HandlerRegistry) Register(jobType string, h JobHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = h
}

// Dispatch looks up and invokes the handler for the job's type.
func (r *HandlerRegistry) Dispatch(job Job) error {
	r.mu.RLock()
	h, ok := r.handlers[job.Type]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no handler registered for type %q", job.Type)
	}
	return h.Handle(job)
}

// ─────────────────────────────────────────────────────────────────────────────
// CONCRETE HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

// EmailHandler simulates sending an email.  Always succeeds.
type EmailHandler struct{}

func (h *EmailHandler) Handle(job Job) error {
	time.Sleep(10 * time.Millisecond) // simulate work
	fmt.Printf("  [email]  job %-12s priority=%-6s payload=%q\n",
		job.ID, job.Priority, job.Payload)
	return nil
}

// ReportHandler simulates generating a PDF report.  Always succeeds.
type ReportHandler struct{}

func (h *ReportHandler) Handle(job Job) error {
	time.Sleep(20 * time.Millisecond)
	fmt.Printf("  [report] job %-12s priority=%-6s payload=%q\n",
		job.ID, job.Priority, job.Payload)
	return nil
}

// PoisonPillHandler always returns an error — used to exercise the DLQ path.
type PoisonPillHandler struct{}

func (h *PoisonPillHandler) Handle(job Job) error {
	time.Sleep(5 * time.Millisecond)
	return errors.New("poison-pill: intentional failure")
}

// ─────────────────────────────────────────────────────────────────────────────
// STATS
// ─────────────────────────────────────────────────────────────────────────────

// Stats holds atomic counters for queue-wide metrics.
type Stats struct {
	Enqueued     int64
	Processed    int64
	Acked        int64
	Nacked       int64
	DeadLettered int64
}

// ─────────────────────────────────────────────────────────────────────────────
// WORKER POOL
// ─────────────────────────────────────────────────────────────────────────────

// WorkerPool manages a fixed number of goroutines that drain the queue.
type WorkerPool struct {
	size     int
	queue    *Queue
	registry *HandlerRegistry
	stats    *Stats
	wg       sync.WaitGroup
}

// NewWorkerPool creates a pool that will launch size worker goroutines.
func NewWorkerPool(size int, q *Queue, r *HandlerRegistry, s *Stats) *WorkerPool {
	return &WorkerPool{
		size:     size,
		queue:    q,
		registry: r,
		stats:    s,
	}
}

// Start launches all worker goroutines.  Workers run until ctx is cancelled.
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := range wp.size {
		wp.wg.Add(1)
		go wp.runWorker(ctx, i+1)
	}
}

// Stop blocks until all workers have finished their current job and exited.
func (wp *WorkerPool) Stop() {
	wp.wg.Wait()
}

func (wp *WorkerPool) runWorker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		// Check cancellation first.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Reap any in-flight jobs whose visibility timeout has expired.
		wp.queue.reapExpired()

		job, ok := wp.queue.Dequeue()
		if !ok {
			// Nothing available; back off briefly before trying again.
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
			continue
		}

		atomic.AddInt64(&wp.stats.Processed, 1)
		fmt.Printf("  [worker%d] dequeued job %-12s type=%-12s priority=%s attempt=%d\n",
			id, job.ID, job.Type, job.Priority, job.Attempts+1)

		err := wp.registry.Dispatch(job)
		if err != nil {
			fmt.Printf("  [worker%d] FAIL job %-12s err=%v\n", id, job.ID, err)
			if nackErr := wp.queue.Nack(job.ID); nackErr != nil {
				fmt.Printf("  [worker%d] nack error: %v\n", id, nackErr)
			}
		} else {
			if ackErr := wp.queue.Ack(job.ID); ackErr != nil {
				fmt.Printf("  [worker%d] ack error: %v\n", id, ackErr)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone F — Job Queue Simulation ===")
	fmt.Println()

	// Shared infrastructure
	stats := &Stats{}
	dlq := &DeadLetterQueue{}
	q := NewQueue(dlq, stats)

	registry := NewHandlerRegistry()
	registry.Register("email", &EmailHandler{})
	registry.Register("report", &ReportHandler{})
	registry.Register("poison-pill", &PoisonPillHandler{})

	// ── Enqueue 10 mixed-priority, mixed-type jobs ──────────────────────────

	jobs := []Job{
		{ID: "J001", Type: "email", Payload: "welcome email", Priority: PriorityHigh, MaxAttempts: 3, VisibilityTimeout: 200 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J002", Type: "report", Payload: "monthly report", Priority: PriorityLow, MaxAttempts: 2, VisibilityTimeout: 300 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J003", Type: "poison-pill", Payload: "bad job #1", Priority: PriorityNormal, MaxAttempts: 3, VisibilityTimeout: 100 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J004", Type: "email", Payload: "password reset", Priority: PriorityHigh, MaxAttempts: 3, VisibilityTimeout: 200 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J005", Type: "report", Payload: "weekly digest", Priority: PriorityNormal, MaxAttempts: 2, VisibilityTimeout: 300 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J006", Type: "poison-pill", Payload: "bad job #2", Priority: PriorityLow, MaxAttempts: 2, VisibilityTimeout: 100 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J007", Type: "email", Payload: "invoice email", Priority: PriorityNormal, MaxAttempts: 3, VisibilityTimeout: 200 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J008", Type: "report", Payload: "audit report", Priority: PriorityHigh, MaxAttempts: 3, VisibilityTimeout: 300 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J009", Type: "email", Payload: "promo blast", Priority: PriorityLow, MaxAttempts: 2, VisibilityTimeout: 200 * time.Millisecond, EnqueuedAt: time.Now()},
		{ID: "J010", Type: "report", Payload: "q4 summary", Priority: PriorityNormal, MaxAttempts: 2, VisibilityTimeout: 300 * time.Millisecond, EnqueuedAt: time.Now()},
	}

	fmt.Println("── Enqueuing jobs ──────────────────────────────────────────────")
	for _, j := range jobs {
		q.Enqueue(j)
		fmt.Printf("  enqueued %-6s type=%-12s priority=%s\n", j.ID, j.Type, j.Priority)
	}
	fmt.Println()

	// ── Start 2 workers ─────────────────────────────────────────────────────

	fmt.Println("── Starting worker pool (2 workers) ────────────────────────────")
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPool(2, q, registry, stats)
	pool.Start(ctx)

	// Let the pool drain.  We wait until the queue is empty AND no jobs are
	// in-flight, with a maximum budget so the test never hangs.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		q.mu.Lock()
		pending := len(q.jobs)
		inFlight := len(q.inFlight)
		q.mu.Unlock()

		dlq.mu.Lock()
		dlqLen := len(dlq.jobs)
		dlq.mu.Unlock()

		// poison-pill jobs will cycle through retries and land in DLQ.
		// We're done when nothing is pending or in-flight.
		if pending == 0 && inFlight == 0 {
			// A short extra wait for any final Ack/Nack calls that might still
			// be running inside worker goroutines.
			time.Sleep(50 * time.Millisecond)
			_ = dlqLen
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	pool.Stop()
	fmt.Println()

	// ── Final stats ──────────────────────────────────────────────────────────

	fmt.Println("── Final stats ──────────────────────────────────────────────────")
	fmt.Printf("  Enqueued:      %d\n", atomic.LoadInt64(&stats.Enqueued))
	fmt.Printf("  Processed:     %d\n", atomic.LoadInt64(&stats.Processed))
	fmt.Printf("  Acked:         %d\n", atomic.LoadInt64(&stats.Acked))
	fmt.Printf("  Nacked:        %d\n", atomic.LoadInt64(&stats.Nacked))
	fmt.Printf("  Dead-lettered: %d\n", atomic.LoadInt64(&stats.DeadLettered))
	fmt.Println()

	// ── DLQ contents ────────────────────────────────────────────────────────

	dead := dlq.List()
	fmt.Printf("── Dead-Letter Queue (%d job(s)) ─────────────────────────────────\n", len(dead))
	for _, j := range dead {
		fmt.Printf("  id=%-6s type=%-12s priority=%-6s attempts=%d payload=%q\n",
			j.ID, j.Type, j.Priority, j.Attempts, j.Payload)
	}
	fmt.Println()

	// ── Demonstrate DLQ requeue ──────────────────────────────────────────────

	if len(dead) > 0 {
		target := dead[0]
		fmt.Printf("── Requeuing %s from DLQ (operator replay) ──────────────────────\n", target.ID)
		if err := dlq.Requeue(target.ID, q); err != nil {
			fmt.Printf("  requeue error: %v\n", err)
		} else {
			fmt.Printf("  %s is back in the queue (attempts reset to 0, len=%d)\n",
				target.ID, q.Len())
		}
	}

	fmt.Println()
	fmt.Println("=== Simulation complete ===")
}
