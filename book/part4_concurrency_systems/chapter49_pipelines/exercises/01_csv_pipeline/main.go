// FILE: book/part4_concurrency_systems/chapter49_pipelines/exercises/01_csv_pipeline/main.go
// CHAPTER: 49 — Pipelines, Fan-In/Out
// EXERCISE: CSV processing pipeline — parse stage → validate stage →
//           enrich stage (parallel) → aggregate stage → report.
//           Demonstrates a real-world multi-stage pipeline with error
//           propagation, fan-out for the slow enrich step, and ordered
//           fan-in to preserve row order.
//
// Run (from the chapter folder):
//   go run ./exercises/01_csv_pipeline

package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type RawRow struct {
	Index int
	Line  string
}

type ParsedRow struct {
	Index    int
	ID       int
	Name     string
	Amount   float64
	Category string
}

type ValidatedRow struct {
	ParsedRow
	Valid bool
	Err   error
}

type EnrichedRow struct {
	ValidatedRow
	Tax      float64
	NetTotal float64
}

type Summary struct {
	TotalRows    int
	ValidRows    int
	InvalidRows  int
	TotalAmount  float64
	TotalTax     float64
	ByCategory   map[string]float64
	Duration     time.Duration
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 1: source — emit raw CSV lines
// ─────────────────────────────────────────────────────────────────────────────

var csvData = []string{
	"1,Alice,1500.00,electronics",
	"2,Bob,250.50,clothing",
	"3,Carol,-50.00,clothing",   // invalid: negative amount
	"4,Dave,999.99,electronics",
	"5,Eve,INVALID,food",        // amount parses to 0 — valid but $0 transaction
	"6,Frank,750.00,food",
	"7,Grace,3200.00,electronics",
	"8,Hank,120.00,clothing",
	"9,Iris,88.00,food",
	"10,Jack,450.00,electronics",
	"11,Kim,310.00,",            // invalid: empty category
	"12,Liam,600.00,food",
}

func sourceStage(ctx context.Context) <-chan RawRow {
	out := make(chan RawRow)
	go func() {
		defer close(out)
		for i, line := range csvData {
			select {
			case out <- RawRow{Index: i, Line: line}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 2: parse — split CSV fields
// ─────────────────────────────────────────────────────────────────────────────

func parseStage(ctx context.Context, in <-chan RawRow) <-chan ParsedRow {
	out := make(chan ParsedRow)
	go func() {
		defer close(out)
		for {
			select {
			case row, ok := <-in:
				if !ok {
					return
				}
				parts := strings.Split(row.Line, ",")
				id, _ := strconv.Atoi(parts[0])
				amount, _ := strconv.ParseFloat(parts[2], 64)
				category := ""
				if len(parts) > 3 {
					category = parts[3]
				}
				p := ParsedRow{
					Index:    row.Index,
					ID:       id,
					Name:     parts[1],
					Amount:   amount,
					Category: category,
				}
				select {
				case out <- p:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 3: validate — check business rules
// ─────────────────────────────────────────────────────────────────────────────

func validateStage(ctx context.Context, in <-chan ParsedRow) <-chan ValidatedRow {
	out := make(chan ValidatedRow)
	go func() {
		defer close(out)
		for {
			select {
			case row, ok := <-in:
				if !ok {
					return
				}
				v := ValidatedRow{ParsedRow: row, Valid: true}
				switch {
				case row.Amount < 0:
					v.Valid = false
					v.Err = fmt.Errorf("row %d: negative amount %.2f", row.ID, row.Amount)
				case row.Category == "":
					v.Valid = false
					v.Err = fmt.Errorf("row %d: missing category", row.ID)
				case row.ID == 0:
					v.Valid = false
					v.Err = fmt.Errorf("row %d (index %d): parse error", row.ID, row.Index)
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 4: enrich — compute tax (fan-out for slow I/O simulation)
// ─────────────────────────────────────────────────────────────────────────────

var taxRates = map[string]float64{
	"electronics": 0.12,
	"clothing":    0.08,
	"food":        0.05,
}

func enrichWorker(ctx context.Context, in <-chan ValidatedRow, out chan<- EnrichedRow) {
	for {
		select {
		case row, ok := <-in:
			if !ok {
				return
			}
			e := EnrichedRow{ValidatedRow: row}
			if row.Valid {
				rate := taxRates[row.Category]
				e.Tax = row.Amount * rate
				e.NetTotal = row.Amount + e.Tax
				// Simulate slow external tax-lookup API.
				select {
				case <-time.After(10 * time.Millisecond):
				case <-ctx.Done():
					return
				}
			}
			select {
			case out <- e:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func enrichStage(ctx context.Context, in <-chan ValidatedRow, workers int) <-chan EnrichedRow {
	out := make(chan EnrichedRow, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enrichWorker(ctx, in, out)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 5: aggregate — collect results into Summary
// ─────────────────────────────────────────────────────────────────────────────

func aggregateStage(ctx context.Context, in <-chan EnrichedRow) Summary {
	s := Summary{ByCategory: make(map[string]float64)}
	for {
		select {
		case row, ok := <-in:
			if !ok {
				return s
			}
			s.TotalRows++
			if row.Valid {
				s.ValidRows++
				s.TotalAmount += row.Amount
				s.TotalTax += row.Tax
				s.ByCategory[row.Category] += row.NetTotal
			} else {
				s.InvalidRows++
				fmt.Printf("  [invalid] %v\n", row.Err)
			}
		case <-ctx.Done():
			return s
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== CSV Pipeline: parse → validate → enrich (fan-out) → aggregate ===")
	fmt.Println()

	ctx := context.Background()
	start := time.Now()

	raw := sourceStage(ctx)
	parsed := parseStage(ctx, raw)
	validated := validateStage(ctx, parsed)
	enriched := enrichStage(ctx, validated, 4) // 4 parallel enrich workers
	summary := aggregateStage(ctx, enriched)

	summary.Duration = time.Since(start)

	fmt.Println()
	fmt.Printf("  total rows  : %d\n", summary.TotalRows)
	fmt.Printf("  valid rows  : %d\n", summary.ValidRows)
	fmt.Printf("  invalid rows: %d\n", summary.InvalidRows)
	fmt.Printf("  total amount: $%.2f\n", summary.TotalAmount)
	fmt.Printf("  total tax   : $%.2f\n", summary.TotalTax)
	fmt.Printf("  duration    : %s\n", summary.Duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("  Revenue by category:")

	// Sort categories for deterministic output.
	cats := make([]string, 0, len(summary.ByCategory))
	for c := range summary.ByCategory {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, c := range cats {
		fmt.Printf("    %-15s $%.2f\n", c, summary.ByCategory[c])
	}
}
