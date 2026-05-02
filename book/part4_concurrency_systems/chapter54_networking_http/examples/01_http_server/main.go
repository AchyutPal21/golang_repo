// FILE: book/part4_concurrency_systems/chapter54_networking_http/examples/01_http_server/main.go
// CHAPTER: 54 — Networking II: HTTP/1.1
// TOPIC: net/http server — ServeMux routing, middleware (logging, auth),
//        JSON request/response, query params, path vars, graceful shutdown.
//
// Run (from the chapter folder):
//   go run ./examples/01_http_server
// Then in another terminal:
//   curl http://localhost:8080/greet?name=Alice
//   curl -X POST http://localhost:8080/echo -H "Content-Type: application/json" -d '{"message":"hello"}'

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ───────────────────────────────────────────────────────────���─────────────────
// HANDLERS
// ──────────────────────────────────���───────────────────────────────��──────────

func greetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	fmt.Fprintf(w, "Hello, %s!\n", name)
}

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Echo      string    `json:"echo"`
	Timestamp time.Time `json:"timestamp"`
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EchoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EchoResponse{
		Echo:      req.Message,
		Timestamp: time.Now().UTC(),
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ────────────────────────────────────────────────────��────────────────────────
// MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────���───────

// loggingMiddleware logs method, path, status code, and duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start).Round(time.Microsecond))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// apiKeyMiddleware requires X-API-Key header.
func apiKeyMiddleware(validKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != validKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────���────────────────────────
// ROUTER + SERVER
// ─────────────────────────────────────────────────────────────────────────────

func buildRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/greet", greetHandler)
	mux.HandleFunc("/echo", echoHandler)
	mux.HandleFunc("/health", healthHandler)

	// Protected route.
	protected := http.NewServeMux()
	protected.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"server": "golang-bible", "version": "1.0"})
	})

	mux.Handle("/admin/", apiKeyMiddleware("secret-key",
		http.StripPrefix("/admin", protected)))

	return loggingMiddleware(mux)
}

// ───────────────────────���─────────────────────────────────────────────────────
// SELF-TEST USING THE REAL SERVER ON A RANDOM PORT
// ───────────────────────────────────────────────────────────────────────────��─

func runRequests(baseURL string) {
	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		desc   string
		method string
		path   string
		body   string
		header map[string]string
	}{
		{"greet no name", "GET", "/greet", "", nil},
		{"greet with name", "GET", "/greet?name=Alice", "", nil},
		{"echo valid", "POST", "/echo", `{"message":"hello"}`, map[string]string{"Content-Type": "application/json"}},
		{"echo wrong method", "GET", "/echo", "", nil},
		{"health", "GET", "/health", "", nil},
		{"admin no key", "GET", "/admin/info", "", nil},
		{"admin valid key", "GET", "/admin/info", "", map[string]string{"X-API-Key": "secret-key"}},
	}

	for _, tt := range tests {
		var bodyReader *strings.Reader
		if tt.body != "" {
			bodyReader = strings.NewReader(tt.body)
		} else {
			bodyReader = strings.NewReader("")
		}
		req, _ := http.NewRequest(tt.method, baseURL+tt.path, bodyReader)
		for k, v := range tt.header {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  %-22s  ERROR: %v\n", tt.desc, err)
			continue
		}
		fmt.Printf("  %-22s  %d\n", tt.desc, resp.StatusCode)
		resp.Body.Close()
	}
}

func main() {
	// Listen on random port for the demo.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	baseURL := "http://" + ln.Addr().String()

	srv := &http.Server{
		Handler:      buildRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start in background.
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("serve error: %v", err)
		}
	}()

	fmt.Printf("=== HTTP/1.1 server on %s ===\n\n", baseURL)
	runRequests(baseURL)

	// Graceful shutdown.
	fmt.Println()
	fmt.Println("--- Graceful shutdown ---")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	fmt.Println("server stopped")
}
