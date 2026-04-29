// EXERCISE 33.1 — Build a data processing pipeline using Strategy + Iterator.
//
// A Pipeline takes an iterator source and a chain of Processor strategies.
// Data flows through each processor in order. The pipeline runs lazily:
// it only advances the iterator when asked for the next result.
//
// Run (from the chapter folder):
//   go run ./exercises/01_pipeline

package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ─── Iterator source ──────────────────────────────────────────────────────────

type Iter func() (string, bool)

func SliceIter(items []string) Iter {
	i := 0
	return func() (string, bool) {
		if i >= len(items) {
			return "", false
		}
		v := items[i]
		i++
		return v, true
	}
}

// ─── Processor strategy ───────────────────────────────────────────────────────

type Processor interface {
	Process(v string) (string, bool) // bool = keep (true) or filter (false)
	Name() string
}

// TrimProcessor strips leading/trailing whitespace.
type TrimProcessor struct{}

func (TrimProcessor) Name() string { return "trim" }
func (TrimProcessor) Process(v string) (string, bool) {
	return strings.TrimSpace(v), true
}

// UpperProcessor converts to uppercase.
type UpperProcessor struct{}

func (UpperProcessor) Name() string { return "upper" }
func (UpperProcessor) Process(v string) (string, bool) {
	return strings.ToUpper(v), true
}

// FilterEmptyProcessor drops empty strings.
type FilterEmptyProcessor struct{}

func (FilterEmptyProcessor) Name() string { return "filter-empty" }
func (FilterEmptyProcessor) Process(v string) (string, bool) {
	return v, v != ""
}

// PrefixProcessor prepends a fixed prefix.
type PrefixProcessor struct{ prefix string }

func (p PrefixProcessor) Name() string { return "prefix:" + p.prefix }
func (p PrefixProcessor) Process(v string) (string, bool) {
	return p.prefix + v, true
}

// LengthFilterProcessor drops strings shorter than Min.
type LengthFilterProcessor struct{ Min int }

func (f LengthFilterProcessor) Name() string { return "len>=" + strconv.Itoa(f.Min) }
func (f LengthFilterProcessor) Process(v string) (string, bool) {
	return v, len(v) >= f.Min
}

// ─── Pipeline ─────────────────────────────────────────────────────────────────

type Pipeline struct {
	source     Iter
	processors []Processor
}

func NewPipeline(source Iter, processors ...Processor) *Pipeline {
	return &Pipeline{source: source, processors: processors}
}

// Next pulls the next value through all processors. Skips filtered values.
func (p *Pipeline) Next() (string, bool) {
	for {
		v, ok := p.source()
		if !ok {
			return "", false
		}
		keep := true
		for _, proc := range p.processors {
			v, keep = proc.Process(v)
			if !keep {
				break
			}
		}
		if keep {
			return v, true
		}
	}
}

// Collect drains the pipeline into a slice.
func (p *Pipeline) Collect() []string {
	var results []string
	for v, ok := p.Next(); ok; v, ok = p.Next() {
		results = append(results, v)
	}
	return results
}

func (p *Pipeline) String() string {
	names := make([]string, len(p.processors))
	for i, proc := range p.processors {
		names[i] = proc.Name()
	}
	return strings.Join(names, " → ")
}

func main() {
	data := []string{
		"  hello  ",
		"",
		"  world  ",
		"  go  ",
		"  programming  ",
		"  ",
		"  language  ",
	}

	fmt.Println("=== Pipeline: trim → filter-empty → upper ===")
	p1 := NewPipeline(SliceIter(data),
		TrimProcessor{},
		FilterEmptyProcessor{},
		UpperProcessor{},
	)
	fmt.Println("  processors:", p1)
	results := p1.Collect()
	fmt.Println("  results:", results)

	fmt.Println()
	fmt.Println("=== Pipeline: trim → filter-empty → len>=4 → prefix:[item] ===")
	p2 := NewPipeline(SliceIter(data),
		TrimProcessor{},
		FilterEmptyProcessor{},
		LengthFilterProcessor{Min: 4},
		PrefixProcessor{"[item] "},
	)
	fmt.Println("  processors:", p2)
	results2 := p2.Collect()
	for _, r := range results2 {
		fmt.Printf("  → %s\n", r)
	}

	fmt.Println()
	fmt.Println("=== Pipeline: lazy — pull one at a time ===")
	p3 := NewPipeline(SliceIter(data),
		TrimProcessor{},
		FilterEmptyProcessor{},
	)
	fmt.Println("  pulling first two items:")
	if v, ok := p3.Next(); ok {
		fmt.Printf("  item 1: %q\n", v)
	}
	if v, ok := p3.Next(); ok {
		fmt.Printf("  item 2: %q\n", v)
	}
	fmt.Printf("  rest: %v\n", p3.Collect())
}
