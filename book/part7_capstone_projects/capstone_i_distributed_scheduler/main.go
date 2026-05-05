// FILE: book/part7_capstone_projects/capstone_i_distributed_scheduler/main.go
// CAPSTONE I — Distributed Scheduler
// Leader election, cron expression parsing, no-miss job execution,
// distributed lock, and job history — no external dependencies.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_i_distributed_scheduler

package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CRON EXPRESSION PARSER
// ─────────────────────────────────────────────────────────────────────────────

// CronExpr represents a 5-field cron expression: minute hour day month weekday
type CronExpr struct {
	raw     string
	minutes [60]bool
	hours   [24]bool
	days    [32]bool // 1-31
	months  [13]bool // 1-12
	weekdays [7]bool // 0=Sun
}

func parseCronField(field string, min, max int, out []bool) error {
	if field == "*" {
		for i := min; i <= max; i++ {
			out[i] = true
		}
		return nil
	}
	// Handle */step
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step in %q", field)
		}
		for i := min; i <= max; i += step {
			out[i] = true
		}
		return nil
	}
	// Handle comma-separated values and ranges
	for _, part := range strings.Split(field, ",") {
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, e1 := strconv.Atoi(bounds[0])
			hi, e2 := strconv.Atoi(bounds[1])
			if e1 != nil || e2 != nil || lo > hi {
				return fmt.Errorf("invalid range %q", part)
			}
			for i := lo; i <= hi; i++ {
				out[i] = true
			}
		} else {
			v, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("invalid value %q", part)
			}
			out[v] = true
		}
	}
	return nil
}

func ParseCron(expr string) (*CronExpr, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}
	c := &CronExpr{raw: expr}
	if err := parseCronField(fields[0], 0, 59, c.minutes[:]); err != nil {
		return nil, fmt.Errorf("minutes: %w", err)
	}
	if err := parseCronField(fields[1], 0, 23, c.hours[:]); err != nil {
		return nil, fmt.Errorf("hours: %w", err)
	}
	if err := parseCronField(fields[2], 1, 31, c.days[:]); err != nil {
		return nil, fmt.Errorf("days: %w", err)
	}
	if err := parseCronField(fields[3], 1, 12, c.months[:]); err != nil {
		return nil, fmt.Errorf("months: %w", err)
	}
	if err := parseCronField(fields[4], 0, 6, c.weekdays[:]); err != nil {
		return nil, fmt.Errorf("weekdays: %w", err)
	}
	return c, nil
}

// Matches returns true if t matches this cron expression (minute precision).
func (c *CronExpr) Matches(t time.Time) bool {
	return c.minutes[t.Minute()] &&
		c.hours[t.Hour()] &&
		c.days[t.Day()] &&
		c.months[int(t.Month())] &&
		c.weekdays[int(t.Weekday())]
}

// NextAfter returns the next time at or after t that matches this expression.
func (c *CronExpr) NextAfter(t time.Time) time.Time {
	// Truncate to minute, step forward
	candidate := t.Truncate(time.Minute)
	for i := 0; i < 366*24*60; i++ {
		if c.Matches(candidate) {
			return candidate
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{} // unreachable for valid expressions
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB HISTORY (ring buffer)
// ─────────────────────────────────────────────────────────────────────────────

type JobRun struct {
	StartedAt time.Time
	Duration  time.Duration
	Error     error
}

func (r JobRun) String() string {
	status := "OK"
	if r.Error != nil {
		status = "ERR: " + r.Error.Error()
	}
	return fmt.Sprintf("started=%s duration=%v status=%s",
		r.StartedAt.Format("15:04:05"), r.Duration.Round(time.Millisecond), status)
}

type jobHistory struct {
	mu   sync.Mutex
	runs []JobRun
	cap  int
}

func newJobHistory(cap int) *jobHistory { return &jobHistory{cap: cap} }

func (h *jobHistory) Add(r JobRun) {
	h.mu.Lock()
	h.runs = append(h.runs, r)
	if len(h.runs) > h.cap {
		h.runs = h.runs[len(h.runs)-h.cap:]
	}
	h.mu.Unlock()
}

func (h *jobHistory) Last(n int) []JobRun {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n > len(h.runs) {
		n = len(h.runs)
	}
	return append([]JobRun{}, h.runs[len(h.runs)-n:]...)
}

// ─────────────────────────────────────────────────────────────────────────────
// DISTRIBUTED LOCK (in-memory simulation)
// ─────────────────────────────────────────────────────────────────────────────

type lockEntry struct {
	holder    string
	expiresAt time.Time
	token     int64
}

type distributedLock struct {
	mu      sync.Mutex
	locks   map[string]lockEntry
	counter atomic.Int64
}

func newDistributedLock() *distributedLock {
	return &distributedLock{locks: map[string]lockEntry{}}
}

func (dl *distributedLock) Acquire(key, holder string, ttl time.Duration) (int64, bool) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	if e, ok := dl.locks[key]; ok && time.Now().Before(e.expiresAt) {
		return 0, false // already held
	}
	token := dl.counter.Add(1)
	dl.locks[key] = lockEntry{holder: holder, expiresAt: time.Now().Add(ttl), token: token}
	return token, true
}

func (dl *distributedLock) Release(key string, token int64) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	e, ok := dl.locks[key]
	if !ok {
		return errors.New("lock not found")
	}
	if e.token != token {
		return errors.New("stale token: lock already transferred")
	}
	delete(dl.locks, key)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LEADER ELECTION
// ─────────────────────────────────────────────────────────────────────────────

type nodeRole int32

const (
	roleFollower nodeRole = iota
	roleLeader
)

type clusterState struct {
	mu            sync.Mutex
	leaderID      string
	epoch         int64
	lastHeartbeat time.Time
}

func (cs *clusterState) SetLeader(nodeID string, epoch int64) {
	cs.mu.Lock()
	cs.leaderID = nodeID
	cs.epoch = epoch
	cs.lastHeartbeat = time.Now()
	cs.mu.Unlock()
}

func (cs *clusterState) Heartbeat(nodeID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.leaderID != nodeID {
		return false
	}
	cs.lastHeartbeat = time.Now()
	return true
}

func (cs *clusterState) IsLeaderAlive(timeout time.Duration) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return time.Since(cs.lastHeartbeat) < timeout
}

func (cs *clusterState) Leader() string {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.leaderID
}

// ─────────────────────────────────────────────────────────────────────────────
// JOB DEFINITION
// ─────────────────────────────────────────────────────────────────────────────

type JobHandler func(ctx context.Context) error

type Job struct {
	Name    string
	Expr    *CronExpr
	Handler JobHandler
	history *jobHistory
}

// ─────────────────────────────────────────────────────────────────────────────
// SCHEDULER NODE
// ─────────────────────────────────────────────────────────────────────────────

type SchedulerNode struct {
	id      string
	role    atomic.Int32
	cluster *clusterState
	lock    *distributedLock
	jobs    []*Job
	mu      sync.RWMutex
	runs    atomic.Int64
	skips   atomic.Int64
}

func NewSchedulerNode(id string, cluster *clusterState, lock *distributedLock) *SchedulerNode {
	return &SchedulerNode{id: id, cluster: cluster, lock: lock}
}

func (n *SchedulerNode) Register(name, cronExpr string, handler JobHandler) error {
	expr, err := ParseCron(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron %q: %w", cronExpr, err)
	}
	n.mu.Lock()
	n.jobs = append(n.jobs, &Job{Name: name, Expr: expr, Handler: handler, history: newJobHistory(10)})
	n.mu.Unlock()
	return nil
}

func (n *SchedulerNode) IsLeader() bool {
	return nodeRole(n.role.Load()) == roleLeader
}

func (n *SchedulerNode) ElectSelf(epoch int64) {
	n.role.Store(int32(roleLeader))
	n.cluster.SetLeader(n.id, epoch)
	fmt.Printf("  [%s] elected as leader (epoch=%d)\n", n.id, epoch)
}

func (n *SchedulerNode) StepDown() {
	n.role.Store(int32(roleFollower))
	fmt.Printf("  [%s] stepped down to follower\n", n.id)
}

// Tick simulates one scheduler tick at time t. Only leader runs jobs.
func (n *SchedulerNode) Tick(ctx context.Context, t time.Time) {
	if !n.IsLeader() {
		return
	}
	n.mu.RLock()
	jobs := append([]*Job{}, n.jobs...)
	n.mu.RUnlock()

	for _, job := range jobs {
		if !job.Expr.Matches(t) {
			continue
		}
		// Try to acquire distributed lock for this job+minute
		lockKey := fmt.Sprintf("job:%s:%s", job.Name, t.Truncate(time.Minute).Format("200601021504"))
		token, acquired := n.lock.Acquire(lockKey, n.id, 30*time.Second)
		if !acquired {
			n.skips.Add(1)
			fmt.Printf("  [%s] SKIP %s (another node holds lock)\n", n.id, job.Name)
			continue
		}

		start := time.Now()
		err := job.Handler(ctx)
		duration := time.Since(start)
		job.history.Add(JobRun{StartedAt: start, Duration: duration, Error: err})
		n.lock.Release(lockKey, token) //nolint:errcheck
		n.runs.Add(1)

		status := "OK"
		if err != nil {
			status = "ERROR: " + err.Error()
		}
		fmt.Printf("  [%s] RAN %s in %v → %s\n", n.id, job.Name, duration.Round(time.Microsecond), status)
	}
}

// CatchUp fires any jobs that were missed between lastRun and now.
func (n *SchedulerNode) CatchUp(ctx context.Context, lastRun, now time.Time) {
	n.mu.RLock()
	jobs := append([]*Job{}, n.jobs...)
	n.mu.RUnlock()

	missed := 0
	t := lastRun.Add(time.Minute)
	for !t.After(now) {
		for _, job := range jobs {
			if job.Expr.Matches(t) {
				missed++
				fmt.Printf("  [%s] CATCHUP %s for missed slot %s\n",
					n.id, job.Name, t.Format("15:04"))
				start := time.Now()
				err := job.Handler(ctx)
				job.history.Add(JobRun{StartedAt: start, Duration: time.Since(start), Error: err})
				n.runs.Add(1)
			}
		}
		t = t.Add(time.Minute)
	}
	if missed > 0 {
		fmt.Printf("  [%s] catch-up complete: %d missed executions fired\n", n.id, missed)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone I: Distributed Scheduler ===")
	fmt.Println()

	// ── CRON PARSER DEMO ──────────────────────────────────────────────────────
	fmt.Println("--- Cron expression parsing ---")
	exprs := []string{
		"* * * * *",       // every minute
		"*/5 * * * *",     // every 5 minutes
		"0 * * * *",       // top of every hour
		"0 9 * * 1-5",     // 9am weekdays
		"30 18 1 * *",     // 6:30pm on 1st of month
	}
	base := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Monday
	for _, expr := range exprs {
		c, err := ParseCron(expr)
		if err != nil {
			fmt.Printf("  ERROR %q: %v\n", expr, err)
			continue
		}
		next := c.NextAfter(base)
		matches := c.Matches(base)
		fmt.Printf("  %-25s matches(09:00 Mon)=%-5v next=%s\n",
			expr, matches, next.Format("Mon 15:04"))
	}
	fmt.Println()

	// ── LEADER ELECTION ───────────────────────────────────────────────────────
	fmt.Println("--- Leader election across 3 nodes ---")
	cluster := &clusterState{}
	lock := newDistributedLock()

	nodeA := NewSchedulerNode("node-A", cluster, lock)
	nodeB := NewSchedulerNode("node-B", cluster, lock)
	nodeC := NewSchedulerNode("node-C", cluster, lock)

	// Register jobs on all nodes (each node knows all jobs)
	for _, n := range []*SchedulerNode{nodeA, nodeB, nodeC} {
		n.Register("report-gen", "0 * * * *", func(ctx context.Context) error { //nolint:errcheck
			return nil
		})
		n.Register("data-cleanup", "*/5 * * * *", func(ctx context.Context) error { //nolint:errcheck
			return nil
		})
		n.Register("health-check", "* * * * *", func(ctx context.Context) error { //nolint:errcheck
			return nil
		})
	}

	// Node A wins election
	nodeA.ElectSelf(1)
	fmt.Printf("  Cluster leader: %s\n\n", cluster.Leader())

	// ── NORMAL EXECUTION ──────────────────────────────────────────────────────
	fmt.Println("--- Normal execution (node-A is leader) ---")
	ctx := context.Background()
	// Simulate ticks at :00, :05, :10
	ticks := []time.Time{
		time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 15, 9, 5, 0, 0, time.UTC),
		time.Date(2024, 1, 15, 9, 10, 0, 0, time.UTC),
	}
	for _, t := range ticks {
		fmt.Printf("  tick %s\n", t.Format("15:04"))
		nodeA.Tick(ctx, t)
		nodeB.Tick(ctx, t) // follower — does nothing
	}
	fmt.Printf("  node-A: runs=%d skips=%d\n\n", nodeA.runs.Load(), nodeA.skips.Load())

	// ── DISTRIBUTED LOCK CONTENTION ───────────────────────────────────────────
	fmt.Println("--- Distributed lock: preventing duplicate execution ---")
	t := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	// Both nodes try to run the same job at the same minute
	fmt.Printf("  Both node-A and node-B tick at %s:\n", t.Format("15:04"))
	nodeB.ElectSelf(2) // simulate split-brain: both think they're leader
	nodeA.Tick(ctx, t)
	nodeB.Tick(ctx, t) // should skip — node-A already holds the lock
	fmt.Printf("  node-A: runs=%d  node-B: runs=%d skips=%d\n\n",
		nodeA.runs.Load(), nodeB.runs.Load(), nodeB.skips.Load())

	// Reset: node-A stays leader
	nodeB.StepDown()

	// ── CATCH-UP AFTER FAILOVER ────────────────────────────────────────────────
	fmt.Println("--- Catch-up: node-B promoted after 10-min outage ---")
	nodeB.ElectSelf(3)
	lastRun := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
	now := time.Date(2024, 1, 15, 11, 10, 0, 0, time.UTC)
	fmt.Printf("  Last known execution: %s, current time: %s\n",
		lastRun.Format("15:04"), now.Format("15:04"))
	nodeB.CatchUp(ctx, lastRun, now)
	fmt.Println()

	// ── JOB HISTORY ───────────────────────────────────────────────────────────
	fmt.Println("--- Job execution history ---")
	for _, job := range nodeA.jobs {
		runs := job.history.Last(3)
		fmt.Printf("  %s: %d recorded runs\n", job.Name, len(runs))
		for _, r := range runs {
			fmt.Printf("    %s\n", r)
		}
	}
	fmt.Println()

	// ── NEXT RUN TIMES ────────────────────────────────────────────────────────
	fmt.Println("--- Next scheduled run times (from 09:00 Mon Jan 15) ---")
	for _, job := range nodeA.jobs {
		next := job.Expr.NextAfter(base)
		fmt.Printf("  %-15s next=%s  expr=%q\n", job.Name, next.Format("Mon 15:04"), job.Expr.raw)
	}
}
