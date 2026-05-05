// FILE: book/part6_production_engineering/chapter98_incidents/exercises/01_incident_timeline/main.go
// CHAPTER: 98 — Incident Management & Debugging
// EXERCISE: Incident timeline builder, health snapshot, MTTR calculator,
//           and combined debugging toolkit.
//
// Run:
//   go run ./book/part6_production_engineering/chapter98_incidents/exercises/01_incident_timeline

package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// INCIDENT TIMELINE
// ─────────────────────────────────────────────────────────────────────────────

type EventSeverity int

const (
	Info EventSeverity = iota
	Warning
	Critical
	Resolved
)

func (s EventSeverity) String() string {
	switch s {
	case Info:
		return "INFO"
	case Warning:
		return "WARN"
	case Critical:
		return "CRIT"
	case Resolved:
		return "OK  "
	default:
		return "????"
	}
}

type IncidentEvent struct {
	Time     time.Time
	Severity EventSeverity
	Source   string
	Message  string
}

type IncidentTimeline struct {
	ID        string
	Title     string
	StartTime time.Time
	EndTime   time.Time
	Events    []IncidentEvent
}

func (it *IncidentTimeline) Add(sev EventSeverity, source, msg string) {
	it.Events = append(it.Events, IncidentEvent{
		Time:     time.Now(),
		Severity: sev,
		Source:   source,
		Message:  msg,
	})
}

func (it *IncidentTimeline) AddAt(t time.Time, sev EventSeverity, source, msg string) {
	it.Events = append(it.Events, IncidentEvent{t, sev, source, msg})
}

func (it *IncidentTimeline) Resolve() {
	it.EndTime = time.Now()
	it.Add(Resolved, "oncall", "Incident resolved")
}

func (it *IncidentTimeline) MTTR() time.Duration {
	if it.EndTime.IsZero() {
		return 0
	}
	return it.EndTime.Sub(it.StartTime)
}

func (it *IncidentTimeline) Print() {
	fmt.Printf("  Incident: %s (%s)\n", it.Title, it.ID)
	fmt.Printf("  %-6s  %-5s  %-18s  %s\n", "T+min", "Sev", "Source", "Event")
	fmt.Printf("  %s\n", strings.Repeat("-", 70))
	for _, e := range it.Events {
		offset := e.Time.Sub(it.StartTime).Round(time.Second)
		minutes := int(offset.Minutes())
		secs := int(offset.Seconds()) % 60
		fmt.Printf("  %3d:%02d  %s  %-18s  %s\n",
			minutes, secs, e.Severity, e.Source, e.Message)
	}
	if !it.EndTime.IsZero() {
		fmt.Printf("\n  MTTR: %v\n", it.MTTR().Round(time.Second))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HEALTH SNAPSHOT
// ─────────────────────────────────────────────────────────────────────────────

type HealthSnapshot struct {
	Timestamp   time.Time
	Goroutines  int
	HeapAllocMB float64
	TotalAllocMB float64
	GCCycles    uint32
	Mallocs     uint64
}

func TakeSnapshot() HealthSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return HealthSnapshot{
		Timestamp:    time.Now(),
		Goroutines:   runtime.NumGoroutine(),
		HeapAllocMB:  float64(m.Alloc) / (1024 * 1024),
		TotalAllocMB: float64(m.TotalAlloc) / (1024 * 1024),
		GCCycles:     m.NumGC,
		Mallocs:      m.Mallocs,
	}
}

func (s HealthSnapshot) Print() {
	fmt.Printf("  %-25s  %v\n", "Timestamp:", s.Timestamp.Format(time.RFC3339))
	fmt.Printf("  %-25s  %d\n", "Goroutines:", s.Goroutines)
	fmt.Printf("  %-25s  %.2f MB\n", "Heap alloc:", s.HeapAllocMB)
	fmt.Printf("  %-25s  %.2f MB\n", "Total alloc:", s.TotalAllocMB)
	fmt.Printf("  %-25s  %d\n", "GC cycles:", s.GCCycles)
	fmt.Printf("  %-25s  %d\n", "Mallocs:", s.Mallocs)
}

// ─────────────────────────────────────────────────────────────────────────────
// REPORT GENERATOR
// ─────────────────────────────────────────────────────────────────────────────

func generateIncidentReport(it *IncidentTimeline, snap HealthSnapshot) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Incident Report: %s\n", it.Title)
	fmt.Fprintf(&sb, "**ID:** %s  **MTTR:** %v\n\n", it.ID, it.MTTR().Round(time.Second))

	fmt.Fprintf(&sb, "## Timeline\n")
	for _, e := range it.Events {
		offset := e.Time.Sub(it.StartTime).Round(time.Second)
		fmt.Fprintf(&sb, "- `T+%v` [%s] **%s**: %s\n", offset, e.Severity, e.Source, e.Message)
	}

	fmt.Fprintf(&sb, "\n## Health Snapshot at Detection\n")
	fmt.Fprintf(&sb, "- Goroutines: %d\n", snap.Goroutines)
	fmt.Fprintf(&sb, "- Heap: %.2f MB\n", snap.HeapAllocMB)
	fmt.Fprintf(&sb, "- GC cycles: %d\n", snap.GCCycles)

	critCount := 0
	for _, e := range it.Events {
		if e.Severity == Critical {
			critCount++
		}
	}
	fmt.Fprintf(&sb, "\n## Summary\n")
	fmt.Fprintf(&sb, "- Total events: %d\n", len(it.Events))
	fmt.Fprintf(&sb, "- Critical events: %d\n", critCount)
	fmt.Fprintf(&sb, "- Resolution: %v after start\n", it.MTTR().Round(time.Second))

	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// MTTR STATISTICS
// ─────────────────────────────────────────────────────────────────────────────

type MTTRStats struct {
	Incidents []time.Duration
}

func (s *MTTRStats) Add(d time.Duration) { s.Incidents = append(s.Incidents, d) }

func (s *MTTRStats) Average() time.Duration {
	if len(s.Incidents) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range s.Incidents {
		total += d
	}
	return total / time.Duration(len(s.Incidents))
}

func (s *MTTRStats) P90() time.Duration {
	n := len(s.Incidents)
	if n == 0 {
		return 0
	}
	idx := int(float64(n) * 0.9)
	if idx >= n {
		idx = n - 1
	}
	// Simplified: find the value at p90 index (not a proper sort)
	max := s.Incidents[0]
	for _, d := range s.Incidents {
		if d > max {
			max = d
		}
	}
	return max
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 98 Exercise: Incident Timeline & Debugging Toolkit ===")
	fmt.Println()

	// ── HEALTH SNAPSHOT ───────────────────────────────────────────────────────
	fmt.Println("--- Health snapshot at T=0 ---")
	snap := TakeSnapshot()
	snap.Print()
	fmt.Println()

	// ── INCIDENT TIMELINE ─────────────────────────────────────────────────────
	fmt.Println("--- Incident timeline simulation ---")
	base := time.Now().Add(-37 * time.Minute)
	it := &IncidentTimeline{
		ID:        "INC-2025-0042",
		Title:     "checkout-service: elevated error rate (5xx)",
		StartTime: base,
	}

	it.AddAt(base, Info, "monitoring", "Error rate crossed 1% threshold")
	it.AddAt(base.Add(3*time.Minute), Warning, "monitoring", "Error rate at 8%; alert firing")
	it.AddAt(base.Add(7*time.Minute), Critical, "oncall", "Oncall paged (SEV-2)")
	it.AddAt(base.Add(10*time.Minute), Critical, "investigation", "Goroutine dump shows 2000+ blocked goroutines")
	it.AddAt(base.Add(15*time.Minute), Critical, "investigation", "Root cause: payment API timeout not set (default: infinite)")
	it.AddAt(base.Add(18*time.Minute), Warning, "mitigation", "Emergency config change: payment timeout=5s")
	it.AddAt(base.Add(22*time.Minute), Info, "mitigation", "Error rate dropping: 8% → 3% → 0.5%")
	it.AddAt(base.Add(30*time.Minute), Info, "recovery", "Goroutine count returning to baseline")
	it.EndTime = base.Add(37 * time.Minute)
	it.Events = append(it.Events, IncidentEvent{
		Time: base.Add(37 * time.Minute), Severity: Resolved,
		Source: "oncall", Message: "Error rate at baseline; incident resolved",
	})

	it.Print()
	fmt.Println()

	// ── INCIDENT REPORT ───────────────────────────────────────────────────────
	fmt.Println("--- Generated incident report ---")
	report := generateIncidentReport(it, snap)
	fmt.Println(report)

	// ── MTTR HISTORY ──────────────────────────────────────────────────────────
	fmt.Println("--- MTTR statistics (last 5 incidents) ---")
	stats := &MTTRStats{}
	for _, d := range []time.Duration{
		37 * time.Minute, 12 * time.Minute, 55 * time.Minute,
		8 * time.Minute, 28 * time.Minute,
	} {
		stats.Add(d)
	}
	fmt.Printf("  Incidents: %d\n", len(stats.Incidents))
	fmt.Printf("  Avg MTTR:  %v\n", stats.Average().Round(time.Minute))
	fmt.Printf("  P90 MTTR:  %v\n", stats.P90().Round(time.Minute))
	fmt.Println()

	// ── DEBUGGING CHECKLIST ───────────────────────────────────────────────────
	fmt.Println("--- Production debugging checklist ---")
	fmt.Println(`  1. Capture health baseline immediately:
       runtime.NumGoroutine(), runtime.ReadMemStats()
  2. Check recent deploys:
       git log --since="1 hour ago" --oneline
  3. Dump goroutines if count is elevated:
       curl /debug/pprof/goroutine?debug=2 | head -200
  4. Capture CPU/heap profile if CPU/memory is high:
       curl /debug/pprof/profile?seconds=30 > cpu.prof
  5. Check circuit breaker states and downstream error rates
  6. Check recent config changes (ConfigMap diff)
  7. Rollback if root cause not found within 15 minutes
  8. Write postmortem within 48 hours of resolution`)
}
