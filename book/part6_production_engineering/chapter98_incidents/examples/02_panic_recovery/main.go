// FILE: book/part6_production_engineering/chapter98_incidents/examples/02_panic_recovery/main.go
// CHAPTER: 98 — Incident Management & Debugging
// TOPIC: Panic recovery, structured incident records, and postmortem patterns.
//
// Run:
//   go run ./book/part6_production_engineering/chapter98_incidents/examples/02_panic_recovery

package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PANIC RECOVERY
// ─────────────────────────────────────────────────────────────────────────────

type PanicRecord struct {
	Timestamp time.Time
	PanicVal  any
	Stack     string
	Context   map[string]string
}

var (
	panicLog   []PanicRecord
	panicLogMu sync.Mutex
)

func withRecovery(ctx map[string]string, fn func()) (rec *PanicRecord) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 64*1024)
			n := runtime.Stack(buf, false)
			record := PanicRecord{
				Timestamp: time.Now(),
				PanicVal:  r,
				Stack:     string(buf[:n]),
				Context:   ctx,
			}
			panicLogMu.Lock()
			panicLog = append(panicLog, record)
			panicLogMu.Unlock()
			rec = &record
		}
	}()
	fn()
	return nil
}

func printPanicRecord(r *PanicRecord) {
	fmt.Printf("  Panic at: %s\n", r.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Value: %v\n", r.PanicVal)
	for k, v := range r.Context {
		fmt.Printf("  Context.%s: %s\n", k, v)
	}
	// Print first 5 stack lines only
	lines := strings.Split(r.Stack, "\n")
	fmt.Println("  Stack (truncated):")
	for i, l := range lines {
		if i >= 8 {
			fmt.Printf("    ... (%d more lines)\n", len(lines)-8)
			break
		}
		fmt.Printf("    %s\n", l)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// POSTMORTEM TEMPLATE
// ─────────────────────────────────────────────────────────────────────────────

type Severity int

const (
	SEV1 Severity = 1
	SEV2 Severity = 2
	SEV3 Severity = 3
	SEV4 Severity = 4
)

func (s Severity) String() string {
	return fmt.Sprintf("SEV-%d", int(s))
}

type TimelineEvent struct {
	Time    time.Time
	Summary string
}

type Postmortem struct {
	Title       string
	Severity    Severity
	StartTime   time.Time
	EndTime     time.Time
	Impact      string
	Timeline    []TimelineEvent
	RootCause   string
	WhyChain    []string
	ActionItems []struct {
		Owner string
		Task  string
		Due   time.Time
	}
	Lessons []string
}

func (pm *Postmortem) MTTR() time.Duration {
	return pm.EndTime.Sub(pm.StartTime)
}

func (pm *Postmortem) Render() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Postmortem: %s\n\n", pm.Title)
	fmt.Fprintf(&sb, "**Severity:** %s  **Duration:** %v\n", pm.Severity, pm.MTTR().Round(time.Minute))
	fmt.Fprintf(&sb, "**Impact:** %s\n\n", pm.Impact)

	fmt.Fprintf(&sb, "## Timeline\n")
	for _, e := range pm.Timeline {
		fmt.Fprintf(&sb, "- `%s` %s\n", e.Time.Format("15:04"), e.Summary)
	}

	fmt.Fprintf(&sb, "\n## Root Cause\n%s\n\n", pm.RootCause)

	fmt.Fprintf(&sb, "## 5 Whys\n")
	for i, why := range pm.WhyChain {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, why)
	}

	fmt.Fprintf(&sb, "\n## Action Items\n")
	for _, a := range pm.ActionItems {
		fmt.Fprintf(&sb, "- [ ] **%s** — %s (due %s)\n", a.Owner, a.Task, a.Due.Format("2006-01-02"))
	}

	fmt.Fprintf(&sb, "\n## Lessons Learned\n")
	for _, l := range pm.Lessons {
		fmt.Fprintf(&sb, "- %s\n", l)
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 98: Panic Recovery & Postmortem ===")
	fmt.Println()

	// ── PANIC RECOVERY DEMO ───────────────────────────────────────────────────
	fmt.Println("--- Panic recovery ---")

	// Controlled panic
	ctx := map[string]string{
		"request_id": "req-abc123",
		"user_id":    "42",
		"path":       "/api/orders/99",
	}
	record := withRecovery(ctx, func() {
		var m map[string]string
		_ = m["key"] // nil map dereference → panic
	})
	if record != nil {
		fmt.Println("  Panic caught:")
		printPanicRecord(record)
	}
	fmt.Println()

	// String panic
	record2 := withRecovery(map[string]string{"component": "payment"}, func() {
		panic("payment gateway unreachable")
	})
	if record2 != nil {
		fmt.Printf("  Caught string panic: %q\n", record2.PanicVal)
	}
	fmt.Println()

	// ── DEFER + RECOVER RULES ─────────────────────────────────────────────────
	fmt.Println("--- defer + recover rules ---")
	fmt.Println("  Rule 1: recover() must be called DIRECTLY in a defer function")
	fmt.Println("    Works:   defer func() { r := recover(); ... }()")
	fmt.Println("    Fails:   defer func() { helper() }()  // helper()'s recover sees no panic")
	fmt.Println()
	fmt.Println("  Rule 2: recover() only intercepts panics from the same goroutine")
	fmt.Println("    Each goroutine needs its own defer/recover.")
	fmt.Println()
	fmt.Println("  Rule 3: Don't silently swallow panics")
	fmt.Println("    Always log: panic value + stack + request context")
	fmt.Println("    Report to error tracking (Sentry, Honeybadger, Rollbar)")
	fmt.Println()
	fmt.Println("  Rule 4: HTTP middleware pattern")
	fmt.Println("    defer func() {")
	fmt.Println("      if rec := recover(); rec != nil {")
	fmt.Println("        stack := make([]byte, 64*1024)")
	fmt.Println("        n := runtime.Stack(stack, false)")
	fmt.Println("        log.Errorf(panic: value+stack, rec, stack[:n])")
	fmt.Println("        http.Error(w, \"internal server error\", 500)")
	fmt.Println("      }")
	fmt.Println("    }()")
	fmt.Println()

	// ── POSTMORTEM ────────────────────────────────────────────────────────────
	fmt.Println("--- Sample postmortem ---")
	base := time.Date(2025, 5, 1, 14, 23, 0, 0, time.UTC)
	pm := Postmortem{
		Title:    "Checkout service outage — nil pointer in payment handler",
		Severity: SEV2,
		StartTime: base,
		EndTime:   base.Add(37 * time.Minute),
		Impact:   "Checkout unavailable for 37 minutes; ~2,400 failed transactions",
		Timeline: []TimelineEvent{
			{base, "First 500 errors detected by uptime monitor"},
			{base.Add(5 * time.Minute), "Alert fires; oncall paged"},
			{base.Add(12 * time.Minute), "Root cause identified: nil pointer in payment handler"},
			{base.Add(20 * time.Minute), "Rollback initiated"},
			{base.Add(37 * time.Minute), "Service restored; error rate back to baseline"},
		},
		RootCause: "Payment gateway config was nil when feature flag was enabled in prod before config was deployed.",
		WhyChain: []string{
			"Checkout service panicked on nil pointer dereference",
			"PaymentGatewayConfig was nil at call site",
			"Feature flag enabled gateway before config ConfigMap was applied",
			"Deploy pipeline applied code before config (ordering not enforced)",
			"No process requires config before code deploys; no nil check at startup",
		},
		ActionItems: []struct {
			Owner string
			Task  string
			Due   time.Time
		}{
			{"platform-team", "Enforce config-before-code ordering in deploy pipeline", base.Add(7 * 24 * time.Hour)},
			{"checkout-team", "Add startup validation: panic if required config is nil", base.Add(3 * 24 * time.Hour)},
			{"oncall", "Add alert for nil pointer panics (check pprof panic count)", base.Add(5 * 24 * time.Hour)},
		},
		Lessons: []string{
			"Feature flags should validate dependencies are configured before enabling",
			"Config and code deploys must be ordered or validated atomically",
			"Startup validation catches missing config before first request",
		},
	}
	fmt.Println(pm.Render())

	fmt.Printf("  Total panics caught in this demo: %d\n", len(panicLog))
}
