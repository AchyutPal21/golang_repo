// FILE: book/part5_building_backends/chapter77_background_jobs/examples/02_scheduler/main.go
// CHAPTER: 77 — Background Jobs and Schedulers
// TOPIC: Cron scheduler, one-shot delayed jobs, distributed lock to prevent
//        duplicate execution across instances, and job lifecycle hooks.
//
// Run (from the chapter folder):
//   go run ./examples/02_scheduler

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CRON EXPRESSION (simplified: second-level precision for demo)
// ─────────────────────────────────────────────────────────────────────────────

type Schedule interface {
	Next(from time.Time) time.Time
}

// Every fires at a fixed interval.
type Every struct{ d time.Duration }

func (e Every) Next(from time.Time) time.Time { return from.Add(e.d) }

// At fires once at a specific absolute time.
type At struct{ t time.Time }

func (a At) Next(from time.Time) time.Time {
	if from.Before(a.t) {
		return a.t
	}
	return time.Time{} // zero = never again
}

// Daily fires once per day at a given hour:minute.
type Daily struct{ Hour, Minute int }

func (d Daily) Next(from time.Time) time.Time {
	candidate := time.Date(from.Year(), from.Month(), from.Day(), d.Hour, d.Minute, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

// ─────────────────────────────────────────────────────────────────────────────
// SCHEDULED TASK
// ─────────────────────────────────────────────────────────────────────────────

type Task struct {
	Name     string
	Schedule Schedule
	Run      func(ctx context.Context) error
	OnError  func(err error)
	nextRun  time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// SCHEDULER
// ─────────────────────────────────────────────────────────────────────────────

type Scheduler struct {
	mu    sync.Mutex
	tasks []*Task
	now   func() time.Time // injectable for testing
	Runs  atomic.Int64
	Errs  atomic.Int64
}

func NewScheduler() *Scheduler {
	return &Scheduler{now: time.Now}
}

func (s *Scheduler) Add(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task.nextRun = task.Schedule.Next(s.now())
	s.tasks = append(s.tasks, task)
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.mu.Lock()
			var due []*Task
			for _, t := range s.tasks {
				if !t.nextRun.IsZero() && !now.Before(t.nextRun) {
					due = append(due, t)
				}
			}
			s.mu.Unlock()

			for _, t := range due {
				t := t
				go func() {
					s.Runs.Add(1)
					fmt.Printf("  [scheduler] running %q at %s\n", t.Name, now.Format("15:04:05.000"))
					if err := t.Run(ctx); err != nil {
						s.Errs.Add(1)
						if t.OnError != nil {
							t.OnError(err)
						}
					}
					next := t.Schedule.Next(now)
					s.mu.Lock()
					t.nextRun = next
					s.mu.Unlock()
				}()
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DISTRIBUTED LOCK (prevents duplicate execution across instances)
// ─────────────────────────────────────────────────────────────────────────────

type DistributedLock struct {
	mu      sync.Mutex
	holders map[string]string // key → ownerID
	expiry  map[string]time.Time
}

func NewDistributedLock() *DistributedLock {
	return &DistributedLock{
		holders: make(map[string]string),
		expiry:  make(map[string]time.Time),
	}
}

func (dl *DistributedLock) Acquire(key, ownerID string, ttl time.Duration) bool {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	now := time.Now()
	exp, exists := dl.expiry[key]
	if exists && now.Before(exp) {
		return false // lock held by someone else
	}
	dl.holders[key] = ownerID
	dl.expiry[key] = now.Add(ttl)
	return true
}

func (dl *DistributedLock) Release(key, ownerID string) bool {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	if dl.holders[key] != ownerID {
		return false // don't release someone else's lock
	}
	delete(dl.holders, key)
	delete(dl.expiry, key)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB LIFECYCLE HOOKS
// ─────────────────────────────────────────────────────────────────────────────

type JobEvent string

const (
	EventBeforeRun JobEvent = "before_run"
	EventAfterRun  JobEvent = "after_run"
	EventOnError   JobEvent = "on_error"
	EventOnRetry   JobEvent = "on_retry"
)

type HookFn func(event JobEvent, jobName string, err error)

type HookedScheduler struct {
	*Scheduler
	hooks []HookFn
}

func NewHookedScheduler(hooks ...HookFn) *HookedScheduler {
	return &HookedScheduler{Scheduler: NewScheduler(), hooks: hooks}
}

func (hs *HookedScheduler) AddHooked(name string, schedule Schedule, fn func(ctx context.Context) error) {
	hs.Add(&Task{
		Name:     name,
		Schedule: schedule,
		Run: func(ctx context.Context) error {
			for _, h := range hs.hooks {
				h(EventBeforeRun, name, nil)
			}
			err := fn(ctx)
			event := EventAfterRun
			if err != nil {
				event = EventOnError
			}
			for _, h := range hs.hooks {
				h(event, name, err)
			}
			return err
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// DELAYED JOB
// ─────────────────────────────────────────────────────────────────────────────

type DelayedJobQueue struct {
	mu   sync.Mutex
	jobs []delayedJob
}

type delayedJob struct {
	runAt   time.Time
	fn      func()
}

func (dq *DelayedJobQueue) Schedule(delay time.Duration, fn func()) {
	dq.mu.Lock()
	defer dq.mu.Unlock()
	dq.jobs = append(dq.jobs, delayedJob{runAt: time.Now().Add(delay), fn: fn})
}

func (dq *DelayedJobQueue) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			dq.mu.Lock()
			var remaining []delayedJob
			for _, j := range dq.jobs {
				if now.After(j.runAt) {
					go j.fn()
				} else {
					remaining = append(remaining, j)
				}
			}
			dq.jobs = remaining
			dq.mu.Unlock()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Scheduler ===")
	fmt.Println()

	// ── RECURRING TASKS ───────────────────────────────────────────────────────
	fmt.Println("--- Recurring tasks (every 50ms, every 80ms) ---")

	sched := NewScheduler()
	var runCounts sync.Map

	sched.Add(&Task{
		Name:     "metrics-flush",
		Schedule: Every{50 * time.Millisecond},
		Run: func(ctx context.Context) error {
			n, _ := runCounts.LoadOrStore("metrics-flush", new(atomic.Int64))
			n.(*atomic.Int64).Add(1)
			return nil
		},
	})

	sched.Add(&Task{
		Name:     "cleanup",
		Schedule: Every{80 * time.Millisecond},
		Run: func(ctx context.Context) error {
			n, _ := runCounts.LoadOrStore("cleanup", new(atomic.Int64))
			n.(*atomic.Int64).Add(1)
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 220*time.Millisecond)
	defer cancel()
	go sched.Run(ctx)
	<-ctx.Done()

	runCounts.Range(func(k, v any) bool {
		fmt.Printf("  task=%s runs=%d\n", k, v.(*atomic.Int64).Load())
		return true
	})

	// ── DELAYED JOBS ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Delayed (one-shot) jobs ---")

	dq := &DelayedJobQueue{}
	delayedCtx, delayedCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer delayedCancel()

	go dq.Run(delayedCtx)

	var fired sync.Map
	dq.Schedule(20*time.Millisecond, func() {
		fmt.Println("  [delayed] send welcome email (20ms)")
		fired.Store("welcome", true)
	})
	dq.Schedule(60*time.Millisecond, func() {
		fmt.Println("  [delayed] generate report (60ms)")
		fired.Store("report", true)
	})
	dq.Schedule(100*time.Millisecond, func() {
		fmt.Println("  [delayed] cleanup temp files (100ms)")
		fired.Store("cleanup", true)
	})

	<-delayedCtx.Done()
	time.Sleep(10 * time.Millisecond) // allow last goroutines to print

	// ── DISTRIBUTED LOCK ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Distributed lock: prevent duplicate job execution ---")

	lock := NewDistributedLock()
	var execCount atomic.Int32
	var wg sync.WaitGroup

	// Simulate 3 scheduler instances trying to run the same cron job.
	for i := 1; i <= 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			ownerID := fmt.Sprintf("instance-%d", i)
			if lock.Acquire("daily-report", ownerID, 5*time.Second) {
				execCount.Add(1)
				fmt.Printf("  [%s] acquired lock, running daily-report\n", ownerID)
				time.Sleep(5 * time.Millisecond) // simulate work
				lock.Release("daily-report", ownerID)
			} else {
				fmt.Printf("  [%s] lock held, skipping\n", ownerID)
			}
		}()
	}
	wg.Wait()
	fmt.Printf("  daily-report executed %d time(s) (should be 1)\n", execCount.Load())

	// ── LIFECYCLE HOOKS ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Job lifecycle hooks ---")

	hs := NewHookedScheduler(func(event JobEvent, name string, err error) {
		if event == EventBeforeRun {
			fmt.Printf("  [hook] BEFORE %s\n", name)
		} else if event == EventAfterRun {
			fmt.Printf("  [hook] AFTER  %s ok\n", name)
		} else if event == EventOnError {
			fmt.Printf("  [hook] ERROR  %s: %v\n", name, err)
		}
	})

	hookCtx, hookCancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer hookCancel()

	var hookRuns atomic.Int32
	hs.AddHooked("data-sync", Every{30 * time.Millisecond}, func(ctx context.Context) error {
		hookRuns.Add(1)
		if hookRuns.Load() == 2 {
			return fmt.Errorf("transient sync error")
		}
		return nil
	})

	go hs.Run(hookCtx)
	<-hookCtx.Done()
	time.Sleep(10 * time.Millisecond)
}
