// FILE: book/part7_capstone_projects/capstone_h_api_gateway/main.go
// CAPSTONE H — API Gateway
// Simulates: route matching, round-robin load balancing, circuit breaker,
// rate limiting, request transformation, auth enforcement, and metrics.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_h_api_gateway

package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// REQUEST / RESPONSE
// ─────────────────────────────────────────────────────────────────────────────

type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	APIKey  string
}

type Response struct {
	Status  int
	Body    string
	Headers map[string]string
}

func (r Response) String() string {
	return fmt.Sprintf("HTTP %d: %s", r.Status, r.Body)
}

// ─────────────────────────────────────────────────────────────────────────────
// CIRCUIT BREAKER
// ─────────────────────────────────────────────────────────────────────────────

type cbState int

const (
	cbClosed   cbState = iota // normal operation
	cbOpen                    // failing — reject fast
	cbHalfOpen                // probe one request
)

type circuitBreaker struct {
	mu           sync.Mutex
	state        cbState
	failures     int
	maxFailures  int
	successCount int
	openedAt     time.Time
	cooldown     time.Duration
}

func newCircuitBreaker(maxFailures int, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{maxFailures: maxFailures, cooldown: cooldown}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.openedAt) > cb.cooldown {
			cb.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		return true
	}
	return false
}

func (cb *circuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = cbClosed
}

func (cb *circuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.maxFailures || cb.state == cbHalfOpen {
		cb.state = cbOpen
		cb.openedAt = time.Now()
	}
}

func (cb *circuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half-open"
	}
	return "unknown"
}

// ─────────────────────────────────────────────────────────────────────────────
// UPSTREAM BACKEND (simulated)
// ─────────────────────────────────────────────────────────────────────────────

type Backend struct {
	Name       string
	FailRate   float64 // 0–1 simulated failure probability
	LatencyMs  int
	cb         *circuitBreaker
	reqCount   atomic.Int64
	errCount   atomic.Int64
}

func newBackend(name string, failRate float64, latencyMs int) *Backend {
	return &Backend{
		Name:      name,
		FailRate:  failRate,
		LatencyMs: latencyMs,
		cb:        newCircuitBreaker(3, 100*time.Millisecond),
	}
}

func (b *Backend) Call(req Request) (Response, error) {
	if !b.cb.Allow() {
		return Response{}, fmt.Errorf("circuit open: %s", b.Name)
	}
	b.reqCount.Add(1)
	// Simulate latency
	_ = b.LatencyMs // in a real proxy: time.Sleep(time.Duration(b.LatencyMs) * time.Millisecond)

	// Simulate failure
	if b.FailRate > 0 && float64(b.reqCount.Load()%10) < b.FailRate*10 {
		b.errCount.Add(1)
		b.cb.OnFailure()
		return Response{}, fmt.Errorf("upstream error from %s", b.Name)
	}
	b.cb.OnSuccess()
	return Response{
		Status: 200,
		Body:   fmt.Sprintf(`{"backend":"%s","path":"%s"}`, b.Name, req.Path),
		Headers: map[string]string{"X-Served-By": b.Name},
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LOAD BALANCER (round-robin)
// ─────────────────────────────────────────────────────────────────────────────

type Upstream struct {
	Name     string
	backends []*Backend
	counter  atomic.Uint64
}

func newUpstream(name string, backends ...*Backend) *Upstream {
	return &Upstream{Name: name, backends: backends}
}

func (u *Upstream) Forward(req Request) (Response, error) {
	if len(u.backends) == 0 {
		return Response{}, errors.New("no backends configured")
	}
	// Round-robin with circuit breaker skip
	n := len(u.backends)
	start := int(u.counter.Add(1)) % n
	for i := 0; i < n; i++ {
		b := u.backends[(start+i)%n]
		resp, err := b.Call(req)
		if err == nil {
			return resp, nil
		}
		if strings.Contains(err.Error(), "circuit open") {
			continue // try next backend
		}
		return Response{}, err
	}
	return Response{Status: 503, Body: "all backends unavailable"}, nil
}

func (u *Upstream) Stats() string {
	var parts []string
	for _, b := range u.backends {
		parts = append(parts, fmt.Sprintf("%s(req=%d err=%d cb=%s)",
			b.Name, b.reqCount.Load(), b.errCount.Load(), b.cb.State()))
	}
	return strings.Join(parts, " | ")
}

// ─────────────────────────────────────────────────────────────────────────────
// RATE LIMITER (token bucket per API key)
// ─────────────────────────────────────────────────────────────────────────────

type rlBucket struct {
	tokens   float64
	cap      float64
	rate     float64
	lastFill time.Time
	mu       sync.Mutex
}

func (b *rlBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed := time.Since(b.lastFill).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.cap {
		b.tokens = b.cap
	}
	b.lastFill = time.Now()
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

type gwRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rlBucket
	cap     float64
	rate    float64
}

func newGWRateLimiter(cap, rate float64) *gwRateLimiter {
	return &gwRateLimiter{buckets: map[string]*rlBucket{}, cap: cap, rate: rate}
}

func (rl *gwRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &rlBucket{tokens: rl.cap, cap: rl.cap, rate: rl.rate, lastFill: time.Now()}
		rl.buckets[key] = b
	}
	rl.mu.Unlock()
	return b.allow()
}

// ─────────────────────────────────────────────────────────────────────────────
// ROUTE TABLE
// ─────────────────────────────────────────────────────────────────────────────

type Route struct {
	Prefix     string
	Upstream   *Upstream
	AuthRequired bool
}

type RouteTable struct {
	routes []Route // sorted longest-prefix first
}

func (rt *RouteTable) Add(prefix string, upstream *Upstream, authRequired bool) {
	rt.routes = append(rt.routes, Route{Prefix: prefix, Upstream: upstream, AuthRequired: authRequired})
	sort.Slice(rt.routes, func(i, j int) bool {
		return len(rt.routes[i].Prefix) > len(rt.routes[j].Prefix)
	})
}

func (rt *RouteTable) Match(path string) (Route, bool) {
	for _, r := range rt.routes {
		if strings.HasPrefix(path, r.Prefix) {
			return r, true
		}
	}
	return Route{}, false
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH STORE
// ─────────────────────────────────────────────────────────────────────────────

type apiKeyStore struct {
	keys map[string]string // apiKey → ownerID
}

func newAPIKeyStore() *apiKeyStore {
	return &apiKeyStore{keys: map[string]string{
		"key-admin-001": "admin",
		"key-user-002":  "user-42",
		"key-user-003":  "user-99",
	}}
}

func (s *apiKeyStore) Validate(key string) (string, bool) {
	owner, ok := s.keys[key]
	return owner, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// METRICS (simple latency histogram)
// ─────────────────────────────────────────────────────────────────────────────

type gatewayMetrics struct {
	mu       sync.Mutex
	requests map[string]int64
	errors   map[string]int64
	latencies map[string][]int // ms samples per route prefix
}

func newGatewayMetrics() *gatewayMetrics {
	return &gatewayMetrics{
		requests:  map[string]int64{},
		errors:    map[string]int64{},
		latencies: map[string][]int{},
	}
}

func (m *gatewayMetrics) Record(prefix string, latencyMs int, isErr bool) {
	m.mu.Lock()
	m.requests[prefix]++
	if isErr {
		m.errors[prefix]++
	}
	m.latencies[prefix] = append(m.latencies[prefix], latencyMs)
	m.mu.Unlock()
}

func (m *gatewayMetrics) P99(prefix string) int {
	m.mu.Lock()
	lats := append([]int{}, m.latencies[prefix]...)
	m.mu.Unlock()
	if len(lats) == 0 {
		return 0
	}
	sort.Ints(lats)
	idx := int(float64(len(lats)) * 0.99)
	if idx >= len(lats) {
		idx = len(lats) - 1
	}
	return lats[idx]
}

func (m *gatewayMetrics) Print() {
	m.mu.Lock()
	prefixes := make([]string, 0, len(m.requests))
	for p := range m.requests {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	m.mu.Unlock()

	fmt.Printf("  %-20s  %8s  %8s  %8s\n", "Route", "Requests", "Errors", "P99 ms")
	fmt.Printf("  %s\n", strings.Repeat("-", 55))
	for _, p := range prefixes {
		m.mu.Lock()
		req := m.requests[p]
		err := m.errors[p]
		m.mu.Unlock()
		fmt.Printf("  %-20s  %8d  %8d  %8d\n", p, req, err, m.P99(p))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GATEWAY
// ─────────────────────────────────────────────────────────────────────────────

type Gateway struct {
	routes  *RouteTable
	rl      *gwRateLimiter
	auth    *apiKeyStore
	metrics *gatewayMetrics
}

func NewGateway() *Gateway {
	return &Gateway{
		routes:  &RouteTable{},
		rl:      newGWRateLimiter(20, 10),
		auth:    newAPIKeyStore(),
		metrics: newGatewayMetrics(),
	}
}

func (g *Gateway) Register(prefix string, upstream *Upstream, authRequired bool) {
	g.routes.Add(prefix, upstream, authRequired)
}

func (g *Gateway) Handle(req Request) Response {
	start := time.Now()

	// Inject request ID
	if req.Headers == nil {
		req.Headers = map[string]string{}
	}
	req.Headers["X-Request-ID"] = fmt.Sprintf("gw-%d", time.Now().UnixNano())

	// Route match
	route, ok := g.routes.Match(req.Path)
	if !ok {
		return Response{Status: 404, Body: "no route found for " + req.Path}
	}

	// Auth check
	if route.AuthRequired {
		if _, valid := g.auth.Validate(req.APIKey); !valid {
			return Response{Status: 401, Body: "invalid API key"}
		}
	}

	// Rate limit
	if !g.rl.Allow(req.APIKey) {
		return Response{Status: 429, Body: "rate limit exceeded"}
	}

	// Forward
	resp, err := route.Upstream.Forward(req)
	latencyMs := int(time.Since(start).Milliseconds())
	isErr := err != nil || resp.Status >= 500
	g.metrics.Record(route.Prefix, latencyMs, isErr)

	if err != nil {
		return Response{Status: 502, Body: "bad gateway: " + err.Error()}
	}
	return resp
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone H: API Gateway ===")
	fmt.Println()

	// ── SETUP UPSTREAMS ───────────────────────────────────────────────────────
	ordersUpstream := newUpstream("orders",
		newBackend("orders-1", 0.0, 5),
		newBackend("orders-2", 0.0, 8),
	)
	usersUpstream := newUpstream("users",
		newBackend("users-1", 0.0, 3),
	)
	catalogUpstream := newUpstream("catalog",
		newBackend("catalog-1", 0.3, 10), // 30% failure rate
		newBackend("catalog-2", 0.0, 6),  // fallback
	)

	gw := NewGateway()
	gw.Register("/api/orders", ordersUpstream, true)
	gw.Register("/api/users", usersUpstream, true)
	gw.Register("/api/catalog", catalogUpstream, false)
	gw.Register("/healthz", newUpstream("health", newBackend("health", 0, 1)), false)

	// ── NORMAL REQUESTS ───────────────────────────────────────────────────────
	fmt.Println("--- Normal requests ---")
	requests := []Request{
		{Method: "GET", Path: "/api/orders/123", APIKey: "key-user-002"},
		{Method: "GET", Path: "/api/users/me", APIKey: "key-admin-001"},
		{Method: "GET", Path: "/api/catalog/products", APIKey: ""},
		{Method: "GET", Path: "/healthz", APIKey: ""},
	}
	for _, r := range requests {
		resp := gw.Handle(r)
		fmt.Printf("  %-35s → %s\n", r.Method+" "+r.Path, resp)
	}
	fmt.Println()

	// ── AUTH FAILURES ─────────────────────────────────────────────────────────
	fmt.Println("--- Auth enforcement ---")
	resp := gw.Handle(Request{Method: "GET", Path: "/api/orders/1", APIKey: "bad-key"})
	fmt.Printf("  Bad API key:    %s\n", resp)
	resp = gw.Handle(Request{Method: "GET", Path: "/api/orders/1", APIKey: ""})
	fmt.Printf("  Missing key:    %s\n", resp)
	fmt.Println()

	// ── ROUTE NOT FOUND ───────────────────────────────────────────────────────
	fmt.Println("--- Route matching ---")
	resp = gw.Handle(Request{Method: "GET", Path: "/api/unknown", APIKey: "key-user-002"})
	fmt.Printf("  Unknown route:  %s\n", resp)
	fmt.Println()

	// ── CIRCUIT BREAKER ───────────────────────────────────────────────────────
	fmt.Println("--- Circuit breaker (catalog has 30% failure rate) ---")
	for i := 0; i < 15; i++ {
		resp := gw.Handle(Request{Method: "GET", Path: "/api/catalog/items"})
		fmt.Printf("  req %2d: %s\n", i+1, resp)
	}
	fmt.Printf("  Upstream stats: %s\n", catalogUpstream.Stats())
	fmt.Println()

	// ── RATE LIMITING ─────────────────────────────────────────────────────────
	fmt.Println("--- Rate limiter (burst=20, 10/s) ---")
	limited := 0
	for i := 0; i < 25; i++ {
		resp := gw.Handle(Request{Method: "GET", Path: "/api/orders/1", APIKey: "key-user-003"})
		if resp.Status == 429 {
			limited++
		}
	}
	fmt.Printf("  25 rapid requests → %d rate-limited\n", limited)
	fmt.Println()

	// ── METRICS ───────────────────────────────────────────────────────────────
	fmt.Println("--- Gateway metrics ---")
	gw.metrics.Print()
}
