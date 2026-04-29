// EXERCISE 32.1 — Build an HTTP-style middleware chain using the Decorator pattern.
//
// Handler is the core interface. Each middleware wraps a Handler and adds
// cross-cutting behaviour. Chain them together; the final handler executes last.
//
// Run (from the chapter folder):
//   go run ./exercises/01_middleware_chain

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─── Core types ───────────────────────────────────────────────────────────────

type Request struct {
	Method string
	Path   string
	UserID string
	Body   string
}

type Response struct {
	Status int
	Body   string
}

func (r Response) String() string {
	return fmt.Sprintf("HTTP %d: %s", r.Status, r.Body)
}

// Handler is the core interface — every middleware and final handler implements it.
type Handler interface {
	Handle(req Request) Response
}

// HandlerFunc is a function adapter so plain functions satisfy Handler.
type HandlerFunc func(Request) Response

func (f HandlerFunc) Handle(req Request) Response { return f(req) }

// ─── Middlewares (Decorator pattern) ─────────────────────────────────────────

// LoggingMiddleware logs each request and response.
type LoggingMiddleware struct{ next Handler }

func WithLogging(next Handler) Handler { return &LoggingMiddleware{next: next} }

func (m *LoggingMiddleware) Handle(req Request) Response {
	start := time.Now()
	resp := m.next.Handle(req)
	elapsed := time.Since(start).Truncate(time.Microsecond)
	fmt.Printf("  [LOG] %s %s → %d  (%s)\n", req.Method, req.Path, resp.Status, elapsed)
	return resp
}

// AuthMiddleware rejects requests without a user ID.
type AuthMiddleware struct{ next Handler }

func WithAuth(next Handler) Handler { return &AuthMiddleware{next: next} }

func (m *AuthMiddleware) Handle(req Request) Response {
	if strings.TrimSpace(req.UserID) == "" {
		return Response{Status: 401, Body: "Unauthorized"}
	}
	return m.next.Handle(req)
}

// RecoveryMiddleware catches panics and returns a 500.
type RecoveryMiddleware struct{ next Handler }

func WithRecovery(next Handler) Handler { return &RecoveryMiddleware{next: next} }

func (m *RecoveryMiddleware) Handle(req Request) Response {
	defer func() {}() // placeholder; real impl would recover()+log
	return m.next.Handle(req)
}

// RateLimitMiddleware allows at most maxReqs requests total.
type RateLimitMiddleware struct {
	next    Handler
	count   int
	maxReqs int
}

func WithRateLimit(next Handler, maxReqs int) Handler {
	return &RateLimitMiddleware{next: next, maxReqs: maxReqs}
}

func (m *RateLimitMiddleware) Handle(req Request) Response {
	m.count++
	if m.count > m.maxReqs {
		return Response{Status: 429, Body: "Too Many Requests"}
	}
	return m.next.Handle(req)
}

// ─── Chain builder ────────────────────────────────────────────────────────────

// Chain applies middlewares right-to-left so the first in the list runs first.
type MiddlewareFunc func(Handler) Handler

func Chain(h Handler, middlewares ...MiddlewareFunc) Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ─── Business handlers ────────────────────────────────────────────────────────

func greetHandler(req Request) Response {
	return Response{Status: 200, Body: fmt.Sprintf("Hello, %s!", req.UserID)}
}

func echoHandler(req Request) Response {
	return Response{Status: 200, Body: "echo: " + req.Body}
}

func notFoundHandler(req Request) Response {
	return Response{Status: 404, Body: "Not Found: " + req.Path}
}

func routerHandler(req Request) Response {
	switch req.Path {
	case "/greet":
		return HandlerFunc(greetHandler).Handle(req)
	case "/echo":
		return HandlerFunc(echoHandler).Handle(req)
	default:
		return HandlerFunc(notFoundHandler).Handle(req)
	}
}

func main() {
	// Compose the middleware stack:
	//   RateLimit → Auth → Logging → Router
	handler := Chain(
		HandlerFunc(routerHandler),
		func(h Handler) Handler { return WithLogging(h) },
		func(h Handler) Handler { return WithAuth(h) },
		func(h Handler) Handler { return WithRateLimit(h, 4) },
	)

	requests := []Request{
		{Method: "GET", Path: "/greet", UserID: "alice"},
		{Method: "POST", Path: "/echo", UserID: "bob", Body: "hello world"},
		{Method: "GET", Path: "/greet", UserID: ""},       // no auth
		{Method: "GET", Path: "/unknown", UserID: "carol"},
		{Method: "GET", Path: "/greet", UserID: "dave"},   // 5th — rate limited
		{Method: "GET", Path: "/greet", UserID: "eve"},    // 6th — rate limited
	}

	fmt.Println("=== Middleware chain: RateLimit → Auth → Logging → Router ===")
	for _, req := range requests {
		resp := handler.Handle(req)
		fmt.Printf("  → %s\n", resp)
	}
}
