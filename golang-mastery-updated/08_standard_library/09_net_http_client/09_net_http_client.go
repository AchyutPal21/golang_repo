// FILE: 08_standard_library/09_net_http_client.go
// TOPIC: net/http client — requests, custom client, JSON API calls, context
//
// Run: go run 08_standard_library/09_net_http_client.go
//
// NOTE: This file makes real HTTP requests to public APIs.
// It uses httptest for the main demos so no network is required.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: net/http Client")
	fmt.Println("════════════════════════════════════════")

	// ── Setup: local test server (no real network needed) ─────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		fmt.Fprintf(w, "Hello, %s!", name)
	})
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"user":   "alice",
			"score":  42,
		})
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		io.Copy(w, r.Body)
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(w, "finally done")
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	base := server.URL

	// ── http.Get — simple GET request ─────────────────────────────────────
	fmt.Println("\n── http.Get ──")
	resp, err := http.Get(base + "/hello?name=Go")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	// CRITICAL: always close response body — even if you don't read it
	// Not closing leaks the HTTP connection from the connection pool.
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  Status: %d\n  Body: %q\n", resp.StatusCode, body)

	// ── Custom http.Client with timeout ───────────────────────────────────
	// http.DefaultClient has NO timeout — a slow server can block forever!
	// ALWAYS use a custom client with a timeout in production code.
	fmt.Println("\n── Custom client with timeout ──")
	client := &http.Client{
		Timeout: 2 * time.Second,
		// Transport: customize connection pooling, TLS, etc.
	}

	resp2, err := client.Get(base + "/json")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	// Decode JSON response:
	var result map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&result)
	fmt.Printf("  JSON response: %v\n", result)

	// ── Query parameters with url.Values ──────────────────────────────────
	fmt.Println("\n── Query parameters (url.Values) ──")
	params := url.Values{}
	params.Set("name", "Alice & Bob")   // url.Values handles encoding
	params.Set("limit", "10")
	fullURL := base + "/hello?" + params.Encode()
	fmt.Printf("  URL: %s\n", fullURL)
	resp3, _ := client.Get(fullURL)
	defer resp3.Body.Close()
	b3, _ := io.ReadAll(resp3.Body)
	fmt.Printf("  Response: %q\n", b3)

	// ── POST with JSON body ────────────────────────────────────────────────
	fmt.Println("\n── POST with JSON body ──")
	payload := map[string]interface{}{"username": "alice", "action": "login"}
	jsonBytes, _ := json.Marshal(payload)

	resp4, err := client.Post(base+"/echo", "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		defer resp4.Body.Close()
		b4, _ := io.ReadAll(resp4.Body)
		fmt.Printf("  Echo response: %q\n", b4)
	}

	// ── http.NewRequest — full control ────────────────────────────────────
	fmt.Println("\n── http.NewRequest (full control) ──")
	req, _ := http.NewRequest(http.MethodPost, base+"/echo", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer my-token")
	req.Header.Set("X-Request-ID", "abc-123")

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		defer resp5.Body.Close()
		b5, _ := io.ReadAll(resp5.Body)
		fmt.Printf("  Response: %q\n", b5)
	}

	// ── Context for cancellation / timeout ─────────────────────────────────
	fmt.Println("\n── Request with context timeout ──")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/slow", nil)
	_, err = client.Do(req2)
	if err != nil {
		fmt.Printf("  Request timed out (expected): %v\n", err)
	}

	// ── Checking status codes ──────────────────────────────────────────────
	fmt.Println("\n── Status code handling ──")
	fmt.Println(`
  Best practice for checking HTTP status:

  resp, err := client.Do(req)
  if err != nil {
      return fmt.Errorf("http request: %w", err)
  }
  defer resp.Body.Close()

  if resp.StatusCode < 200 || resp.StatusCode >= 300 {
      body, _ := io.ReadAll(resp.Body)
      return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
  }

  // Now safe to decode body
  json.NewDecoder(resp.Body).Decode(&result)
`)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  ALWAYS close resp.Body.Close() — use defer")
	fmt.Println("  NEVER use http.DefaultClient in production — no timeout!")
	fmt.Println("  Custom &http.Client{Timeout: 30*time.Second}")
	fmt.Println("  url.Values for query params — handles URL encoding")
	fmt.Println("  http.NewRequestWithContext for cancellable requests")
	fmt.Println("  Check status codes — 2xx is not always success for your domain")
	fmt.Println("  Drain body before closing: io.Copy(io.Discard, resp.Body)")
}
