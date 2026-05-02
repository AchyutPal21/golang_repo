// EXERCISE 38.1 — Stream-process a log file without loading it into memory.
//
// LogProcessor reads a log stream line by line, filters by level,
// and writes matching lines to an output writer — all as an io pipeline.
//
// Run (from the chapter folder):
//   go run ./exercises/01_log_processor

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ─── Log entry ────────────────────────────────────────────────────────────────

type Level string

const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

func levelOrder(l Level) int {
	switch l {
	case DEBUG:
		return 0
	case INFO:
		return 1
	case WARN:
		return 2
	case ERROR:
		return 3
	default:
		return -1
	}
}

// parseLine extracts the level from a log line formatted as "LEVEL message".
func parseLine(line string) (Level, string, bool) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	lvl := Level(strings.ToUpper(parts[0]))
	if levelOrder(lvl) < 0 {
		return "", "", false
	}
	return lvl, parts[1], true
}

// ─── LogProcessor — streaming filter+transform pipeline ─────────────────────

type LogProcessor struct {
	minLevel Level
	stats    map[Level]int
}

func NewLogProcessor(minLevel Level) *LogProcessor {
	return &LogProcessor{
		minLevel: minLevel,
		stats:    make(map[Level]int),
	}
}

// Process reads from src, filters by minLevel, writes matches to dst.
// Returns (lines read, lines written, error).
func (p *LogProcessor) Process(src io.Reader, dst io.Writer) (read, written int, err error) {
	bw := bufio.NewWriter(dst)
	defer func() {
		if ferr := bw.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()
		read++

		lvl, msg, ok := parseLine(line)
		if !ok {
			continue
		}
		p.stats[lvl]++

		if levelOrder(lvl) >= levelOrder(p.minLevel) {
			if _, werr := fmt.Fprintf(bw, "[%s] %s\n", lvl, msg); werr != nil {
				return read, written, werr
			}
			written++
		}
	}
	if serr := scanner.Err(); serr != nil {
		return read, written, serr
	}
	return read, written, nil
}

func (p *LogProcessor) Stats() map[Level]int { return p.stats }

// ─── Multi-sink: write matching lines to multiple outputs ─────────────────────

func processToMultiple(src io.Reader, minLevel Level, sinks ...io.Writer) (read, written int, err error) {
	mw := io.MultiWriter(sinks...)
	proc := NewLogProcessor(minLevel)
	return proc.Process(src, mw)
}

func main() {
	logData := `INFO application started
DEBUG loading config from /etc/app.conf
INFO server listening on :8080
DEBUG request received GET /health
INFO health check passed
WARN disk usage at 75%
DEBUG request received POST /api/orders
ERROR failed to connect to database: connection refused
INFO retrying database connection
WARN database connection pool: 80% utilised
ERROR order processing failed: timeout after 30s
INFO graceful shutdown initiated
`

	fmt.Println("=== Filter INFO and above ===")
	src := strings.NewReader(logData)
	var out strings.Builder
	proc := NewLogProcessor(INFO)
	read, written, err := proc.Process(src, &out)
	fmt.Printf("  read=%d  written=%d  err=%v\n", read, written, err)
	fmt.Println("  stats:", proc.Stats())
	fmt.Println("  output:")
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		fmt.Println("   ", line)
	}

	fmt.Println()
	fmt.Println("=== Filter WARN and above ===")
	src2 := strings.NewReader(logData)
	var out2 strings.Builder
	proc2 := NewLogProcessor(WARN)
	r2, w2, _ := proc2.Process(src2, &out2)
	fmt.Printf("  read=%d  written=%d\n", r2, w2)
	fmt.Print("  output:\n")
	for _, line := range strings.Split(strings.TrimSpace(out2.String()), "\n") {
		fmt.Println("   ", line)
	}

	fmt.Println()
	fmt.Println("=== Multi-sink: stdout + audit log ===")
	src3 := strings.NewReader(logData)
	var stdoutSink strings.Builder
	var auditSink strings.Builder
	r3, w3, _ := processToMultiple(src3, ERROR, &stdoutSink, &auditSink)
	fmt.Printf("  read=%d  written=%d\n", r3, w3)
	fmt.Printf("  stdoutSink == auditSink: %v\n", stdoutSink.String() == auditSink.String())
	fmt.Println("  errors only:")
	for _, line := range strings.Split(strings.TrimSpace(stdoutSink.String()), "\n") {
		fmt.Println("   ", line)
	}
}
