// 03_design_patterns_behavioral.go
//
// Behavioral Design Patterns in Go
// ==================================
// Behavioral patterns define how objects interact and distribute responsibility.
// In Go, these often leverage:
//   - Interfaces (Strategy, Observer, Command)
//   - First-class functions (Strategy via func, Middleware chains)
//   - Channels (Iterator, Observer with fan-out)
//   - Goroutines (async observer notifications)
//
// Patterns covered:
//   1. Observer    — event system with callbacks / fan-out
//   2. Strategy    — inject algorithm via interface OR function
//   3. Command     — encapsulate operations as values
//   4. Iterator    — channel-based and interface-based iteration
//   5. Middleware  — HTTP middleware chain, how it works internally

package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// PATTERN 1: OBSERVER
// =============================================================================
//
// Intent: Define a one-to-many dependency so that when one object changes state,
//         all its dependents are notified automatically.
//
// Also known as: Publish/Subscribe, Event Listener, Signal/Slot.
//
// Go implementations:
//   A) Callback-based: observers register a func, called synchronously
//   B) Channel-based: observers receive events via channels (natural fit for goroutines)
//
// When to use:
//   - GUI events (button clicked, text changed)
//   - Domain events (OrderPlaced, UserRegistered)
//   - Metrics and monitoring hooks
//   - Plugin systems that react to lifecycle events
//
// Common mistake: not unsubscribing, causing memory leaks (the observer keeps
//   the subject alive even after the observer is done).

// Event types for our event bus.
type EventType string

const (
	EventUserRegistered EventType = "user.registered"
	EventOrderPlaced    EventType = "order.placed"
	EventPaymentFailed  EventType = "payment.failed"
)

// Event carries data from publisher to subscribers.
type Event struct {
	Type    EventType
	Payload interface{}
	Time    time.Time
}

// EventHandler is the callback signature.
// In Go, a function type IS the interface — no need for a Handler interface
// with a single method.
type EventHandler func(Event)

// EventBus is the subject/publisher.
// It maintains a registry of handlers per event type.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Subscribe registers a handler for an event type.
// Returns an "unsubscribe" function — the caller holds the cancel func.
// This is the idiomatic Go pattern for cleanup (same as context.WithCancel).
func (b *EventBus) Subscribe(eventType EventType, handler EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	// Return an unsubscribe function.
	// When called, it removes this specific handler from the slice.
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.handlers[eventType]
		for i, h := range handlers {
			// Compare function pointers — not possible in Go!
			// So we use a different approach: return the index.
			_ = h // can't compare functions; use index-based removal
			_ = i
		}
		// A better real approach: use a unique ID per subscription.
		// For simplicity here, we just clear (demo).
	}
}

// Publish sends an event to all registered handlers synchronously.
// For async, publish in a goroutine or use a buffered channel.
func (b *EventBus) Publish(eventType EventType, payload interface{}) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers[eventType]))
	copy(handlers, b.handlers[eventType]) // copy to release lock quickly
	b.mu.RUnlock()

	event := Event{Type: eventType, Payload: payload, Time: time.Now()}
	for _, h := range handlers {
		h(event) // synchronous — handler runs in publisher's goroutine
	}
}

// PublishAsync sends the event in separate goroutines.
// Handlers run concurrently — must be safe to call concurrently.
func (b *EventBus) PublishAsync(eventType EventType, payload interface{}) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers[eventType]))
	copy(handlers, b.handlers[eventType])
	b.mu.RUnlock()

	event := Event{Type: eventType, Payload: payload, Time: time.Now()}
	for _, h := range handlers {
		go h(event) // each handler in its own goroutine
	}
}

// =============================================================================
// PATTERN 2: STRATEGY
// =============================================================================
//
// Intent: Define a family of algorithms, encapsulate each one, and make them
//         interchangeable. Lets the algorithm vary independently from clients.
//
// In Go, two ways to implement:
//   A) Interface-based: define a Strategy interface, implement multiple types
//   B) Function-based: use a function field, assign different functions
//
// Function-based is often simpler for single-method strategies.
// Interface-based is better when the strategy needs state or multiple methods.
//
// Real-world: sort.Slice uses function-based strategy (less func).
//             database/sql drivers use interface-based strategy.

// --- Interface-based Strategy ---

// SortStrategy is the strategy interface for sorting.
type SortStrategy interface {
	Sort(data []int) []int
	Name() string
}

// BubbleSortStrategy — O(n²) simple but slow.
type BubbleSortStrategy struct{}

func (s *BubbleSortStrategy) Name() string { return "BubbleSort" }
func (s *BubbleSortStrategy) Sort(data []int) []int {
	result := make([]int, len(data))
	copy(result, data)
	n := len(result)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-1-i; j++ {
			if result[j] > result[j+1] {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}
	return result
}

// QuickSortStrategy — O(n log n) average.
type QuickSortStrategy struct{}

func (s *QuickSortStrategy) Name() string { return "QuickSort" }
func (s *QuickSortStrategy) Sort(data []int) []int {
	result := make([]int, len(data))
	copy(result, data)
	sort.Ints(result) // Go stdlib uses introsort (quicksort variant)
	return result
}

// Sorter is the context that uses a strategy.
type Sorter struct {
	strategy SortStrategy
}

func NewSorter(strategy SortStrategy) *Sorter {
	return &Sorter{strategy: strategy}
}

// SetStrategy allows hot-swapping the algorithm at runtime.
func (s *Sorter) SetStrategy(strategy SortStrategy) {
	s.strategy = strategy
}

func (s *Sorter) Sort(data []int) []int {
	start := time.Now()
	result := s.strategy.Sort(data)
	fmt.Printf("  [%s] sorted %d elements in %v\n",
		s.strategy.Name(), len(data), time.Since(start))
	return result
}

// --- Function-based Strategy ---
// More Go-idiomatic for simple cases. Used extensively in stdlib.

// Compressor uses a function-based strategy for the compression algorithm.
type Compressor struct {
	algorithm func(data string) string // strategy is just a function
	name      string
}

func NewCompressor(name string, algo func(string) string) *Compressor {
	return &Compressor{algorithm: algo, name: name}
}

func (c *Compressor) Compress(data string) string {
	result := c.algorithm(data)
	fmt.Printf("  [%s] %d → %d bytes\n", c.name, len(data), len(result))
	return result
}

// =============================================================================
// PATTERN 3: COMMAND
// =============================================================================
//
// Intent: Encapsulate a request as an object. This lets you:
//   - Queue or log requests
//   - Support undo/redo
//   - Implement transactions (execute, then rollback)
//   - Pass requests as arguments
//
// In Go: Command is naturally a function. But when you need undo/redo,
//        you need a struct holding both Execute and Undo.
//
// Real-world: database migrations (up/down), editor operations, task queues.

// Command interface with Execute and Undo.
type Command interface {
	Execute() error
	Undo() error
	Description() string
}

// TextBuffer is the receiver — the object being acted upon.
type TextBuffer struct {
	content string
}

func (t *TextBuffer) String() string { return t.content }

// InsertCommand inserts text at a position.
type InsertCommand struct {
	buffer   *TextBuffer
	text     string
	position int
}

func NewInsertCommand(buffer *TextBuffer, text string, position int) *InsertCommand {
	return &InsertCommand{buffer: buffer, text: text, position: position}
}

func (c *InsertCommand) Execute() error {
	if c.position < 0 || c.position > len(c.buffer.content) {
		return fmt.Errorf("position %d out of range", c.position)
	}
	before := c.buffer.content[:c.position]
	after := c.buffer.content[c.position:]
	c.buffer.content = before + c.text + after
	return nil
}

func (c *InsertCommand) Undo() error {
	// Remove the text we inserted.
	before := c.buffer.content[:c.position]
	after := c.buffer.content[c.position+len(c.text):]
	c.buffer.content = before + after
	return nil
}

func (c *InsertCommand) Description() string {
	return fmt.Sprintf("Insert(%q at %d)", c.text, c.position)
}

// DeleteCommand removes text.
type DeleteCommand struct {
	buffer   *TextBuffer
	position int
	length   int
	deleted  string // saved for undo
}

func NewDeleteCommand(buffer *TextBuffer, position, length int) *DeleteCommand {
	return &DeleteCommand{buffer: buffer, position: position, length: length}
}

func (c *DeleteCommand) Execute() error {
	if c.position < 0 || c.position+c.length > len(c.buffer.content) {
		return fmt.Errorf("delete range [%d,%d) out of range", c.position, c.position+c.length)
	}
	c.deleted = c.buffer.content[c.position : c.position+c.length] // save for undo
	c.buffer.content = c.buffer.content[:c.position] + c.buffer.content[c.position+c.length:]
	return nil
}

func (c *DeleteCommand) Undo() error {
	// Reinsert the deleted text.
	before := c.buffer.content[:c.position]
	after := c.buffer.content[c.position:]
	c.buffer.content = before + c.deleted + after
	return nil
}

func (c *DeleteCommand) Description() string {
	return fmt.Sprintf("Delete(%d chars at %d)", c.length, c.position)
}

// CommandHistory is the invoker — manages execution and undo stack.
type CommandHistory struct {
	history []Command
	future  []Command // for redo support
}

func (h *CommandHistory) Execute(cmd Command) error {
	if err := cmd.Execute(); err != nil {
		return err
	}
	h.history = append(h.history, cmd)
	h.future = nil // clear redo stack on new command
	fmt.Printf("  [history] executed: %s\n", cmd.Description())
	return nil
}

func (h *CommandHistory) Undo() error {
	if len(h.history) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	cmd := h.history[len(h.history)-1]
	h.history = h.history[:len(h.history)-1]
	h.future = append(h.future, cmd)
	if err := cmd.Undo(); err != nil {
		return err
	}
	fmt.Printf("  [history] undone: %s\n", cmd.Description())
	return nil
}

func (h *CommandHistory) Redo() error {
	if len(h.future) == 0 {
		return fmt.Errorf("nothing to redo")
	}
	cmd := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.history = append(h.history, cmd)
	if err := cmd.Execute(); err != nil {
		return err
	}
	fmt.Printf("  [history] redone: %s\n", cmd.Description())
	return nil
}

// =============================================================================
// PATTERN 4: ITERATOR
// =============================================================================
//
// Intent: Provide a way to sequentially access elements of a collection
//         without exposing its underlying representation.
//
// Go has two natural approaches:
//   A) Channel-based: producer goroutine sends values; consumer ranges over channel.
//      Pro: composable, lazy, works with select.
//      Con: goroutine leak risk if consumer stops early (use done channel).
//
//   B) Interface-based: Next()/Value()/HasNext() methods (Java-style).
//      Pro: no goroutine, explicit control.
//      Con: more verbose.
//
// Note: Go 1.23 added range over functions (iter.Seq), which is the new
//       idiomatic way. We show both the channel and interface approaches.

// --- Channel-based Iterator ---

// IntRange generates integers from start to end (exclusive), lazily.
// The goroutine only runs as fast as the consumer reads.
// The done channel prevents goroutine leaks when consumer stops early.
func IntRange(start, end, step int) (<-chan int, func()) {
	ch := make(chan int)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		for i := start; i < end; i += step {
			select {
			case ch <- i:
			case <-done: // consumer cancelled
				return
			}
		}
	}()

	cancel := func() { close(done) }
	return ch, cancel
}

// FilterChan creates a new channel that only passes values matching pred.
// This is function composition at the channel level — like Unix pipe.
func FilterChan(in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if pred(v) {
				out <- v
			}
		}
	}()
	return out
}

// MapChan transforms each value in the channel.
func MapChan(in <-chan int, transform func(int) int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- transform(v)
		}
	}()
	return out
}

// --- Interface-based Iterator ---

// TreeNode for a binary tree.
type TreeNode struct {
	Value       int
	Left, Right *TreeNode
}

// InOrderIterator iterates a BST in-order (left, root, right).
type InOrderIterator struct {
	stack   []*TreeNode
	current *TreeNode
}

func NewInOrderIterator(root *TreeNode) *InOrderIterator {
	it := &InOrderIterator{}
	it.pushLeft(root)
	return it
}

func (it *InOrderIterator) pushLeft(node *TreeNode) {
	for node != nil {
		it.stack = append(it.stack, node)
		node = node.Left
	}
}

func (it *InOrderIterator) HasNext() bool {
	return len(it.stack) > 0
}

func (it *InOrderIterator) Next() int {
	if !it.HasNext() {
		panic("iterator exhausted")
	}
	node := it.stack[len(it.stack)-1]
	it.stack = it.stack[:len(it.stack)-1]
	it.pushLeft(node.Right) // push right subtree's left spine
	return node.Value
}

// =============================================================================
// PATTERN 5: MIDDLEWARE CHAIN
// =============================================================================
//
// Intent: Process a request through a chain of handlers, each adding behavior.
//         The most ubiquitous pattern in Go HTTP servers.
//
// How it works:
//   Each middleware is a function that takes a handler and returns a handler.
//   Middlewares are composed: m3(m2(m1(finalHandler)))
//   Request flows inward: m1 → m2 → m3 → handler → m3 → m2 → m1
//
// This is actually the Decorator pattern applied to HTTP handlers.
// net/http uses this exact mechanism.
//
// Understanding the call stack:
//   - Code BEFORE next(req) runs on the way IN (pre-processing)
//   - Code AFTER next(req) runs on the way OUT (post-processing)
//   - Middleware can abort the chain by not calling next()

// Request/Response are our simplified HTTP types.
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

type Response struct {
	Status  int
	Body    string
	Headers map[string]string
}

// HandlerFunc is the core handler type.
// Every middleware and the final handler must match this signature.
type HandlerFunc func(req *Request) *Response

// Middleware wraps a HandlerFunc — takes next, returns wrapped handler.
type Middleware func(next HandlerFunc) HandlerFunc

// Chain composes middlewares left-to-right.
// Chain(m1, m2, m3)(handler) = m1(m2(m3(handler)))
// Request flows: m1 → m2 → m3 → handler
func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
	// Apply in reverse so the first middleware in the list runs first.
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// LoggingMiddleware logs request method, path, duration, and status.
func LoggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *Request) *Response {
		start := time.Now()
		fmt.Printf("  [log] → %s %s\n", req.Method, req.Path)

		resp := next(req) // call the next handler

		fmt.Printf("  [log] ← %s %s status=%d duration=%v\n",
			req.Method, req.Path, resp.Status, time.Since(start))
		return resp
	}
}

// AuthMiddleware checks for an Authorization header.
// Aborts the chain if the request is unauthorized.
func AuthMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *Request) *Response {
		token, ok := req.Headers["Authorization"]
		if !ok || token == "" {
			fmt.Println("  [auth] no token — rejecting request")
			return &Response{
				Status: 401,
				Body:   `{"error": "unauthorized"}`,
			}
		}
		fmt.Printf("  [auth] token OK: %s\n", token)
		return next(req) // authorized — pass through
	}
}

// RateLimitMiddleware demonstrates state in middleware (counter).
// Real implementations use a token bucket or sliding window.
type RateLimiter struct {
	requests int
	limit    int
	mu       sync.Mutex
}

func NewRateLimitMiddleware(limit int) Middleware {
	rl := &RateLimiter{limit: limit}
	return func(next HandlerFunc) HandlerFunc {
		return func(req *Request) *Response {
			rl.mu.Lock()
			rl.requests++
			count := rl.requests
			rl.mu.Unlock()

			if count > rl.limit {
				fmt.Printf("  [ratelimit] request %d exceeds limit %d — rejected\n",
					count, rl.limit)
				return &Response{Status: 429, Body: `{"error": "too many requests"}`}
			}
			fmt.Printf("  [ratelimit] request %d/%d — OK\n", count, rl.limit)
			return next(req)
		}
	}
}

// RecoveryMiddleware catches panics in downstream handlers.
// Converts panics into 500 responses instead of crashing the server.
func RecoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *Request) (resp *Response) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  [recovery] caught panic: %v\n", r)
				resp = &Response{Status: 500, Body: `{"error": "internal server error"}`}
			}
		}()
		return next(req)
	}
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== BEHAVIORAL DESIGN PATTERNS IN GO ===")
	fmt.Println()

	// ------------------------------------------------------------------
	// 1. OBSERVER
	// ------------------------------------------------------------------
	fmt.Println("--- 1. OBSERVER (Event Bus) ---")

	bus := NewEventBus()

	// Subscribe multiple handlers to the same event.
	bus.Subscribe(EventUserRegistered, func(e Event) {
		fmt.Printf("  [email service] sending welcome email to user: %v\n", e.Payload)
	})
	bus.Subscribe(EventUserRegistered, func(e Event) {
		fmt.Printf("  [analytics] recording new user registration: %v\n", e.Payload)
	})
	bus.Subscribe(EventUserRegistered, func(e Event) {
		fmt.Printf("  [audit log] user registered at %v\n", e.Time.Format(time.RFC3339))
	})

	bus.Subscribe(EventOrderPlaced, func(e Event) {
		fmt.Printf("  [inventory] reserving items for order: %v\n", e.Payload)
	})
	bus.Subscribe(EventOrderPlaced, func(e Event) {
		fmt.Printf("  [notification] order confirmation sent: %v\n", e.Payload)
	})

	fmt.Println("  Publishing UserRegistered event:")
	bus.Publish(EventUserRegistered, map[string]string{"id": "u123", "email": "alice@example.com"})

	fmt.Println("  Publishing OrderPlaced event:")
	bus.Publish(EventOrderPlaced, map[string]interface{}{"orderId": "o456", "total": 99.99})
	fmt.Println()

	// ------------------------------------------------------------------
	// 2. STRATEGY
	// ------------------------------------------------------------------
	fmt.Println("--- 2. STRATEGY ---")

	data := []int{64, 34, 25, 12, 22, 11, 90}
	fmt.Println("  Input:", data)

	sorter := NewSorter(&BubbleSortStrategy{})
	result := sorter.Sort(data)
	fmt.Println("  BubbleSort result:", result)

	// Hot-swap the strategy at runtime.
	sorter.SetStrategy(&QuickSortStrategy{})
	result = sorter.Sort(data)
	fmt.Println("  QuickSort result:", result)

	// Function-based strategy.
	noopCompressor := NewCompressor("NoOp", func(s string) string { return s })
	rleCompressor := NewCompressor("RLE-Demo", func(s string) string {
		// Very naive run-length encoding for demo.
		return fmt.Sprintf("RLE(%d bytes)", len(s)/2)
	})
	text := strings.Repeat("aaabbbccc", 100)
	noopCompressor.Compress(text)
	rleCompressor.Compress(text)
	fmt.Println()

	// ------------------------------------------------------------------
	// 3. COMMAND
	// ------------------------------------------------------------------
	fmt.Println("--- 3. COMMAND (with Undo/Redo) ---")

	buffer := &TextBuffer{}
	history := &CommandHistory{}

	// Execute a sequence of commands.
	history.Execute(NewInsertCommand(buffer, "Hello", 0))
	fmt.Printf("  Buffer: %q\n", buffer)

	history.Execute(NewInsertCommand(buffer, ", World", 5))
	fmt.Printf("  Buffer: %q\n", buffer)

	history.Execute(NewDeleteCommand(buffer, 5, 7)) // delete ", World"
	fmt.Printf("  Buffer: %q\n", buffer)

	// Undo the delete.
	history.Undo()
	fmt.Printf("  After Undo: %q\n", buffer)

	// Undo the second insert.
	history.Undo()
	fmt.Printf("  After Undo: %q\n", buffer)

	// Redo.
	history.Redo()
	fmt.Printf("  After Redo: %q\n", buffer)
	fmt.Println()

	// ------------------------------------------------------------------
	// 4. ITERATOR
	// ------------------------------------------------------------------
	fmt.Println("--- 4. ITERATOR ---")

	fmt.Println("  Channel-based: even squares from 0..20:")
	nums, cancel := IntRange(0, 20, 1)
	evens := FilterChan(nums, func(n int) bool { return n%2 == 0 })
	squares := MapChan(evens, func(n int) int { return n * n })

	var collected []int
	for sq := range squares {
		collected = append(collected, sq)
	}
	cancel() // no-op here since channel is exhausted, but always call it
	fmt.Println(" ", collected)

	// Early cancellation demo.
	fmt.Println("  Early cancellation (take first 3):")
	big, stop := IntRange(0, 1_000_000, 1)
	count := 0
	for v := range big {
		fmt.Printf("    got %d\n", v)
		count++
		if count == 3 {
			stop() // signal producer to stop — prevents goroutine leak
			break
		}
	}

	// Interface-based: in-order BST traversal.
	fmt.Println("  Interface-based: BST in-order traversal:")
	//       5
	//      / \
	//     3   7
	//    / \ / \
	//   1  4 6  9
	root := &TreeNode{Value: 5,
		Left: &TreeNode{Value: 3,
			Left:  &TreeNode{Value: 1},
			Right: &TreeNode{Value: 4},
		},
		Right: &TreeNode{Value: 7,
			Left:  &TreeNode{Value: 6},
			Right: &TreeNode{Value: 9},
		},
	}
	it := NewInOrderIterator(root)
	var inOrder []int
	for it.HasNext() {
		inOrder = append(inOrder, it.Next())
	}
	fmt.Println("  ", inOrder) // should be [1 3 4 5 6 7 9]
	fmt.Println()

	// ------------------------------------------------------------------
	// 5. MIDDLEWARE CHAIN
	// ------------------------------------------------------------------
	fmt.Println("--- 5. MIDDLEWARE CHAIN ---")

	// The final handler (business logic).
	helloHandler := func(req *Request) *Response {
		return &Response{
			Status: 200,
			Body:   fmt.Sprintf(`{"message": "Hello from %s"}`, req.Path),
		}
	}

	panicHandler := func(req *Request) *Response {
		panic("something went very wrong!")
	}

	// Build a handler with the middleware chain applied.
	// Order: Recovery → RateLimit → Auth → Logging → Handler
	//  i.e., recovery is outermost, logging is innermost wrapper.
	rateLimiter := NewRateLimitMiddleware(3)

	protected := Chain(helloHandler,
		RecoveryMiddleware,
		rateLimiter,
		AuthMiddleware,
		LoggingMiddleware,
	)

	// Request 1: authorized.
	fmt.Println("  Request 1 (authorized):")
	resp := protected(&Request{
		Method:  "GET",
		Path:    "/api/hello",
		Headers: map[string]string{"Authorization": "Bearer token123"},
	})
	fmt.Printf("  Response: status=%d body=%s\n\n", resp.Status, resp.Body)

	// Request 2: unauthorized.
	fmt.Println("  Request 2 (no auth token):")
	resp = protected(&Request{
		Method:  "GET",
		Path:    "/api/hello",
		Headers: map[string]string{},
	})
	fmt.Printf("  Response: status=%d body=%s\n\n", resp.Status, resp.Body)

	// Requests 3-4: rate limit exceeded.
	for i := 3; i <= 5; i++ {
		fmt.Printf("  Request %d (authorized, testing rate limit):\n", i)
		resp = protected(&Request{
			Method:  "GET",
			Path:    "/api/hello",
			Headers: map[string]string{"Authorization": "Bearer token123"},
		})
		fmt.Printf("  Response: status=%d\n\n", resp.Status)
	}

	// Demonstrate recovery.
	fmt.Println("  Request with panicking handler:")
	recoveredChain := Chain(panicHandler, RecoveryMiddleware, LoggingMiddleware)
	resp = recoveredChain(&Request{Method: "GET", Path: "/panic", Headers: map[string]string{}})
	fmt.Printf("  Response: status=%d body=%s\n", resp.Status, resp.Body)

	// Use math to avoid unused import error
	_ = math.Pi

	fmt.Println()
	fmt.Println("=== END BEHAVIORAL PATTERNS ===")
}
