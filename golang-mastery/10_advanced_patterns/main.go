package main

// =============================================================================
// MODULE 10: ADVANCED PATTERNS — Senior-level Go
// =============================================================================
// Run: go run 10_advanced_patterns/main.go
//
// Topics:
//   - Context package (cancellation, deadlines, values)
//   - Testing patterns (table-driven, mocks, benchmarks)
//   - Common design patterns in Go
//   - Reflection basics
//   - HTTP server patterns
//   - Production best practices
// =============================================================================

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// CONTEXT — Cancellation, deadlines, request-scoped values
// =============================================================================
// context.Context is passed as the FIRST parameter to functions that:
//   - Make network calls
//   - Run long computations
//   - Read from databases
//   - Spawn goroutines
//
// Rules:
//   1. Always accept ctx context.Context as first parameter
//   2. Never store context in a struct (pass it, don't store it)
//   3. Pass context down the call chain
//   4. Cancel contexts when done — always defer cancel()

// Simulates a slow database query that respects context
func slowDBQuery(ctx context.Context, query string) (string, error) {
	// Simulate network delay
	resultCh := make(chan string, 1)
	go func() {
		time.Sleep(200 * time.Millisecond) // pretend DB call
		resultCh <- "result: " + query
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		// ctx.Err() returns context.Canceled or context.DeadlineExceeded
		return "", ctx.Err()
	}
}

// Context with values — for request-scoped data (user ID, trace ID, etc.)
type contextKey string

const (
	userIDKey  contextKey = "userID"
	requestKey contextKey = "requestID"
)

func getUserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}

func processRequest(ctx context.Context) {
	if userID, ok := getUserID(ctx); ok {
		fmt.Printf("Processing request for user %d\n", userID)
	}
	if reqID, ok := ctx.Value(requestKey).(string); ok {
		fmt.Printf("Request ID: %s\n", reqID)
	}
}

// =============================================================================
// DESIGN PATTERNS IN GO
// =============================================================================

// ---- PATTERN 1: SINGLETON ----
type singleton struct {
	data string
}

var (
	instance *singleton
	once     sync.Once
)

func GetInstance() *singleton {
	once.Do(func() {
		instance = &singleton{data: "initialized"}
		fmt.Println("[Singleton] Created")
	})
	return instance
}

// ---- PATTERN 2: OBSERVER ----
type EventType string

const (
	EventCreated EventType = "created"
	EventUpdated EventType = "updated"
	EventDeleted EventType = "deleted"
)

type Event struct {
	Type    EventType
	Payload interface{}
}

type Handler func(Event)

type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[EventType][]Handler)}
}

func (eb *EventBus) Subscribe(eventType EventType, h Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], h)
}

func (eb *EventBus) Publish(e Event) {
	eb.mu.RLock()
	handlers := eb.handlers[e.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		go h(e) // async dispatch
	}
}

// ---- PATTERN 3: MIDDLEWARE / DECORATOR ----
type HandlerFunc func(string) string

func Timing(next HandlerFunc) HandlerFunc {
	return func(req string) string {
		start := time.Now()
		result := next(req)
		fmt.Printf("[timing] took %v\n", time.Since(start))
		return result
	}
}

func Logging(name string, next HandlerFunc) HandlerFunc {
	return func(req string) string {
		fmt.Printf("[log] %s: handling %q\n", name, req)
		result := next(req)
		fmt.Printf("[log] %s: result %q\n", name, result)
		return result
	}
}

func Uppercase(next HandlerFunc) HandlerFunc {
	return func(req string) string {
		return strings.ToUpper(next(req))
	}
}

// ---- PATTERN 4: FUNCTIONAL OPTIONS (revisited as production pattern) ----
type HTTPClient struct {
	baseURL    string
	timeout    time.Duration
	maxRetries int
	headers    map[string]string
}

type HTTPClientOption func(*HTTPClient)

func WithBaseURL(url string) HTTPClientOption {
	return func(c *HTTPClient) { c.baseURL = url }
}

func WithTimeout2(d time.Duration) HTTPClientOption {
	return func(c *HTTPClient) { c.timeout = d }
}

func WithMaxRetries(n int) HTTPClientOption {
	return func(c *HTTPClient) { c.maxRetries = n }
}

func WithHeader(key, val string) HTTPClientOption {
	return func(c *HTTPClient) { c.headers[key] = val }
}

func NewHTTPClient(opts ...HTTPClientOption) *HTTPClient {
	c := &HTTPClient{
		baseURL:    "http://localhost",
		timeout:    30 * time.Second,
		maxRetries: 3,
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *HTTPClient) String() string {
	return fmt.Sprintf("HTTPClient{url=%s timeout=%v retries=%d headers=%v}",
		c.baseURL, c.timeout, c.maxRetries, c.headers)
}

// ---- PATTERN 5: CIRCUIT BREAKER ----
type CircuitState int

const (
	Closed   CircuitState = iota // Normal — requests pass through
	Open                         // Failed — requests are rejected
	HalfOpen                     // Testing — one request allowed
)

type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	threshold    int
	lastFailTime time.Time
	timeout      time.Duration
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     Closed,
		threshold: threshold,
		timeout:   timeout,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Open:
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = HalfOpen
			fmt.Println("[CB] half-open: testing...")
		} else {
			return fmt.Errorf("circuit breaker open")
		}
	case HalfOpen:
		// allow one request through
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()
		if cb.failures >= cb.threshold || cb.state == HalfOpen {
			cb.state = Open
			fmt.Printf("[CB] opened! failures=%d\n", cb.failures)
		}
		return err
	}

	// Success — reset
	cb.failures = 0
	cb.state = Closed
	return nil
}

// ---- PATTERN 6: GRACEFUL SHUTDOWN ----
// (See note — requires HTTP, shown as pattern)
/*
func startServer() {
    srv := &http.Server{Addr: ":8080"}
    go srv.ListenAndServe()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
*/

// =============================================================================
// REFLECTION — inspect types at runtime
// =============================================================================
// Reflection is powerful but slow — use only when necessary.
// Common uses: serialization, ORM, dependency injection, testing

func inspectValue(v interface{}) {
	t := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	fmt.Printf("Type: %s, Kind: %s\n", t, t.Kind())

	switch t.Kind() {
	case reflect.Struct:
		fmt.Printf("Fields: %d\n", t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldVal := val.Field(i)
			fmt.Printf("  %s %s = %v (tag: %q)\n",
				field.Name, field.Type, fieldVal.Interface(), field.Tag)
		}
	case reflect.Slice:
		fmt.Printf("Length: %d\n", val.Len())
	case reflect.Map:
		fmt.Printf("Keys: %v\n", val.MapKeys())
	}
}

// Generic deep equal using reflection
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// Inspect function signature using reflection
func inspectFunc(fn interface{}) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		fmt.Println("not a function")
		return
	}
	fmt.Printf("Function: %d params, %d returns\n", t.NumIn(), t.NumOut())
	for i := 0; i < t.NumIn(); i++ {
		fmt.Printf("  param[%d]: %s\n", i, t.In(i))
	}
	for i := 0; i < t.NumOut(); i++ {
		fmt.Printf("  return[%d]: %s\n", i, t.Out(i))
	}
}

// =============================================================================
// TESTING PATTERNS (shown here — run with go test)
// =============================================================================
// Tests live in *_test.go files in the same package.
// Test functions: func TestXxx(t *testing.T)
// Benchmark:      func BenchmarkXxx(b *testing.B)
// Example:        func ExampleXxx()
// Setup:          func TestMain(m *testing.M)

/*
--- Table-Driven Tests (most important Go testing pattern) ---

func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 2, 3, 5},
        {"negative", -2, -3, -5},
        {"mixed", -2, 3, 1},
        {"zeros", 0, 0, 0},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got := Add(tc.a, tc.b)
            if got != tc.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.expected)
            }
        })
    }
}

--- Subtests ---
    t.Run("subtest name", func(t *testing.T) { ... })
    go test -run TestAdd/positive   // run specific subtest

--- Test helpers ---
    t.Helper()   // marks function as helper (better error line numbers)
    t.Skip("reason")   // skip a test
    t.Fatal("msg")     // stop test immediately
    t.Error("msg")     // mark failed, continue

--- Benchmarks ---
func BenchmarkAdd(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Add(2, 3)
    }
}
go test -bench=. -benchmem ./...

--- Mocking interfaces ---
type UserRepo interface {
    GetUser(id int) (*User, error)
    SaveUser(u *User) error
}

type MockUserRepo struct {
    users map[int]*User
}

func (m *MockUserRepo) GetUser(id int) (*User, error) {
    if u, ok := m.users[id]; ok { return u, nil }
    return nil, errors.New("not found")
}
// ...
*/

// =============================================================================
// PRODUCTION BEST PRACTICES — Senior-level concerns
// =============================================================================

// 1. Always pass context to I/O operations
// 2. Use structured logging (zerolog, zap, slog)
// 3. Handle all errors — never discard
// 4. Use -race flag in tests and CI
// 5. Profile with pprof before optimizing
// 6. Prefer table-driven tests
// 7. Keep interfaces small (1-3 methods)
// 8. Accept interfaces, return concrete types
// 9. Prefer sync.Mutex over channels for shared state
// 10. Use context for cancellation, not channels

// =============================================================================
// RUNTIME PACKAGE — introspect Go runtime
// =============================================================================

func runtimeInfo() {
	fmt.Println("Go version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Println("Arch:", runtime.GOARCH)
	fmt.Println("NumCPU:", runtime.NumCPU())
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0)) // 0 = query, don't set
	fmt.Println("NumGoroutine:", runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc: %v KB\n", m.Alloc/1024)
	fmt.Printf("TotalAlloc: %v KB\n", m.TotalAlloc/1024)
	fmt.Printf("HeapObjects: %v\n", m.HeapObjects)
	fmt.Printf("GC cycles: %v\n", m.NumGC)
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 10: ADVANCED PATTERNS ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Context
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Context ---")

	// Background — root context (never cancelled)
	ctx := context.Background()

	// WithTimeout — auto-cancels after duration
	ctxTimeout, cancel1 := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel1()

	result, err := slowDBQuery(ctxTimeout, "SELECT * FROM users")
	if err != nil {
		fmt.Println("Query error:", err)
	} else {
		fmt.Println("Query result:", result)
	}

	// Context that will timeout
	ctxShort, cancel2 := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel2()

	_, err2 := slowDBQuery(ctxShort, "SELECT * FROM logs")
	if err2 != nil {
		fmt.Println("Short context error:", err2) // deadline exceeded
	}

	// WithCancel — manual cancellation
	ctxCancel, cancel3 := context.WithCancel(ctx)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel3() // cancel after 100ms
	}()
	<-ctxCancel.Done()
	fmt.Println("Context cancelled:", ctxCancel.Err())

	// WithValue — pass request-scoped data
	ctxValues := context.WithValue(ctx, userIDKey, 42)
	ctxValues = context.WithValue(ctxValues, requestKey, "req-abc123")
	processRequest(ctxValues)

	// WithDeadline — cancels at specific time
	deadline := time.Now().Add(1 * time.Second)
	ctxDeadline, cancel4 := context.WithDeadline(ctx, deadline)
	defer cancel4()
	dl, _ := ctxDeadline.Deadline()
	fmt.Printf("Deadline: %s from now\n", time.Until(dl).Round(time.Millisecond))

	// -------------------------------------------------------------------------
	// SECTION 2: Singleton pattern
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Singleton ---")

	s1 := GetInstance()
	s2 := GetInstance()
	s3 := GetInstance()
	fmt.Println("Same instance:", s1 == s2 && s2 == s3)
	fmt.Println("Data:", s1.data)

	// -------------------------------------------------------------------------
	// SECTION 3: Observer / Event Bus
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Observer / Event Bus ---")

	bus := NewEventBus()

	bus.Subscribe(EventCreated, func(e Event) {
		fmt.Println("Handler 1 [created]:", e.Payload)
	})
	bus.Subscribe(EventCreated, func(e Event) {
		fmt.Println("Handler 2 [created]:", e.Payload)
	})
	bus.Subscribe(EventUpdated, func(e Event) {
		fmt.Println("Handler [updated]:", e.Payload)
	})

	bus.Publish(Event{Type: EventCreated, Payload: "user:Alice"})
	bus.Publish(Event{Type: EventUpdated, Payload: "user:Alice→Bob"})

	time.Sleep(10 * time.Millisecond) // wait for async handlers

	// -------------------------------------------------------------------------
	// SECTION 4: Middleware chain
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Middleware Chain ---")

	baseHandler := func(req string) string {
		return "processed: " + req
	}

	// Compose middleware — innermost runs first
	handler := Uppercase(Timing(Logging("myhandler", baseHandler)))
	result2 := handler("hello world")
	fmt.Println("Final:", result2)

	// -------------------------------------------------------------------------
	// SECTION 5: Functional options
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Functional Options ---")

	client := NewHTTPClient(
		WithBaseURL("https://api.example.com"),
		WithTimeout2(10*time.Second),
		WithMaxRetries(5),
		WithHeader("Authorization", "Bearer token123"),
		WithHeader("Accept", "application/json"),
	)
	fmt.Println(client)

	// -------------------------------------------------------------------------
	// SECTION 6: Circuit Breaker
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Circuit Breaker ---")

	cb := NewCircuitBreaker(3, 1*time.Second)

	failCount := 0
	operation := func() error {
		failCount++
		if failCount <= 5 {
			return fmt.Errorf("service unavailable")
		}
		return nil
	}

	for i := 0; i < 8; i++ {
		err3 := cb.Call(operation)
		if err3 != nil {
			fmt.Printf("call %d failed: %v\n", i+1, err3)
		} else {
			fmt.Printf("call %d succeeded\n", i+1)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// -------------------------------------------------------------------------
	// SECTION 7: Reflection
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Reflection ---")

	type Config struct {
		Host    string `yaml:"host" env:"APP_HOST"`
		Port    int    `yaml:"port" env:"APP_PORT"`
		Debug   bool   `yaml:"debug" env:"APP_DEBUG"`
	}

	cfg := Config{Host: "localhost", Port: 8080, Debug: true}
	inspectValue(cfg)

	fmt.Println("\nInspect slice:")
	inspectValue([]int{1, 2, 3, 4, 5})

	fmt.Println("\nInspect map:")
	inspectValue(map[string]int{"a": 1, "b": 2})

	fmt.Println("\nInspect function:")
	inspectFunc(func(a int, b string) (bool, error) { return true, nil })

	// DeepEqual
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}
	fmt.Println("\nDeepEqual a,b:", deepEqual(a, b)) // true
	fmt.Println("DeepEqual a,c:", deepEqual(a, c))   // false

	// Reflect TypeOf and ValueOf
	x := 42
	fmt.Printf("\nreflect.TypeOf(x): %v\n", reflect.TypeOf(x))
	fmt.Printf("reflect.ValueOf(x): %v\n", reflect.ValueOf(x))
	fmt.Printf("reflect.ValueOf(x).Kind(): %v\n", reflect.ValueOf(x).Kind())

	// -------------------------------------------------------------------------
	// SECTION 8: Runtime info
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Runtime Info ---")
	runtimeInfo()

	// -------------------------------------------------------------------------
	// SECTION 9: Common Go idioms
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Go Idioms Summary ---")
	fmt.Println(`
1.  Error as value: always return error as last value
2.  Accept interfaces, return concrete types
3.  Make zero values useful (var mu sync.Mutex is ready to use)
4.  Context first: func Do(ctx context.Context, ...) error
5.  Small interfaces: prefer 1-2 method interfaces
6.  Embedding for composition, not inheritance
7.  defer for cleanup (always right after acquiring resource)
8.  Table-driven tests for all functions
9.  Named return values only for documentation, not logic
10. Keep goroutines clean: always cancel context, close channels
11. Don't communicate by sharing memory — share memory by communicating
12. Channels for ownership transfer, sync.Mutex for state protection
13. sync.Once for initialization, sync.Pool for expensive objects
14. Build package APIs with exported types + unexported implementation
15. Use go vet and staticcheck in CI/CD pipeline
`)

	fmt.Println("=== MODULE 10 COMPLETE ===")
}
