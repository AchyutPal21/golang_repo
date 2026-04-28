// EXERCISE 24.1 — Generic ordered set and priority queue.
//
// Implement OrderedSet[T comparable] with Add, Remove, Has, Len, Items.
// Items() returns elements in insertion order.
//
// Run (from the chapter folder):
//   go run ./exercises/01_collections

package main

import "fmt"

// OrderedSet maintains insertion order while preventing duplicates.
type OrderedSet[T comparable] struct {
	items []T
	index map[T]int // value → position in items
}

func NewOrderedSet[T comparable]() *OrderedSet[T] {
	return &OrderedSet[T]{index: make(map[T]int)}
}

func (s *OrderedSet[T]) Add(v T) bool {
	if _, ok := s.index[v]; ok {
		return false // already present
	}
	s.index[v] = len(s.items)
	s.items = append(s.items, v)
	return true
}

func (s *OrderedSet[T]) Has(v T) bool {
	_, ok := s.index[v]
	return ok
}

func (s *OrderedSet[T]) Remove(v T) bool {
	pos, ok := s.index[v]
	if !ok {
		return false
	}
	// Replace with last element to avoid O(n) shift.
	last := len(s.items) - 1
	if pos != last {
		moved := s.items[last]
		s.items[pos] = moved
		s.index[moved] = pos
	}
	s.items = s.items[:last]
	delete(s.index, v)
	return true
}

func (s *OrderedSet[T]) Len() int     { return len(s.items) }
func (s *OrderedSet[T]) Items() []T   {
	cp := make([]T, len(s.items))
	copy(cp, s.items)
	return cp
}

// Union returns a new set with all elements from both a and b.
func Union[T comparable](a, b *OrderedSet[T]) *OrderedSet[T] {
	result := NewOrderedSet[T]()
	for _, v := range a.Items() {
		result.Add(v)
	}
	for _, v := range b.Items() {
		result.Add(v)
	}
	return result
}

func main() {
	s := NewOrderedSet[string]()
	for _, w := range []string{"go", "python", "rust", "go", "java"} {
		added := s.Add(w)
		fmt.Printf("Add(%q): %v  len=%d\n", w, added, s.Len())
	}

	fmt.Println("Items:", s.Items())
	fmt.Println("Has rust:", s.Has("rust"))
	fmt.Println("Has ruby:", s.Has("ruby"))

	s.Remove("python")
	fmt.Println("After Remove(python):", s.Items())

	fmt.Println()

	a := NewOrderedSet[int]()
	b := NewOrderedSet[int]()
	for _, v := range []int{1, 2, 3} { a.Add(v) }
	for _, v := range []int{2, 3, 4} { b.Add(v) }
	u := Union(a, b)
	fmt.Println("Union:", u.Items())
}
