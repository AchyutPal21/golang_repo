// EXERCISE 23.1 — HTTP-like middleware stack using embedding.
//
// Build a Handler interface and three middleware structs that embed
// each other to form a chain: LoggingMiddleware → AuthMiddleware → Handler.
//
// Run (from the chapter folder):
//   go run ./exercises/01_middleware_stack

package main

import "fmt"

type Request struct {
	Path   string
	UserID int
}

type Response struct {
	Status int
	Body   string
}

type Handler interface {
	Handle(r Request) Response
}

// HandlerFunc adapts a func to Handler.
type HandlerFunc func(Request) Response

func (f HandlerFunc) Handle(r Request) Response { return f(r) }

// LoggingMiddleware logs every request.
type LoggingMiddleware struct {
	Next Handler
}

func (l *LoggingMiddleware) Handle(r Request) Response {
	fmt.Printf("[LOG] %s (user=%d)\n", r.Path, r.UserID)
	resp := l.Next.Handle(r)
	fmt.Printf("[LOG] → %d\n", resp.Status)
	return resp
}

// AuthMiddleware rejects requests with userID <= 0.
type AuthMiddleware struct {
	Next Handler
}

func (a *AuthMiddleware) Handle(r Request) Response {
	if r.UserID <= 0 {
		fmt.Println("[AUTH] rejected")
		return Response{Status: 401, Body: "unauthorized"}
	}
	return a.Next.Handle(r)
}

// Chain wraps handler with middleware outermost-last.
func Chain(h Handler, middleware ...func(Handler) Handler) Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func main() {
	core := HandlerFunc(func(r Request) Response {
		return Response{Status: 200, Body: "Hello, user " + fmt.Sprint(r.UserID)}
	})

	stack := Chain(core,
		func(next Handler) Handler { return &LoggingMiddleware{Next: next} },
		func(next Handler) Handler { return &AuthMiddleware{Next: next} },
	)

	fmt.Println("=== authenticated request ===")
	resp := stack.Handle(Request{Path: "/api/data", UserID: 42})
	fmt.Println("body:", resp.Body)

	fmt.Println()
	fmt.Println("=== unauthenticated request ===")
	resp = stack.Handle(Request{Path: "/api/data", UserID: 0})
	fmt.Println("body:", resp.Body)
}
