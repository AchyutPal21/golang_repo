// FILE: book/part4_concurrency_systems/chapter54_networking_http/examples/02_http_client/main.go
// CHAPTER: 54 — Networking II: HTTP/1.1
// TOPIC: http.Client configuration — custom Transport, timeouts,
//        retries with backoff, concurrent requests, response body handling.
//
// Run (from the chapter folder):
//   go run ./examples/02_http_client

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM CLIENT BUILDER
// ─────────────────────────────────────────────────────────────────────────────

func newClient(totalTimeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second, // TCP connect timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   totalTimeout,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY WITH EXPONENTIAL BACKOFF
// ─────────────────────────────────────────────────────────────────────────────

func withRetry(ctx context.Context, client *http.Client, maxAttempts int, backoff time.Duration,
	fn func() (*http.Request, error)) (*http.Response, error) {

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			wait := backoff * time.Duration(1<<(attempt-1)) // 2^(attempt-1) × backoff
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := fn()
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 5xx.
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("all %d attempts failed: %w", maxAttempts, lastErr)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: basic GET and JSON decode
// ─────────────────────────────────────────────────────────────────────────────

func demoBasicRequest() {
	fmt.Println("=== Basic GET + JSON decode ===")

	// Use httptest.Server for a hermetic demo.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   1,
			"name": r.URL.Query().Get("name"),
		})
	}))
	defer ts.Close()

	client := newClient(5 * time.Second)
	resp, err := client.Get(ts.URL + "?name=Alice")
	if err != nil {
		fmt.Println("  GET error:", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("  status: %d  body: %v\n", resp.StatusCode, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: POST with JSON body
// ─────────────────────────────────────────────────────────────────────────────

func demoPostJSON() {
	fmt.Println()
	fmt.Println("=== POST JSON ===")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		body["echoed"] = true
		json.NewEncoder(w).Encode(body)
	}))
	defer ts.Close()

	client := newClient(5 * time.Second)
	body := strings.NewReader(`{"key":"value","count":42}`)
	resp, err := client.Post(ts.URL, "application/json", body)
	if err != nil {
		fmt.Println("  POST error:", err)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("  status: %d  body: %s\n", resp.StatusCode, strings.TrimSpace(string(data)))
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: retry on 503
// ─────────────────────────────────────────────────────────────────────────────

func demoRetry() {
	fmt.Println()
	fmt.Println("=== Retry on 503 ===")

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	client := newClient(10 * time.Second)
	ctx := context.Background()
	resp, err := withRetry(ctx, client, 5, 10*time.Millisecond, func() (*http.Request, error) {
		return http.NewRequest("GET", ts.URL, nil)
	})
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("  succeeded on attempt %d: %s\n", attempts, string(data))
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 4: concurrent requests
// ─────────────────────────────────────────────────────────────────────────────

func demoConcurrentRequests() {
	fmt.Println()
	fmt.Println("=== Concurrent requests ===")

	mu := sync.Mutex{}
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		id := count
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintf(w, "req-%d", id)
	}))
	defer ts.Close()

	client := newClient(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	responses := make([]string, 10)

	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
			resp, err := client.Do(req)
			if err != nil {
				responses[idx] = "err"
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			responses[idx] = string(data)
		}(i)
	}

	wg.Wait()
	fmt.Printf("  %d responses: %v\n", len(responses), responses[:5])
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 5: response body must always be read and closed
// ─────────────────────────────────────────────────────────────────────────────

func demoBodyHandling() {
	fmt.Println()
	fmt.Println("=== Response body: always read + close ===")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := range 5 {
			fmt.Fprintf(w, "line %d\n", i)
		}
	}))
	defer ts.Close()

	client := newClient(5 * time.Second)
	resp, err := client.Get(ts.URL)
	if err != nil {
		fmt.Println("  GET error:", err)
		return
	}
	defer resp.Body.Close()

	// Read body — if you skip this, the connection cannot be reused.
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("  read %d bytes from body\n", len(data))
	fmt.Println("  (always call io.ReadAll or io.Copy + resp.Body.Close())")
}

func main() {
	demoBasicRequest()
	demoPostJSON()
	demoRetry()
	demoConcurrentRequests()
	demoBodyHandling()
}
