// FILE: book/part5_building_backends/chapter76_grpc/examples/01_grpc_basics/main.go
// CHAPTER: 76 — gRPC
// TOPIC: gRPC fundamentals — protobuf message contracts, unary RPC,
//        server/client streaming, interceptors, and error codes.
//        Simulated in-process with Go interfaces so no external tooling is required.
//        See README for real gRPC setup with google.golang.org/grpc.
//
// Run (from the chapter folder):
//   go run ./examples/01_grpc_basics

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PROTO-EQUIVALENT MESSAGE TYPES
// (In real gRPC these are generated from .proto files by protoc-gen-go)
// ─────────────────────────────────────────────────────────────────────────────

type GetProductRequest struct {
	ProductID string
}

type Product struct {
	ID       string
	Name     string
	Price    int // cents
	InStock  bool
}

type ListProductsRequest struct {
	Category string
	Page     int32
	PageSize int32
}

type ListProductsResponse struct {
	Products  []*Product
	NextPage  int32
	Total     int32
}

type CreateProductRequest struct {
	Name    string
	Price   int
	InStock bool
}

// ─────────────────────────────────────────────────────────────────────────────
// STATUS CODES — mirrors google.golang.org/grpc/codes
// ─────────────────────────────────────────────────────────────────────────────

type Code int

const (
	CodeOK             Code = 0
	CodeCancelled      Code = 1
	CodeUnknown        Code = 2
	CodeInvalidArg     Code = 3
	CodeNotFound       Code = 5
	CodeAlreadyExists  Code = 6
	CodePermDenied     Code = 7
	CodeResourceExhausted Code = 8
	CodeInternal       Code = 13
	CodeUnavailable    Code = 14
)

func (c Code) String() string {
	switch c {
	case CodeOK:
		return "OK"
	case CodeCancelled:
		return "Cancelled"
	case CodeInvalidArg:
		return "InvalidArgument"
	case CodeNotFound:
		return "NotFound"
	case CodeAlreadyExists:
		return "AlreadyExists"
	case CodePermDenied:
		return "PermissionDenied"
	case CodeResourceExhausted:
		return "ResourceExhausted"
	case CodeInternal:
		return "Internal"
	case CodeUnavailable:
		return "Unavailable"
	default:
		return "Unknown"
	}
}

type StatusError struct {
	Code    Code
	Message string
}

func (s *StatusError) Error() string {
	return fmt.Sprintf("rpc error: code = %s desc = %s", s.Code, s.Message)
}

func statusErr(code Code, msg string) error {
	return &StatusError{Code: code, Message: msg}
}

func codeOf(err error) Code {
	var s *StatusError
	if errors.As(err, &s) {
		return s.Code
	}
	return CodeUnknown
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE INTERFACE (mirrors generated ServiceServer interface)
// ─────────────────────────────────────────────────────────────────────────────

type ProductServiceServer interface {
	GetProduct(ctx context.Context, req *GetProductRequest) (*Product, error)
	ListProducts(ctx context.Context, req *ListProductsRequest) (*ListProductsResponse, error)
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error)
	// Server-streaming: sends product updates to a channel until ctx is done.
	WatchProducts(ctx context.Context, req *ListProductsRequest, send func(*Product) error) error
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVER IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type productServer struct {
	products map[string]*Product
	nextID   int
}

func newProductServer() *productServer {
	s := &productServer{products: make(map[string]*Product)}
	s.seed()
	return s
}

func (s *productServer) seed() {
	for _, p := range []*Product{
		{ID: "p-1", Name: "Go Programming Book", Price: 3999, InStock: true},
		{ID: "p-2", Name: "Mechanical Keyboard", Price: 12999, InStock: true},
		{ID: "p-3", Name: "USB-C Hub", Price: 4999, InStock: false},
	} {
		s.products[p.ID] = p
	}
	s.nextID = 4
}

func (s *productServer) GetProduct(_ context.Context, req *GetProductRequest) (*Product, error) {
	if req.ProductID == "" {
		return nil, statusErr(CodeInvalidArg, "product_id is required")
	}
	p, ok := s.products[req.ProductID]
	if !ok {
		return nil, statusErr(CodeNotFound, fmt.Sprintf("product %q not found", req.ProductID))
	}
	return p, nil
}

func (s *productServer) ListProducts(_ context.Context, req *ListProductsRequest) (*ListProductsResponse, error) {
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	var all []*Product
	for _, p := range s.products {
		all = append(all, p)
	}
	start := int(req.Page * req.PageSize)
	if start >= len(all) {
		return &ListProductsResponse{Total: int32(len(all))}, nil
	}
	end := start + int(req.PageSize)
	if end > len(all) {
		end = len(all)
	}
	resp := &ListProductsResponse{
		Products: all[start:end],
		Total:    int32(len(all)),
	}
	if end < len(all) {
		resp.NextPage = req.Page + 1
	}
	return resp, nil
}

func (s *productServer) CreateProduct(_ context.Context, req *CreateProductRequest) (*Product, error) {
	if req.Name == "" {
		return nil, statusErr(CodeInvalidArg, "name is required")
	}
	if req.Price < 0 {
		return nil, statusErr(CodeInvalidArg, "price must be non-negative")
	}
	id := fmt.Sprintf("p-%d", s.nextID)
	s.nextID++
	p := &Product{ID: id, Name: req.Name, Price: req.Price, InStock: req.InStock}
	s.products[id] = p
	return p, nil
}

func (s *productServer) WatchProducts(ctx context.Context, req *ListProductsRequest, send func(*Product) error) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	sent := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, p := range s.products {
				if err := send(p); err != nil {
					return err
				}
				sent++
				if sent >= 3 { // stream 3 updates then stop for demo
					return nil
				}
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// INTERCEPTORS — mirrors grpc.UnaryServerInterceptor
// ─────────────────────────────────────────────────────────────────────────────

type UnaryHandler func(ctx context.Context, req any) (any, error)
type UnaryInterceptor func(ctx context.Context, req any, info string, handler UnaryHandler) (any, error)

func chainInterceptors(interceptors []UnaryInterceptor) UnaryInterceptor {
	return func(ctx context.Context, req any, info string, handler UnaryHandler) (any, error) {
		// Build a chain: interceptors[0] wraps interceptors[1] wraps ... wraps handler.
		// Walk backwards so the innermost call is handler, outermost is interceptors[0].
		h := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			inner := h
			h = func(ctx context.Context, req any) (any, error) {
				return interceptor(ctx, req, info, inner)
			}
		}
		return h(ctx, req)
	}
}

func loggingInterceptor(ctx context.Context, req any, info string, handler UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("  [interceptor] %s err=%v dur=%s\n", info, err, dur.Round(time.Microsecond))
	} else {
		fmt.Printf("  [interceptor] %s ok dur=%s\n", info, dur.Round(time.Microsecond))
	}
	return resp, err
}

func authInterceptor(ctx context.Context, req any, info string, handler UnaryHandler) (any, error) {
	token, _ := ctx.Value(ctxKey("token")).(string)
	if token == "" {
		return nil, statusErr(CodePermDenied, "missing auth token")
	}
	return handler(ctx, req)
}

type ctxKey string

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== gRPC Basics (in-process simulation) ===")
	fmt.Println()

	svc := newProductServer()
	ctx := context.WithValue(context.Background(), ctxKey("token"), "bearer-xyz")

	chain := chainInterceptors([]UnaryInterceptor{loggingInterceptor, authInterceptor})

	call := func(method string, handler UnaryHandler, req any) (any, error) {
		return chain(ctx, req, method, handler)
	}

	// ── UNARY RPC: GetProduct ─────────────────────────────────────────────────
	fmt.Println("--- Unary RPC: GetProduct ---")
	resp, err := call("ProductService/GetProduct",
		func(ctx context.Context, req any) (any, error) {
			return svc.GetProduct(ctx, req.(*GetProductRequest))
		},
		&GetProductRequest{ProductID: "p-1"},
	)
	if err == nil {
		p := resp.(*Product)
		fmt.Printf("  found: id=%s name=%q price=%d inStock=%v\n", p.ID, p.Name, p.Price, p.InStock)
	}

	// ── NOT FOUND ERROR ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Error handling: not found ---")
	_, err = call("ProductService/GetProduct",
		func(ctx context.Context, req any) (any, error) {
			return svc.GetProduct(ctx, req.(*GetProductRequest))
		},
		&GetProductRequest{ProductID: "p-999"},
	)
	fmt.Printf("  error code=%s\n", codeOf(err))

	// ── INVALID ARGUMENT ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Error handling: invalid argument ---")
	_, err = call("ProductService/CreateProduct",
		func(ctx context.Context, req any) (any, error) {
			return svc.CreateProduct(ctx, req.(*CreateProductRequest))
		},
		&CreateProductRequest{Name: "", Price: 100},
	)
	fmt.Printf("  error code=%s msg=%v\n", codeOf(err), err)

	// ── AUTH INTERCEPTOR ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Auth interceptor: missing token ---")
	noAuthCtx := context.Background()
	chainNoAuth := chainInterceptors([]UnaryInterceptor{loggingInterceptor, authInterceptor})
	_, err = chainNoAuth(noAuthCtx, &GetProductRequest{ProductID: "p-1"}, "ProductService/GetProduct",
		func(ctx context.Context, req any) (any, error) {
			return svc.GetProduct(ctx, req.(*GetProductRequest))
		})
	fmt.Printf("  error code=%s\n", codeOf(err))

	// ── LIST PRODUCTS ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Unary RPC: ListProducts (page 0, size 2) ---")
	resp, _ = call("ProductService/ListProducts",
		func(ctx context.Context, req any) (any, error) {
			return svc.ListProducts(ctx, req.(*ListProductsRequest))
		},
		&ListProductsRequest{PageSize: 2, Page: 0},
	)
	lr := resp.(*ListProductsResponse)
	fmt.Printf("  total=%d returned=%d nextPage=%d\n", lr.Total, len(lr.Products), lr.NextPage)
	for _, p := range lr.Products {
		fmt.Printf("    id=%s name=%q\n", p.ID, p.Name)
	}

	// ── SERVER STREAMING ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Server streaming: WatchProducts ---")
	streamCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	var streamReceived int
	err = svc.WatchProducts(streamCtx, &ListProductsRequest{}, func(p *Product) error {
		streamReceived++
		fmt.Printf("  [stream] received id=%s name=%q\n", p.ID, p.Name)
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Printf("  stream error: %v\n", err)
	}
	fmt.Printf("  stream ended; received %d updates\n", streamReceived)

	// ── PROTO WIRE FORMAT EXPLANATION ─────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Protobuf vs JSON (conceptual) ---")
	productJSON := `{"id":"p-1","name":"Go Programming Book","price":3999,"in_stock":true}`
	// Proto encoding is binary; field numbers replace names.
	// field 1 (id=string), field 2 (name=string), field 3 (price=int32), field 4 (in_stock=bool)
	protoApprox := fmt.Sprintf("[0x0a %d bytes id][0x12 %d bytes name][0x18 0xaf1f][0x20 0x01]",
		len("p-1"), len("Go Programming Book"))
	fmt.Printf("  JSON  (%d bytes): %s\n", len(productJSON), productJSON)
	fmt.Printf("  Proto (~%d bytes): %s\n", 4+3+4+18+2+1, protoApprox)
	fmt.Printf("  Proto is ~%.0f%% smaller for this message\n",
		float64(len(productJSON)-30)/float64(len(productJSON))*100)

	// ── KEY gRPC CONCEPTS SUMMARY ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Key gRPC concepts ---")
	concepts := []string{
		"Unary RPC:          single request → single response (like HTTP/1.1 POST)",
		"Server streaming:   single request → stream of responses (like SSE)",
		"Client streaming:   stream of requests → single response (like chunked upload)",
		"Bidirectional:      stream of requests ↔ stream of responses (like WebSocket)",
		"Status codes:       structured errors with Code + message (not HTTP status)",
		"Interceptors:       middleware for auth, logging, metrics, tracing",
		"Deadlines:          context.WithDeadline propagates via HTTP/2 HEADERS",
		"Reflection:         server can describe its own schema (like GraphQL introspection)",
	}
	for _, c := range concepts {
		parts := strings.SplitN(c, ":", 2)
		fmt.Printf("  %-24s: %s\n", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
}
