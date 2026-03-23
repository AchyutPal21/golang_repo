// FILE: 05_collections/08_collections_patterns.go
// TOPIC: Collection Patterns — stack, queue, set ops, dedup, partition, groupBy
//
// Run: go run 05_collections/08_collections_patterns.go

package main

import "fmt"

// ── STACK (LIFO) using slice ──────────────────────────────────────────────────
type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T)        { s.items = append(s.items, v) }
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}
func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}
func (s *Stack[T]) Len() int { return len(s.items) }

// ── QUEUE (FIFO) using slice ──────────────────────────────────────────────────
type Queue[T any] struct {
	items []T
}

func (q *Queue[T]) Enqueue(v T)      { q.items = append(q.items, v) }
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	front := q.items[0]
	q.items = q.items[1:]
	return front, true
}
func (q *Queue[T]) Len() int { return len(q.items) }

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Collection Patterns")
	fmt.Println("════════════════════════════════════════")

	// ── Stack ──────────────────────────────────────────────────────────
	fmt.Println("\n── Stack (LIFO) ──")
	s := &Stack[int]{}
	for _, v := range []int{1, 2, 3, 4, 5} {
		s.Push(v)
	}
	fmt.Printf("  Pushed: [1 2 3 4 5], len=%d\n", s.Len())
	for s.Len() > 0 {
		v, _ := s.Pop()
		fmt.Printf("  Pop: %d\n", v)
	}

	// ── Queue ──────────────────────────────────────────────────────────
	fmt.Println("\n── Queue (FIFO) ──")
	q := &Queue[string]{}
	for _, v := range []string{"first", "second", "third"} {
		q.Enqueue(v)
	}
	for q.Len() > 0 {
		v, _ := q.Dequeue()
		fmt.Printf("  Dequeue: %q\n", v)
	}

	// ── SET OPERATIONS ─────────────────────────────────────────────────
	fmt.Println("\n── Set operations ──")
	toSet := func(s []int) map[int]struct{} {
		m := make(map[int]struct{}, len(s))
		for _, v := range s {
			m[v] = struct{}{}
		}
		return m
	}
	toSlice := func(m map[int]struct{}) []int {
		out := make([]int, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}

	a := []int{1, 2, 3, 4, 5}
	b := []int{3, 4, 5, 6, 7}
	sa, sb := toSet(a), toSet(b)

	// Union
	union := make(map[int]struct{})
	for k := range sa { union[k] = struct{}{} }
	for k := range sb { union[k] = struct{}{} }
	fmt.Printf("  Union:        %v\n", toSlice(union))

	// Intersection
	inter := make(map[int]struct{})
	for k := range sa {
		if _, ok := sb[k]; ok {
			inter[k] = struct{}{}
		}
	}
	fmt.Printf("  Intersection: %v\n", toSlice(inter))

	// Difference (a - b)
	diff := make(map[int]struct{})
	for k := range sa {
		if _, ok := sb[k]; !ok {
			diff[k] = struct{}{}
		}
	}
	fmt.Printf("  Difference:   %v\n", toSlice(diff))

	// ── DEDUPLICATION ─────────────────────────────────────────────────
	fmt.Println("\n── Deduplication ──")
	dupes := []string{"go", "rust", "go", "python", "rust", "go"}
	seen := make(map[string]struct{})
	var unique []string
	for _, v := range dupes {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			unique = append(unique, v)
		}
	}
	fmt.Printf("  Input:  %v\n  Output: %v\n", dupes, unique)

	// ── PARTITION ─────────────────────────────────────────────────────
	fmt.Println("\n── Partition by predicate ──")
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var evens, odds []int
	for _, n := range nums {
		if n%2 == 0 {
			evens = append(evens, n)
		} else {
			odds = append(odds, n)
		}
	}
	fmt.Printf("  Evens: %v\n  Odds:  %v\n", evens, odds)

	// ── GROUP BY ──────────────────────────────────────────────────────
	fmt.Println("\n── GroupBy ──")
	type Item struct{ Name, Category string }
	items := []Item{
		{"apple", "fruit"}, {"banana", "fruit"},
		{"carrot", "vegetable"}, {"broccoli", "vegetable"},
		{"cherry", "fruit"},
	}
	grouped := make(map[string][]string)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item.Name)
	}
	for _, cat := range []string{"fruit", "vegetable"} {
		fmt.Printf("  %s: %v\n", cat, grouped[cat])
	}

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Stack: append to push, slice[:n-1] to pop")
	fmt.Println("  Queue: append to enqueue, slice[1:] to dequeue")
	fmt.Println("  Set ops: use map[T]struct{} for union/intersection/diff")
	fmt.Println("  Dedup: map to track seen items, preserve order")
	fmt.Println("  Partition/GroupBy: foundational slice+map patterns")
}
