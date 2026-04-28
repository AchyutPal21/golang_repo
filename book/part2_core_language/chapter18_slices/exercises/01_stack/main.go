// EXERCISE 18.1 — Generic stack backed by a slice.
//
// Implement Stack[T] with Push, Pop, Peek, Len, and IsEmpty.
// The stack must be safe to use as a zero value (no explicit constructor needed).
//
// Run (from the chapter folder):
//   go run ./exercises/01_stack

package main

import "fmt"

type Stack[T any] struct {
	items []T
}

// Push adds v to the top.
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop removes and returns the top element.
// Returns the zero value of T and false if empty.
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

// Peek returns the top element without removing it.
func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int     { return len(s.items) }
func (s *Stack[T]) IsEmpty() bool { return len(s.items) == 0 }

// balancedParens uses a stack to check balanced brackets.
func balancedParens(s string) bool {
	var stack Stack[rune]
	pair := map[rune]rune{')': '(', ']': '[', '}': '{'}

	for _, ch := range s {
		switch ch {
		case '(', '[', '{':
			stack.Push(ch)
		case ')', ']', '}':
			top, ok := stack.Pop()
			if !ok || top != pair[ch] {
				return false
			}
		}
	}
	return stack.IsEmpty()
}

func main() {
	var s Stack[int]
	fmt.Println("empty:", s.IsEmpty())

	for _, v := range []int{10, 20, 30} {
		s.Push(v)
	}
	fmt.Println("len:", s.Len())

	if top, ok := s.Peek(); ok {
		fmt.Println("peek:", top)
	}

	for {
		v, ok := s.Pop()
		if !ok {
			break
		}
		fmt.Printf("pop: %d  (len=%d)\n", v, s.Len())
	}

	fmt.Println()

	// String stack
	var ss Stack[string]
	ss.Push("a")
	ss.Push("b")
	ss.Push("c")
	v, _ := ss.Pop()
	fmt.Println("string pop:", v)

	fmt.Println()

	// Balanced parentheses
	cases := []string{
		"(a+b)*[c-d]",
		"(((",
		"{[()]}",
		"([)]",
	}
	for _, tc := range cases {
		fmt.Printf("balanced(%q): %v\n", tc, balancedParens(tc))
	}
}
