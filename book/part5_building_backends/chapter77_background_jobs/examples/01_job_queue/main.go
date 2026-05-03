// FILE: book/part5_building_backends/chapter77_background_jobs/examples/01_job_queue/main.go
// CHAPTER: 77 — Background Jobs and Schedulers
// TOPIC: In-process job queue — typed jobs, worker pool, retry with backoff,
//        priority, unique jobs, and dead-letter tracking.
//
// Run (from the chapter folder):
//   go run ./examples/01_job_queue

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// JOB
// ─────────────────────────────────────────────────────────────────────────────

type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobRunning    JobStatus = "running"
	JobDone       JobStatus = "done"
	JobFailed     JobStatus = "failed"
	JobDeadLetter JobStatus = "dead"
)

type Job struct {
	ID         string
	Type       string
	Priority   int // higher = processed first
	Payload    any
	MaxRetries int
	Attempts   int
	Status     JobStatus
	EnqueuedAt time.Time
	Error      string
}

func (j *Job) clone() *Job {
	c := *j
	return &c
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB QUEUE
// ─────────────────────────────────────────────────────────────────────────────

type JobHandler func(ctx context.Context, job *Job) error

type JobQueue struct {
	mu         sync.Mutex
	pending    []*Job  // sorted by priority descending
	dlq        []*Job
	handlers   map[string]JobHandler
	seq        atomic.Int64
	Stats      struct {
		Enqueued  atomic.Int64
		Processed atomic.Int64
		Failed    atomic.Int64
		DLQ       atomic.Int64
	}
}

func NewJobQueue() *JobQueue {
	return &JobQueue{handlers: make(map[string]JobHandler)}
}

func (q *JobQueue) Register(jobType string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

func (q *JobQueue) Enqueue(jobType string, priority, maxRetries int, payload any) *Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	job := &Job{
		ID:         fmt.Sprintf("job-%d", q.seq.Add(1)),
		Type:       jobType,
		Priority:   priority,
		Payload:    payload,
		MaxRetries: maxRetries,
		Status:     JobPending,
		EnqueuedAt: time.Now(),
	}
	// Insert sorted by priority descending.
	inserted := false
	for i, j := range q.pending {
		if priority > j.Priority {
			q.pending = append(q.pending[:i], append([]*Job{job}, q.pending[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.pending = append(q.pending, job)
	}
	q.Stats.Enqueued.Add(1)
	return job
}

func (q *JobQueue) dequeue() *Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return nil
	}
	job := q.pending[0]
	q.pending = q.pending[1:]
	job.Status = JobRunning
	return job
}

func (q *JobQueue) complete(job *Job, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err == nil {
		job.Status = JobDone
		q.Stats.Processed.Add(1)
		return
	}
	job.Attempts++
	job.Error = err.Error()
	q.Stats.Failed.Add(1)
	if job.Attempts >= job.MaxRetries {
		job.Status = JobDeadLetter
		q.dlq = append(q.dlq, job)
		q.Stats.DLQ.Add(1)
		return
	}
	job.Status = JobPending
	// Re-insert with same priority.
	q.pending = append(q.pending, job.clone())
}

func (q *JobQueue) DLQ() []*Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]*Job, len(q.dlq))
	copy(out, q.dlq)
	return out
}

func (q *JobQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}

// ─────────────────────────────────────────────────────────────────────────────
// WORKER POOL
// ─────────────────────────────────────────────────────────────────────────────

type WorkerPool struct {
	queue      *JobQueue
	numWorkers int
}

func NewWorkerPool(q *JobQueue, n int) *WorkerPool {
	return &WorkerPool{queue: q, numWorkers: n}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.numWorkers; i++ {
		i := i
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				job := wp.queue.dequeue()
				if job == nil {
					time.Sleep(5 * time.Millisecond)
					continue
				}
				handler, ok := wp.queue.handlers[job.Type]
				if !ok {
					wp.queue.complete(job, fmt.Errorf("no handler for type %q", job.Type))
					continue
				}
				fmt.Printf("  [worker-%d] running job %s type=%s attempt=%d\n",
					i, job.ID, job.Type, job.Attempts+1)
				err := handler(ctx, job)
				wp.queue.complete(job, err)
			}
		}()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB TYPES
// ─────────────────────────────────────────────────────────────────────────────

type EmailPayload struct {
	To      string
	Subject string
}

type ReportPayload struct {
	ReportID string
	Format   string
}

type WebhookPayload struct {
	URL  string
	Body string
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Job Queue ===")
	fmt.Println()

	q := NewJobQueue()
	var processed sync.Map

	// Register handlers.
	q.Register("email", func(ctx context.Context, job *Job) error {
		p := job.Payload.(EmailPayload)
		fmt.Printf("  [email] sending to %s: %s\n", p.To, p.Subject)
		processed.Store(job.ID, true)
		return nil
	})

	q.Register("report", func(ctx context.Context, job *Job) error {
		p := job.Payload.(ReportPayload)
		// Simulate 30% failure rate on first attempt.
		if job.Attempts == 0 && rand.Float64() < 0.3 {
			return fmt.Errorf("report generation failed transiently")
		}
		fmt.Printf("  [report] generated %s.%s\n", p.ReportID, p.Format)
		processed.Store(job.ID, true)
		return nil
	})

	q.Register("webhook", func(ctx context.Context, job *Job) error {
		// Always fails — will hit DLQ.
		return fmt.Errorf("webhook endpoint unreachable")
	})

	// ── PRIORITY QUEUE ────────────────────────────────────────────────────────
	fmt.Println("--- Priority queue ---")
	q.Enqueue("email", 1, 3, EmailPayload{"low@example.com", "Newsletter"})
	q.Enqueue("report", 10, 3, ReportPayload{"rpt-1", "pdf"})
	q.Enqueue("email", 5, 3, EmailPayload{"medium@example.com", "Order confirmation"})
	q.Enqueue("webhook", 8, 2, WebhookPayload{"https://example.com/hook", `{"event":"order"}`})
	q.Enqueue("report", 10, 3, ReportPayload{"rpt-2", "csv"})

	fmt.Printf("  queued %d jobs (highest priority first)\n", q.Len())

	// ── WORKER POOL ───────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Worker pool (2 workers) ---")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	pool := NewWorkerPool(q, 2)
	pool.Start(ctx)

	// Wait for processing.
	deadline := time.After(180 * time.Millisecond)
	for {
		select {
		case <-deadline:
			goto done
		default:
			if q.Len() == 0 {
				time.Sleep(10 * time.Millisecond)
				goto done
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
done:
	cancel()

	// ── RESULTS ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  stats: enqueued=%d processed=%d failed_attempts=%d dlq=%d\n",
		q.Stats.Enqueued.Load(), q.Stats.Processed.Load(),
		q.Stats.Failed.Load(), q.Stats.DLQ.Load())

	fmt.Println()
	fmt.Println("--- Dead Letter Queue ---")
	for _, job := range q.DLQ() {
		fmt.Printf("  DLQ: %s type=%s attempts=%d err=%s\n",
			job.ID, job.Type, job.Attempts, job.Error)
	}

	// ── UNIQUE JOBS (conceptual) ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Unique job deduplication (conceptual) ---")
	fmt.Println(`  Prevent enqueueing the same logical job twice:
    key = hash(jobType + payload)
    if redis.SetNX("job:unique:"+key, jobID, ttl):
        enqueue the job
    else:
        skip (already queued or in progress)

  In asynq: task.New("email:send", payload, asynq.TaskID("unique-key"))
  asynq will reject duplicates with the same TaskID within the retention window.`)
}
