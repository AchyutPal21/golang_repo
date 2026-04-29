// EXERCISE 29.1 — Identify and fix SOLID violations.
//
// The code below contains violations of all five SOLID principles.
// Read the violation comments, then apply the fixes shown beneath each one.
//
// Run (from the chapter folder):
//   go run ./exercises/01_refactor_violations

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// VIOLATION 1 — SRP
//
// ReportService handles fetching, formatting, AND emailing reports.
// Any change to formatting or email breaks the same struct.
//
// FIX: Split into ReportFetcher, ReportFormatter, ReportEmailer.
//      A thin ReportPipeline coordinator wires them together.
// ─────────────────────────────────────────────────────────────────────────────

type ReportRow struct {
	Name  string
	Value float64
}

// ── After fix: three focused types ───────────────────────────────────────────

type ReportFetcher struct{ rows []ReportRow }

func (f *ReportFetcher) Fetch() []ReportRow { return f.rows }

type TextFormatter struct{}

func (t TextFormatter) Format(rows []ReportRow) string {
	var sb strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&sb, "  %-20s %.2f\n", r.Name, r.Value)
	}
	return sb.String()
}

type ReportEmailer struct{ from string }

func (e *ReportEmailer) Email(to, body string) {
	fmt.Printf("[EMAIL] from=%s to=%s\n%s", e.from, to, body)
}

type ReportPipeline struct {
	fetcher   *ReportFetcher
	formatter TextFormatter
	emailer   *ReportEmailer
}

func (p *ReportPipeline) Run(recipient string) {
	rows := p.fetcher.Fetch()
	body := p.formatter.Format(rows)
	p.emailer.Email(recipient, body)
}

// ─────────────────────────────────────────────────────────────────────────────
// VIOLATION 2 — OCP
//
// exportReport uses a switch on format string.
// Adding "xml" or "csv" requires editing this function.
//
// FIX: Define an Exporter interface. Each format is a new type.
// ─────────────────────────────────────────────────────────────────────────────

type Exporter interface {
	Export(rows []ReportRow) string
	ContentType() string
}

type JSONExporter struct{}

func (j JSONExporter) Export(rows []ReportRow) string {
	var parts []string
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf(`{"name":%q,"value":%.2f}`, r.Name, r.Value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
func (j JSONExporter) ContentType() string { return "application/json" }

type CSVExporter struct{}

func (c CSVExporter) Export(rows []ReportRow) string {
	var sb strings.Builder
	sb.WriteString("name,value\n")
	for _, r := range rows {
		fmt.Fprintf(&sb, "%s,%.2f\n", r.Name, r.Value)
	}
	return sb.String()
}
func (c CSVExporter) ContentType() string { return "text/csv" }

func exportReport(rows []ReportRow, e Exporter) {
	fmt.Printf("[EXPORT] content-type=%s\n%s\n", e.ContentType(), e.Export(rows))
}

// ─────────────────────────────────────────────────────────────────────────────
// VIOLATION 3 — LSP
//
// SilentLogger claims to implement Logger but discards messages silently.
// Callers that depend on Logger contract (messages are recorded) are surprised.
//
// FIX: Create a NullLogger that explicitly documents it discards output,
//      or give Logger a narrower contract that SilentLogger can honour.
// ─────────────────────────────────────────────────────────────────────────────

type Logger interface {
	Log(msg string)
	Entries() []string // contract: every Log call appends one entry
}

type stdoutLogger struct{ entries []string }

func (l *stdoutLogger) Log(msg string) {
	l.entries = append(l.entries, msg)
	fmt.Println("  [LOG]", msg)
}
func (l *stdoutLogger) Entries() []string { return l.entries }

// NullLogger honours the contract — Entries returns the accumulated slice.
type NullLogger struct{ entries []string }

func (n *NullLogger) Log(msg string)      { n.entries = append(n.entries, msg) }
func (n *NullLogger) Entries() []string   { return n.entries }

// ─────────────────────────────────────────────────────────────────────────────
// VIOLATION 4 — ISP
//
// JobWorker is forced to implement Pause/Resume even though it is a
// one-shot job that cannot be paused.
//
// FIX: Split into Runnable (one-shot) and Pausable (interruptible).
//      Consumers ask for the slice they actually use.
// ─────────────────────────────────────────────────────────────────────────────

type Runnable interface {
	Run() error
}

type Pausable interface {
	Pause()
	Resume()
}

// BatchJob is a one-shot — satisfies Runnable only.
type BatchJob struct{ name string }

func (b *BatchJob) Run() error {
	fmt.Printf("  [JOB] %s started at %s\n", b.name, time.Now().Format("15:04:05"))
	return nil
}

// StreamingJob can be paused — satisfies both Runnable and Pausable.
type StreamingJob struct {
	name   string
	paused bool
}

func (s *StreamingJob) Run() error   { fmt.Printf("  [STREAM] %s running\n", s.name); return nil }
func (s *StreamingJob) Pause()       { s.paused = true; fmt.Printf("  [STREAM] %s paused\n", s.name) }
func (s *StreamingJob) Resume()      { s.paused = false; fmt.Printf("  [STREAM] %s resumed\n", s.name) }

func startJob(r Runnable) {
	if err := r.Run(); err != nil {
		fmt.Println("  job error:", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VIOLATION 5 — DIP
//
// AnalyticsService directly constructs a MySQLStore.
// High-level logic depends on low-level detail.
//
// FIX: AnalyticsService receives an EventStore interface via constructor.
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	Name string
	At   time.Time
}

type EventStore interface {
	Record(e Event) error
	Count() int
}

type inMemEventStore struct{ events []Event }

func (s *inMemEventStore) Record(e Event) error { s.events = append(s.events, e); return nil }
func (s *inMemEventStore) Count() int           { return len(s.events) }

type AnalyticsService struct{ store EventStore }

func NewAnalyticsService(s EventStore) *AnalyticsService { return &AnalyticsService{store: s} }

func (a *AnalyticsService) Track(name string) {
	e := Event{Name: name, At: time.Now()}
	if err := a.store.Record(e); err != nil {
		fmt.Println("  track error:", err)
	}
	fmt.Printf("  [ANALYTICS] tracked %q (total: %d)\n", name, a.store.Count())
}

func main() {
	rows := []ReportRow{{"Revenue", 12345.67}, {"Expenses", 9876.54}, {"Net", 2469.13}}

	fmt.Println("=== SRP fix: pipeline ===")
	pipeline := &ReportPipeline{
		fetcher:   &ReportFetcher{rows: rows},
		formatter: TextFormatter{},
		emailer:   &ReportEmailer{from: "reports@example.com"},
	}
	pipeline.Run("cfo@example.com")

	fmt.Println()
	fmt.Println("=== OCP fix: exporters ===")
	exportReport(rows, JSONExporter{})
	exportReport(rows, CSVExporter{})

	fmt.Println()
	fmt.Println("=== LSP fix: NullLogger honours Entries() contract ===")
	var log Logger = &NullLogger{}
	log.Log("startup")
	log.Log("ready")
	fmt.Printf("  null logger recorded %d entries\n", len(log.Entries()))

	log2 := &stdoutLogger{}
	log2.Log("hello")
	fmt.Printf("  stdout logger recorded %d entries\n", len(log2.Entries()))

	fmt.Println()
	fmt.Println("=== ISP fix: Runnable vs Pausable ===")
	batch := &BatchJob{name: "nightly-export"}
	stream := &StreamingJob{name: "live-feed"}
	startJob(batch)
	startJob(stream)
	stream.Pause()
	stream.Resume()

	fmt.Println()
	fmt.Println("=== DIP fix: AnalyticsService receives EventStore ===")
	svc := NewAnalyticsService(&inMemEventStore{})
	svc.Track("page_view")
	svc.Track("purchase")
	svc.Track("page_view")
}
