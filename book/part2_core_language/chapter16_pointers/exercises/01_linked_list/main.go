// EXERCISE 16.1 — Singly linked list using pointers.
//
// Implement a singly linked list with Push, Pop, Len, and String.
// The list must work correctly when empty (nil head).
//
// Run (from the chapter folder):
//   go run ./exercises/01_linked_list

package main

import (
	"fmt"
	"strings"
)

type Node struct {
	Value int
	Next  *Node
}

type List struct {
	head *Node
	len  int
}

// Push adds v to the front of the list.
func (l *List) Push(v int) {
	l.head = &Node{Value: v, Next: l.head}
	l.len++
}

// Pop removes and returns the front value.
// Returns 0, false if the list is empty.
func (l *List) Pop() (int, bool) {
	if l.head == nil {
		return 0, false
	}
	v := l.head.Value
	l.head = l.head.Next
	l.len--
	return v, true
}

// Len returns the number of elements.
func (l *List) Len() int { return l.len }

// String renders the list as "[ 3 2 1 ]".
func (l *List) String() string {
	if l.head == nil {
		return "[ ]"
	}
	var parts []string
	for n := l.head; n != nil; n = n.Next {
		parts = append(parts, fmt.Sprintf("%d", n.Value))
	}
	return "[ " + strings.Join(parts, " → ") + " ]"
}

// Contains reports whether v is in the list.
func (l *List) Contains(v int) bool {
	for n := l.head; n != nil; n = n.Next {
		if n.Value == v {
			return true
		}
	}
	return false
}

func main() {
	var l List

	fmt.Println("empty list:", &l, "len:", l.Len())

	for _, v := range []int{1, 2, 3, 4, 5} {
		l.Push(v)
	}
	fmt.Println("after Push 1-5:", &l, "len:", l.Len())

	fmt.Println("contains 3:", l.Contains(3))
	fmt.Println("contains 9:", l.Contains(9))

	for {
		v, ok := l.Pop()
		if !ok {
			break
		}
		fmt.Printf("Pop: %d  (remaining: %s)\n", v, &l)
	}

	fmt.Println("empty after pops:", &l, "len:", l.Len())

	// Pop from empty list is safe
	_, ok := l.Pop()
	fmt.Println("Pop from empty: ok =", ok)
}
