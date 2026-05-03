// FILE: book/part5_building_backends/chapter72_message_queues/examples/01_queue_patterns/main.go
// CHAPTER: 72 — Message Queues
// TOPIC: In-memory message queue with at-least-once delivery, worker pool,
//        retry with backoff, and dead-letter queue.
//
// Run (from the chapter folder):
//   go run ./examples/01_queue_patterns

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
// MESSAGE
// ─────────────────────────────────────────────────────────────────────────────

type Message struct {
	ID        string
	Topic     string
	Payload   []byte
	Timestamp time.Time
	Attempts  int
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY QUEUE
// ─────────────────────────────────────────────────────────────────────────────

type Queue struct {
	name       string
	ch         chan *Message
	dlq        chan *Message // dead-letter queue
	maxRetries int
	mu         sync.Mutex
	inflight   map[string]*Message // messages being processed
}

func NewQueue(name string, size, maxRetries int) *Queue {
	return &Queue{
		name:       name,
		ch:         make(chan *Message, size),
		dlq:        make(chan *Message, size),
		maxRetries: maxRetries,
		inflight:   make(map[string]*Message),
	}
}

func (q *Queue) Publish(msg *Message) error {
	select {
	case q.ch <- msg:
		return nil
	default:
		return fmt.Errorf("queue %s: full", q.name)
	}
}

// Receive blocks until a message is available or ctx is done.
func (q *Queue) Receive(ctx context.Context) (*Message, error) {
	select {
	case msg := <-q.ch:
		q.mu.Lock()
		q.inflight[msg.ID] = msg
		q.mu.Unlock()
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Ack acknowledges successful processing.
func (q *Queue) Ack(msgID string) {
	q.mu.Lock()
	delete(q.inflight, msgID)
	q.mu.Unlock()
}

// Nack requeues (with attempt increment) or sends to DLQ.
func (q *Queue) Nack(msgID string) {
	q.mu.Lock()
	msg, ok := q.inflight[msgID]
	if !ok {
		q.mu.Unlock()
		return
	}
	delete(q.inflight, msgID)
	q.mu.Unlock()

	msg.Attempts++
	if msg.Attempts >= q.maxRetries {
		select {
		case q.dlq <- msg:
		default:
		}
		return
	}
	// Requeue.
	select {
	case q.ch <- msg:
	default:
	}
}

func (q *Queue) DLQ() <-chan *Message { return q.dlq }
func (q *Queue) Len() int             { return len(q.ch) }

// ─────────────────────────────────────────────────────────────────────────────
// WORKER POOL
// ─────────────────────────────────────────────────────────────────────────────

type WorkerPool struct {
	queue      *Queue
	numWorkers int
	handler    func(msg *Message) error
}

func NewWorkerPool(q *Queue, n int, handler func(*Message) error) *WorkerPool {
	return &WorkerPool{queue: q, numWorkers: n, handler: handler}
}

func (wp *WorkerPool) Start(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < wp.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				msg, err := wp.queue.Receive(ctx)
				if err != nil {
					return // context cancelled
				}
				if err := wp.handler(msg); err != nil {
					wp.queue.Nack(msg.ID)
				} else {
					wp.queue.Ack(msg.ID)
				}
			}
		}(i)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Message Queue Patterns ===")
	fmt.Println()

	// ── BASIC PUBLISH / CONSUME ──────────────────────────────────────────────
	fmt.Println("--- Basic publish / consume ---")
	q := NewQueue("orders", 100, 3)

	for i := 1; i <= 5; i++ {
		q.Publish(&Message{
			ID:        fmt.Sprintf("msg-%d", i),
			Topic:     "orders",
			Payload:   []byte(fmt.Sprintf(`{"order_id":"ord-%d"}`, i)),
			Timestamp: time.Now(),
		})
	}
	fmt.Printf("  queued: %d messages\n", q.Len())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for i := 0; i < 5; i++ {
		msg, _ := q.Receive(ctx)
		fmt.Printf("  consumed: %s payload=%s\n", msg.ID, msg.Payload)
		q.Ack(msg.ID)
	}

	// ── RETRY + DEAD LETTER ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Retry → Dead Letter Queue (maxRetries=3) ---")

	q2 := NewQueue("payments", 100, 3)
	q2.Publish(&Message{ID: "pay-1", Topic: "payments", Payload: []byte(`{"amount":99}`), Timestamp: time.Now()})

	// Simulate 3 failures → DLQ.
	msg, _ := q2.Receive(ctx)
	fmt.Printf("  attempt %d: NACK\n", msg.Attempts+1)
	q2.Nack(msg.ID)

	msg2, _ := q2.Receive(ctx)
	fmt.Printf("  attempt %d: NACK\n", msg2.Attempts+1)
	q2.Nack(msg2.ID)

	msg3, _ := q2.Receive(ctx)
	fmt.Printf("  attempt %d: NACK → dead letter\n", msg3.Attempts+1)
	q2.Nack(msg3.ID)

	dlqMsg := <-q2.DLQ()
	fmt.Printf("  DLQ received: %s attempts=%d\n", dlqMsg.ID, dlqMsg.Attempts)

	// ── WORKER POOL ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Worker pool (3 workers, 20 messages) ---")

	q3 := NewQueue("tasks", 200, 2)
	var processed atomic.Int64
	var failed atomic.Int64

	for i := 1; i <= 20; i++ {
		q3.Publish(&Message{
			ID:      fmt.Sprintf("task-%d", i),
			Payload: []byte(fmt.Sprintf(`{"task":%d}`, i)),
		})
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	var wg sync.WaitGroup
	pool := NewWorkerPool(q3, 3, func(msg *Message) error {
		// 20% failure rate.
		if rand.Float64() < 0.2 {
			failed.Add(1)
			return fmt.Errorf("simulated failure")
		}
		processed.Add(1)
		return nil
	})
	pool.Start(ctx2, &wg)
	wg.Wait()

	fmt.Printf("  processed=%d failed_attempts=%d\n", processed.Load(), failed.Load())

	// ── AT-LEAST-ONCE DELIVERY ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- At-least-once delivery explanation ---")
	fmt.Println("  • Message stays in-flight map until ACKed or NACKed")
	fmt.Println("  • If worker crashes before ACK → message redelivered on restart")
	fmt.Println("  • Consumer must be idempotent (deduplicate by msg.ID)")
	fmt.Println("  • Exactly-once requires distributed transactions or dedup store")
}
