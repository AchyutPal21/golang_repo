// FILE: book/part5_building_backends/chapter76_grpc/examples/02_grpc_patterns/main.go
// CHAPTER: 76 — gRPC
// TOPIC: gRPC production patterns — client-side streaming, bidirectional streaming,
//        deadlines, retry with backoff, metadata (headers/trailers), and connection pooling.
//
// Run (from the chapter folder):
//   go run ./examples/02_grpc_patterns

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// STATUS CODES (same as example 01)
// ─────────────────────────────────────────────────────────────────────────────

type Code int

const (
	CodeOK          Code = 0
	CodeCancelled   Code = 1
	CodeUnavailable Code = 14
	CodeDeadline    Code = 4
)

type StatusError struct {
	Code    Code
	Message string
}

func (s *StatusError) Error() string {
	return fmt.Sprintf("code=%d desc=%s", s.Code, s.Message)
}

func statusErr(code Code, msg string) error { return &StatusError{Code: code, Message: msg} }
func codeOf(err error) Code {
	var s *StatusError
	if errors.As(err, &s) {
		return s.Code
	}
	return -1
}

// ─────────────────────────────────────────────────────────────────────────────
// METADATA — key-value pairs sent alongside RPC calls (headers + trailers)
// ─────────────────────────────────────────────────────────────────────────────

type MD map[string]string

type metaKey struct{}

func WithMetadata(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, metaKey{}, md)
}

func MetadataFromContext(ctx context.Context) MD {
	if md, ok := ctx.Value(metaKey{}).(MD); ok {
		return md
	}
	return MD{}
}

// ─────────────────────────────────────────────────────────────────────────────
// CLIENT-SIDE STREAMING
// A stream of requests sent from client to server; server returns one response.
// Use case: bulk upload, batch mutations.
// ─────────────────────────────────────────────────────────────────────────────

type InventoryUpdate struct {
	ProductID string
	Delta     int // positive = add stock, negative = remove
}

type BulkUpdateResult struct {
	Applied int
	Failed  int
	Errors  []string
}

// BulkUpdateInventory accepts a series of updates and returns a summary.
func BulkUpdateInventory(ctx context.Context, updates []InventoryUpdate) (*BulkUpdateResult, error) {
	// Simulate a server that processes a client-sent stream.
	inventory := map[string]int{"p-1": 10, "p-2": 5, "p-3": 0}
	result := &BulkUpdateResult{}

	for _, u := range updates {
		select {
		case <-ctx.Done():
			return nil, statusErr(CodeDeadline, "context deadline exceeded")
		default:
		}
		current, ok := inventory[u.ProductID]
		if !ok {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: not found", u.ProductID))
			continue
		}
		newStock := current + u.Delta
		if newStock < 0 {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: insufficient stock", u.ProductID))
			continue
		}
		inventory[u.ProductID] = newStock
		result.Applied++
		fmt.Printf("  [server] %s stock %d → %d\n", u.ProductID, current, newStock)
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// BIDIRECTIONAL STREAMING
// Both client and server send streams of messages concurrently.
// Use case: chat, live price feed with dynamic subscriptions.
// ─────────────────────────────────────────────────────────────────────────────

type PriceRequest struct {
	ProductID string
	Subscribe bool // true=subscribe, false=unsubscribe
}

type PriceUpdate struct {
	ProductID string
	Price     int
	Timestamp time.Time
}

// BidirectionalPriceFeed simulates a bidi stream: client sends subscribe/unsubscribe
// commands and server streams back price updates for subscribed products.
func BidirectionalPriceFeed(
	ctx context.Context,
	requests []PriceRequest,
	onUpdate func(*PriceUpdate),
) error {
	subscribed := make(map[string]bool)
	prices := map[string]int{"p-1": 3999, "p-2": 12999, "p-3": 4999}
	var mu sync.Mutex

	// Server sends price updates in a goroutine.
	stopServer := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		ticker := time.NewTicker(30 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopServer:
				return
			case <-ticker.C:
				mu.Lock()
				for id, price := range prices {
					if subscribed[id] {
						onUpdate(&PriceUpdate{ProductID: id, Price: price, Timestamp: time.Now()})
						prices[id] = price + 1 // tiny fluctuation
					}
				}
				mu.Unlock()
			}
		}
	}()

	// Client sends subscription commands.
	for _, req := range requests {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		if req.Subscribe {
			subscribed[req.ProductID] = true
			fmt.Printf("  [client] subscribed to %s\n", req.ProductID)
		} else {
			delete(subscribed, req.ProductID)
			fmt.Printf("  [client] unsubscribed from %s\n", req.ProductID)
		}
		mu.Unlock()
	}

	close(stopServer)
	<-serverDone
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DEADLINE PROPAGATION
// ─────────────────────────────────────────────────────────────────────────────

func slowOperation(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return statusErr(CodeDeadline, ctx.Err().Error())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY WITH EXPONENTIAL BACKOFF
// ─────────────────────────────────────────────────────────────────────────────

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	// RetryOn returns true if the error should be retried.
	RetryOn func(err error) bool
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 4,
		BaseDelay:   time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
		RetryOn: func(err error) bool {
			return codeOf(err) == CodeUnavailable
		},
	}
}

func withRetry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	delay := policy.BaseDelay
	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !policy.RetryOn(lastErr) {
			return lastErr
		}
		if attempt == policy.MaxAttempts {
			break
		}
		fmt.Printf("  [retry] attempt %d/%d failed (%v), waiting %s\n",
			attempt, policy.MaxAttempts, lastErr, delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", policy.MaxAttempts, lastErr)
}

// ─────────────────────────────────────────────────────────────────────────────
// CONNECTION POOL (simulated)
// ─────────────────────────────────────────────────────────────────────────────

type Conn struct {
	ID      int
	InUse   bool
}

type ConnPool struct {
	mu    sync.Mutex
	conns []*Conn
}

func NewConnPool(size int) *ConnPool {
	pool := &ConnPool{}
	for i := 0; i < size; i++ {
		pool.conns = append(pool.conns, &Conn{ID: i + 1})
	}
	return pool
}

func (p *ConnPool) Get() (*Conn, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		if !c.InUse {
			c.InUse = true
			return c, true
		}
	}
	return nil, false
}

func (p *ConnPool) Put(c *Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c.InUse = false
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== gRPC Patterns ===")
	fmt.Println()

	// ── CLIENT-SIDE STREAMING ─────────────────────────────────────────────────
	fmt.Println("--- Client-side streaming: BulkUpdateInventory ---")
	ctx := context.Background()

	updates := []InventoryUpdate{
		{"p-1", 5},   // add 5
		{"p-2", -3},  // remove 3
		{"p-3", -1},  // fails: stock 0 - 1 < 0
		{"p-99", 10}, // fails: not found
	}
	result, err := BulkUpdateInventory(ctx, updates)
	if err != nil {
		fmt.Printf("  error: %v\n", err)
	} else {
		fmt.Printf("  applied=%d failed=%d errors=%v\n", result.Applied, result.Failed, result.Errors)
	}

	// ── METADATA ─────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Metadata (gRPC headers/trailers) ---")
	metaCtx := WithMetadata(ctx, MD{
		"authorization": "Bearer token-xyz",
		"x-request-id": "req-001",
		"x-trace-id":   "trace-abc",
	})
	md := MetadataFromContext(metaCtx)
	for k, v := range md {
		fmt.Printf("  metadata: %s = %s\n", k, v)
	}
	fmt.Println("  (in real gRPC: metadata.NewOutgoingContext / metadata.FromIncomingContext)")

	// ── BIDIRECTIONAL STREAMING ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Bidirectional streaming: PriceFeed ---")
	biCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	var updateCount int
	err = BidirectionalPriceFeed(biCtx, []PriceRequest{
		{ProductID: "p-1", Subscribe: true},
		{ProductID: "p-2", Subscribe: true},
		{ProductID: "p-1", Subscribe: false},
	}, func(u *PriceUpdate) {
		updateCount++
		fmt.Printf("  [price] %s = %d\n", u.ProductID, u.Price)
	})
	fmt.Printf("  received %d price updates\n", updateCount)

	// ── DEADLINE PROPAGATION ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Deadline propagation ---")

	deadlineCtx, deadlineCancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer deadlineCancel()

	fmt.Println("  fast operation (5ms deadline=10ms): should succeed")
	if err := slowOperation(deadlineCtx, 5*time.Millisecond); err != nil {
		fmt.Printf("  unexpected error: %v\n", err)
	} else {
		fmt.Println("  fast operation: ok")
	}

	deadlineCtx2, deadlineCancel2 := context.WithTimeout(ctx, 5*time.Millisecond)
	defer deadlineCancel2()
	fmt.Println("  slow operation (20ms) with deadline=5ms: should fail")
	if err := slowOperation(deadlineCtx2, 20*time.Millisecond); err != nil {
		fmt.Printf("  deadline exceeded: code=%d\n", codeOf(err))
	}

	// ── RETRY WITH EXPONENTIAL BACKOFF ────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Retry with exponential backoff ---")

	var callCount int
	retryCtx := context.Background()
	err = withRetry(retryCtx, defaultRetryPolicy(), func() error {
		callCount++
		if callCount < 3 {
			return statusErr(CodeUnavailable, "service temporarily down")
		}
		fmt.Printf("  call %d: succeeded\n", callCount)
		return nil
	})
	if err != nil {
		fmt.Printf("  final error: %v\n", err)
	} else {
		fmt.Printf("  succeeded after %d attempts\n", callCount)
	}

	// ── CONNECTION POOL ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Connection pool (5 connections, 3 concurrent RPCs) ---")
	pool := NewConnPool(5)
	var poolWg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		i := i
		poolWg.Add(1)
		go func() {
			defer poolWg.Done()
			conn, ok := pool.Get()
			if !ok {
				fmt.Printf("  worker-%d: no connection available\n", i)
				return
			}
			fmt.Printf("  worker-%d acquired conn-%d\n", i, conn.ID)
			time.Sleep(10 * time.Millisecond) // simulate RPC
			pool.Put(conn)
			fmt.Printf("  worker-%d released conn-%d\n", i, conn.ID)
		}()
	}
	poolWg.Wait()

	// ── REAL gRPC QUICK-REFERENCE ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Real gRPC quick reference ---")
	fmt.Println(`  // Server
  s := grpc.NewServer(
      grpc.UnaryInterceptor(loggingInterceptor),
      grpc.ChainUnaryInterceptor(authInterceptor, traceInterceptor),
  )
  pb.RegisterProductServiceServer(s, &productServer{})
  s.Serve(lis)

  // Client
  conn, _ := grpc.Dial("localhost:50051",
      grpc.WithTransportCredentials(insecure.NewCredentials()),
      grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4<<20)),
  )
  client := pb.NewProductServiceClient(conn)
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  resp, err := client.GetProduct(ctx, &pb.GetProductRequest{ProductId: "p-1"})`)
}
