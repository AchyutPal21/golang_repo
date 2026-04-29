// FILE: book/part3_designing_software/chapter33_behavioral_patterns/examples/02_command_iterator_state/main.go
// CHAPTER: 33 — Behavioral Patterns
// TOPIC: Command (undo-able operations), Iterator (sequential traversal),
//        and State (behaviour changes with internal state).
//
// Run (from the chapter folder):
//   go run ./examples/02_command_iterator_state

package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// COMMAND
//
// Encapsulates a request as an object. Enables undo/redo, queuing, logging.
// In Go: an interface with Execute() and Undo(), plus a history stack.
// ─────────────────────────────────────────────────────────────────────────────

type Command interface {
	Execute() error
	Undo() error
	Description() string
}

// Text document — the receiver.
type Document struct{ content strings.Builder }

func (d *Document) Insert(pos int, text string) {
	current := d.content.String()
	if pos < 0 || pos > len(current) {
		pos = len(current)
	}
	d.content.Reset()
	d.content.WriteString(current[:pos] + text + current[pos:])
}

func (d *Document) Delete(pos, length int) {
	current := d.content.String()
	if pos < 0 || pos >= len(current) {
		return
	}
	end := pos + length
	if end > len(current) {
		end = len(current)
	}
	d.content.Reset()
	d.content.WriteString(current[:pos] + current[end:])
}

func (d *Document) Text() string { return d.content.String() }

// InsertCommand — inserts text at a position.
type InsertCommand struct {
	doc  *Document
	pos  int
	text string
}

func (c *InsertCommand) Execute() error {
	c.doc.Insert(c.pos, c.text)
	return nil
}

func (c *InsertCommand) Undo() error {
	c.doc.Delete(c.pos, len(c.text))
	return nil
}

func (c *InsertCommand) Description() string {
	return fmt.Sprintf("insert %q at %d", c.text, c.pos)
}

// DeleteCommand — deletes a range.
type DeleteCommand struct {
	doc     *Document
	pos     int
	length  int
	deleted string // saved for undo
}

func (c *DeleteCommand) Execute() error {
	text := c.doc.Text()
	end := c.pos + c.length
	if end > len(text) {
		end = len(text)
	}
	c.deleted = text[c.pos:end]
	c.doc.Delete(c.pos, c.length)
	return nil
}

func (c *DeleteCommand) Undo() error {
	c.doc.Insert(c.pos, c.deleted)
	return nil
}

func (c *DeleteCommand) Description() string {
	return fmt.Sprintf("delete %d chars at %d", c.length, c.pos)
}

// CommandHistory — undo/redo stack.
type CommandHistory struct {
	done   []Command
	undone []Command
}

func (h *CommandHistory) Execute(cmd Command) error {
	if err := cmd.Execute(); err != nil {
		return err
	}
	h.done = append(h.done, cmd)
	h.undone = nil // clear redo stack
	return nil
}

func (h *CommandHistory) Undo() error {
	if len(h.done) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	cmd := h.done[len(h.done)-1]
	h.done = h.done[:len(h.done)-1]
	if err := cmd.Undo(); err != nil {
		return err
	}
	h.undone = append(h.undone, cmd)
	fmt.Printf("  [UNDO] %s\n", cmd.Description())
	return nil
}

func (h *CommandHistory) Redo() error {
	if len(h.undone) == 0 {
		return fmt.Errorf("nothing to redo")
	}
	cmd := h.undone[len(h.undone)-1]
	h.undone = h.undone[:len(h.undone)-1]
	if err := cmd.Execute(); err != nil {
		return err
	}
	h.done = append(h.done, cmd)
	fmt.Printf("  [REDO] %s\n", cmd.Description())
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ITERATOR
//
// Provides sequential access to elements without exposing the collection.
// In Go: a closure-based iterator or a struct with Next()/Value() methods.
// ─────────────────────────────────────────────────────────────────────────────

// Tree — a binary tree, traversed via an iterator.
type TreeNode struct {
	Value       int
	Left, Right *TreeNode
}

// InorderIterator — closure-based iterator over a BST.
func InorderIterator(root *TreeNode) func() (int, bool) {
	stack := []*TreeNode{}
	current := root
	return func() (int, bool) {
		for current != nil || len(stack) > 0 {
			for current != nil {
				stack = append(stack, current)
				current = current.Left
			}
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			current = node.Right
			return node.Value, true
		}
		return 0, false
	}
}

func insert(root *TreeNode, val int) *TreeNode {
	if root == nil {
		return &TreeNode{Value: val}
	}
	if val < root.Value {
		root.Left = insert(root.Left, val)
	} else {
		root.Right = insert(root.Right, val)
	}
	return root
}

// RangeIterator — iterates integers from start to end (exclusive).
func RangeIterator(start, end, step int) func() (int, bool) {
	current := start
	return func() (int, bool) {
		if current >= end {
			return 0, false
		}
		v := current
		current += step
		return v, true
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// STATE
//
// Allows an object to alter its behaviour when its internal state changes.
// In Go: a state interface with the allowed transitions; the context delegates
// all behaviour to the current state.
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus interface {
	Name() string
	Pay(o *Order) error
	Ship(o *Order) error
	Deliver(o *Order) error
	Cancel(o *Order) error
}

type Order struct {
	ID     string
	status OrderStatus
}

func NewOrder(id string) *Order {
	o := &Order{ID: id}
	o.status = &pendingState{o}
	return o
}

func (o *Order) Status() string     { return o.status.Name() }
func (o *Order) Pay() error         { return o.status.Pay(o) }
func (o *Order) Ship() error        { return o.status.Ship(o) }
func (o *Order) Deliver() error     { return o.status.Deliver(o) }
func (o *Order) Cancel() error      { return o.status.Cancel(o) }
func (o *Order) setState(s OrderStatus) { o.status = s }

// States.

type pendingState struct{ o *Order }

func (s *pendingState) Name() string { return "pending" }
func (s *pendingState) Pay(o *Order) error {
	fmt.Printf("  [%s] paid\n", o.ID)
	o.setState(&paidState{o})
	return nil
}
func (s *pendingState) Ship(_ *Order) error { return fmt.Errorf("must pay before shipping") }
func (s *pendingState) Deliver(_ *Order) error { return fmt.Errorf("must pay before delivery") }
func (s *pendingState) Cancel(o *Order) error {
	fmt.Printf("  [%s] cancelled (pending)\n", o.ID)
	o.setState(&cancelledState{})
	return nil
}

type paidState struct{ o *Order }

func (s *paidState) Name() string { return "paid" }
func (s *paidState) Pay(_ *Order) error { return fmt.Errorf("already paid") }
func (s *paidState) Ship(o *Order) error {
	fmt.Printf("  [%s] shipped\n", o.ID)
	o.setState(&shippedState{o})
	return nil
}
func (s *paidState) Deliver(_ *Order) error { return fmt.Errorf("must ship before delivery") }
func (s *paidState) Cancel(o *Order) error {
	fmt.Printf("  [%s] cancelled (paid — refunding)\n", o.ID)
	o.setState(&cancelledState{})
	return nil
}

type shippedState struct{ o *Order }

func (s *shippedState) Name() string { return "shipped" }
func (s *shippedState) Pay(_ *Order) error    { return fmt.Errorf("already paid") }
func (s *shippedState) Ship(_ *Order) error   { return fmt.Errorf("already shipped") }
func (s *shippedState) Deliver(o *Order) error {
	fmt.Printf("  [%s] delivered\n", o.ID)
	o.setState(&deliveredState{})
	return nil
}
func (s *shippedState) Cancel(_ *Order) error { return fmt.Errorf("cannot cancel shipped order") }

type deliveredState struct{}

func (deliveredState) Name() string          { return "delivered" }
func (deliveredState) Pay(_ *Order) error    { return fmt.Errorf("already paid") }
func (deliveredState) Ship(_ *Order) error   { return fmt.Errorf("already shipped") }
func (deliveredState) Deliver(_ *Order) error { return fmt.Errorf("already delivered") }
func (deliveredState) Cancel(_ *Order) error  { return fmt.Errorf("already delivered") }

type cancelledState struct{}

func (cancelledState) Name() string           { return "cancelled" }
func (cancelledState) Pay(_ *Order) error     { return fmt.Errorf("order cancelled") }
func (cancelledState) Ship(_ *Order) error    { return fmt.Errorf("order cancelled") }
func (cancelledState) Deliver(_ *Order) error { return fmt.Errorf("order cancelled") }
func (cancelledState) Cancel(_ *Order) error  { return fmt.Errorf("already cancelled") }

func main() {
	fmt.Println("=== Command: document with undo/redo ===")
	doc := &Document{}
	history := &CommandHistory{}

	ops := []Command{
		&InsertCommand{doc, 0, "Hello"},
		&InsertCommand{doc, 5, " World"},
		&InsertCommand{doc, 11, "!"},
	}
	for _, cmd := range ops {
		_ = history.Execute(cmd)
		fmt.Printf("  doc: %q\n", doc.Text())
	}

	fmt.Println("  -- undo last two --")
	_ = history.Undo()
	fmt.Printf("  doc: %q\n", doc.Text())
	_ = history.Undo()
	fmt.Printf("  doc: %q\n", doc.Text())

	fmt.Println("  -- redo one --")
	_ = history.Redo()
	fmt.Printf("  doc: %q\n", doc.Text())

	fmt.Println("  -- delete 5 chars at 0, then undo --")
	_ = history.Execute(&DeleteCommand{doc: doc, pos: 0, length: 5})
	fmt.Printf("  doc: %q\n", doc.Text())
	_ = history.Undo()
	fmt.Printf("  doc: %q\n", doc.Text())

	fmt.Println()
	fmt.Println("=== Iterator: BST inorder traversal ===")
	var root *TreeNode
	for _, v := range []int{5, 3, 7, 1, 4, 6, 8} {
		root = insert(root, v)
	}
	next := InorderIterator(root)
	var values []string
	for v, ok := next(); ok; v, ok = next() {
		values = append(values, fmt.Sprintf("%d", v))
	}
	fmt.Printf("  inorder: %s\n", strings.Join(values, " "))

	fmt.Println()
	fmt.Println("=== Iterator: range ===")
	rng := RangeIterator(0, 10, 2)
	var evens []string
	for v, ok := rng(); ok; v, ok = rng() {
		evens = append(evens, fmt.Sprintf("%d", v))
	}
	fmt.Printf("  even 0..9: %s\n", strings.Join(evens, " "))

	fmt.Println()
	fmt.Println("=== State: order lifecycle ===")
	ord := NewOrder("ORD-001")
	fmt.Printf("  status: %s\n", ord.Status())

	fmt.Println("  -- try ship before pay --")
	if err := ord.Ship(); err != nil {
		fmt.Println("  error:", err)
	}

	_ = ord.Pay()
	fmt.Printf("  status: %s\n", ord.Status())
	_ = ord.Ship()
	fmt.Printf("  status: %s\n", ord.Status())
	_ = ord.Deliver()
	fmt.Printf("  status: %s\n", ord.Status())

	fmt.Println("  -- try cancel delivered order --")
	if err := ord.Cancel(); err != nil {
		fmt.Println("  error:", err)
	}

	fmt.Println()
	fmt.Println("  -- cancel a paid order --")
	ord2 := NewOrder("ORD-002")
	_ = ord2.Pay()
	_ = ord2.Cancel()
	fmt.Printf("  status: %s\n", ord2.Status())
}
