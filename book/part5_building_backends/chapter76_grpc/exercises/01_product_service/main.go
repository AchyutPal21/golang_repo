// FILE: book/part5_building_backends/chapter76_grpc/exercises/01_product_service/main.go
// CHAPTER: 76 — gRPC
// TOPIC: Product service with CRUD, server streaming, auth + tracing interceptors,
//        retry policy, and client connection management.
//
// Run (from the chapter folder):
//   go run ./exercises/01_product_service

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// STATUS CODES
// ─────────────────────────────────────────────────────────────────────────────

type Code int

const (
	CodeOK           Code = 0
	CodeInvalidArg   Code = 3
	CodeNotFound     Code = 5
	CodeAlreadyExists Code = 6
	CodePermDenied   Code = 7
	CodeUnavailable  Code = 14
)

type StatusError struct {
	Code    Code
	Message string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("grpc %d: %s", e.Code, e.Message)
}

func newErr(code Code, msg string) error { return &StatusError{code, msg} }
func codeOf(err error) Code {
	var s *StatusError
	if errors.As(err, &s) {
		return s.Code
	}
	return -1
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Category string

const (
	CategoryBooks     Category = "books"
	CategoryElectronics Category = "electronics"
	CategoryClothing  Category = "clothing"
)

type Product struct {
	ID          string
	Name        string
	Category    Category
	Price       int // cents
	StockCount  int
	Tags        []string
	CreatedAt   time.Time
}

type SearchRequest struct {
	Query    string
	Category Category
	MinPrice int
	MaxPrice int
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

type ProductService interface {
	CreateProduct(ctx context.Context, p *Product) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	UpdateProduct(ctx context.Context, p *Product) (*Product, error)
	DeleteProduct(ctx context.Context, id string) error
	// Server streaming: streams products matching filter; calls send for each.
	SearchProducts(ctx context.Context, req *SearchRequest, send func(*Product) error) error
	// Server streaming: watches stock changes; stops when ctx is done.
	WatchStock(ctx context.Context, productID string, send func(int) error) error
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVER IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type productService struct {
	mu       sync.RWMutex
	products map[string]*Product
	nextID   int
	// Stock watchers: productID → list of send-channels.
	watchers map[string][]chan int
}

func NewProductService() ProductService {
	svc := &productService{
		products: make(map[string]*Product),
		watchers: make(map[string][]chan int),
		nextID:   1,
	}
	svc.seed()
	return svc
}

func (s *productService) seed() {
	seeds := []*Product{
		{Name: "The Go Programming Language", Category: CategoryBooks, Price: 3999, StockCount: 50, Tags: []string{"go", "programming"}},
		{Name: "Wireless Keyboard", Category: CategoryElectronics, Price: 8999, StockCount: 12, Tags: []string{"keyboard", "wireless"}},
		{Name: "Dev Hoodie", Category: CategoryClothing, Price: 4999, StockCount: 30, Tags: []string{"hoodie", "software"}},
		{Name: "Clean Code", Category: CategoryBooks, Price: 2999, StockCount: 20, Tags: []string{"books", "programming"}},
	}
	for _, p := range seeds {
		p.ID = fmt.Sprintf("p-%d", s.nextID)
		p.CreatedAt = time.Now()
		s.products[p.ID] = p
		s.nextID++
	}
}

func (s *productService) CreateProduct(_ context.Context, p *Product) (*Product, error) {
	if p.Name == "" {
		return nil, newErr(CodeInvalidArg, "name is required")
	}
	if p.Price < 0 {
		return nil, newErr(CodeInvalidArg, "price must be >= 0")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = fmt.Sprintf("p-%d", s.nextID)
	p.CreatedAt = time.Now()
	s.nextID++
	s.products[p.ID] = p
	return p, nil
}

func (s *productService) GetProduct(_ context.Context, id string) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	if !ok {
		return nil, newErr(CodeNotFound, fmt.Sprintf("product %q not found", id))
	}
	return p, nil
}

func (s *productService) UpdateProduct(_ context.Context, p *Product) (*Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[p.ID]; !ok {
		return nil, newErr(CodeNotFound, fmt.Sprintf("product %q not found", p.ID))
	}
	s.products[p.ID] = p
	// Notify watchers of stock change.
	if chs, ok := s.watchers[p.ID]; ok {
		for _, ch := range chs {
			select {
			case ch <- p.StockCount:
			default:
			}
		}
	}
	return p, nil
}

func (s *productService) DeleteProduct(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return newErr(CodeNotFound, fmt.Sprintf("product %q not found", id))
	}
	delete(s.products, id)
	return nil
}

func (s *productService) SearchProducts(ctx context.Context, req *SearchRequest, send func(*Product) error) error {
	s.mu.RLock()
	var matches []*Product
	for _, p := range s.products {
		if req.Category != "" && p.Category != req.Category {
			continue
		}
		if req.Query != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.Query)) {
			continue
		}
		if req.MinPrice > 0 && p.Price < req.MinPrice {
			continue
		}
		if req.MaxPrice > 0 && p.Price > req.MaxPrice {
			continue
		}
		matches = append(matches, p)
	}
	s.mu.RUnlock()

	for _, p := range matches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := send(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *productService) WatchStock(ctx context.Context, productID string, send func(int) error) error {
	ch := make(chan int, 8)
	s.mu.Lock()
	s.watchers[productID] = append(s.watchers[productID], ch)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		chs := s.watchers[productID]
		for i, c := range chs {
			if c == ch {
				s.watchers[productID] = append(chs[:i], chs[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case stock := <-ch:
			if err := send(stock); err != nil {
				return err
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// INTERCEPTOR CHAIN
// ─────────────────────────────────────────────────────────────────────────────

type Handler func(ctx context.Context, req any) (any, error)
type Interceptor func(ctx context.Context, req any, method string, next Handler) (any, error)

func Chain(interceptors ...Interceptor) func(Handler) Handler {
	return func(h Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			wrapped := h
			for i := len(interceptors) - 1; i >= 0; i-- {
				ic := interceptors[i]
				inner := wrapped
				method := ""
				wrapped = func(ctx context.Context, req any) (any, error) {
					return ic(ctx, req, method, inner)
				}
			}
			return wrapped(ctx, req)
		}
	}
}

// AuthInterceptor checks for a valid token in the context.
func AuthInterceptor(ctx context.Context, req any, method string, next Handler) (any, error) {
	type tokenKey struct{}
	token, _ := ctx.Value(tokenKey{}).(string)
	if token == "" {
		return nil, newErr(CodePermDenied, "unauthenticated: missing token")
	}
	return next(ctx, req)
}

type traceKey struct{}

// TracingInterceptor injects a trace ID and logs timing.
var traceCounter atomic.Int64

func TracingInterceptor(ctx context.Context, req any, method string, next Handler) (any, error) {
	traceID := fmt.Sprintf("trace-%d", traceCounter.Add(1))
	ctx = context.WithValue(ctx, traceKey{}, traceID)
	start := time.Now()
	resp, err := next(ctx, req)
	dur := time.Since(start).Round(time.Microsecond)
	status := "ok"
	if err != nil {
		status = fmt.Sprintf("err code=%d", codeOf(err))
	}
	fmt.Printf("  [trace %s] method=%s dur=%s status=%s\n", traceID, method, dur, status)
	return resp, err
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED CLIENT
// ─────────────────────────────────────────────────────────────────────────────

type tokenKey struct{}

type Client struct {
	svc     ProductService
	chain   func(Handler) Handler
	baseCtx context.Context
}

func NewClient(svc ProductService, token string) *Client {
	ctx := context.WithValue(context.Background(), tokenKey{}, token)
	return &Client{
		svc:     svc,
		chain:   Chain(TracingInterceptor),
		baseCtx: ctx,
	}
}

func (c *Client) call(method string, req any, fn Handler) (any, error) {
	ctx := context.WithValue(c.baseCtx, struct{ k string }{"method"}, method)
	h := c.chain(fn)
	return h(ctx, req)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Product Service (gRPC exercise) ===")
	fmt.Println()

	svc := NewProductService()
	client := NewClient(svc, "bearer-abc")

	// ── CRUD ─────────────────────────────────────────────────────────────────
	fmt.Println("--- CRUD operations ---")

	// Create
	resp, err := client.call("CreateProduct", &Product{
		Name: "Ergonomic Mouse", Category: CategoryElectronics, Price: 5999, StockCount: 25,
	}, func(ctx context.Context, req any) (any, error) {
		return svc.CreateProduct(ctx, req.(*Product))
	})
	if err != nil {
		fmt.Printf("  create error: %v\n", err)
	} else {
		created := resp.(*Product)
		fmt.Printf("  created: id=%s name=%q\n", created.ID, created.Name)
	}

	// Get
	resp, _ = client.call("GetProduct", "p-1", func(ctx context.Context, req any) (any, error) {
		return svc.GetProduct(ctx, req.(string))
	})
	p := resp.(*Product)
	fmt.Printf("  get p-1: %q price=%d stock=%d\n", p.Name, p.Price, p.StockCount)

	// Update
	p.StockCount = 100
	_, _ = client.call("UpdateProduct", p, func(ctx context.Context, req any) (any, error) {
		return svc.UpdateProduct(ctx, req.(*Product))
	})

	// Delete
	_, _ = client.call("DeleteProduct", "p-3", func(ctx context.Context, req any) (any, error) {
		return svc.DeleteProduct(ctx, req.(string)), nil
	})

	// Not found after delete
	_, err = client.call("GetProduct", "p-3", func(ctx context.Context, req any) (any, error) {
		return svc.GetProduct(ctx, req.(string))
	})
	fmt.Printf("  get deleted p-3: code=%d\n", codeOf(err))

	// ── SERVER STREAMING: SEARCH ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Server streaming: SearchProducts (category=books) ---")
	ctx := context.Background()
	var bookCount int
	err = svc.SearchProducts(ctx, &SearchRequest{Category: CategoryBooks}, func(p *Product) error {
		bookCount++
		fmt.Printf("  [stream] %s price=%d\n", p.Name, p.Price)
		return nil
	})
	fmt.Printf("  found %d books\n", bookCount)

	// Price range search
	fmt.Println()
	fmt.Println("--- Search: price 3000–6000 cents ---")
	var priceMatches int
	_ = svc.SearchProducts(ctx, &SearchRequest{MinPrice: 3000, MaxPrice: 6000}, func(p *Product) error {
		priceMatches++
		fmt.Printf("  [stream] %s price=%d\n", p.Name, p.Price)
		return nil
	})
	fmt.Printf("  found %d products in range\n", priceMatches)

	// ── SERVER STREAMING: WATCH STOCK ────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Server streaming: WatchStock ---")

	watchCtx, watchCancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer watchCancel()

	var stockUpdates []int
	var watchWg sync.WaitGroup
	watchWg.Add(1)
	go func() {
		defer watchWg.Done()
		_ = svc.WatchStock(watchCtx, "p-1", func(stock int) error {
			stockUpdates = append(stockUpdates, stock)
			fmt.Printf("  [watch] p-1 stock changed: %d\n", stock)
			return nil
		})
	}()

	time.Sleep(20 * time.Millisecond)
	p1, _ := svc.GetProduct(ctx, "p-1")
	p1.StockCount = 95
	_, _ = svc.UpdateProduct(ctx, p1)
	p1.StockCount = 90
	_, _ = svc.UpdateProduct(ctx, p1)

	watchCancel()
	watchWg.Wait()
	fmt.Printf("  received %d stock watch events\n", len(stockUpdates))

	// ── ERROR HANDLING ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Error handling ---")

	_, err = svc.CreateProduct(ctx, &Product{Name: "", Price: 100})
	fmt.Printf("  empty name: code=%d\n", codeOf(err))

	_, err = svc.GetProduct(ctx, "nonexistent")
	fmt.Printf("  not found: code=%d\n", codeOf(err))

	// ── TRACING SUMMARY ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  total traced calls: %d\n", traceCounter.Load())
}
