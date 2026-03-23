// 01_design_patterns_creational.go
//
// Creational Design Patterns in Go
// =================================
// Creational patterns deal with object creation mechanisms.
// In Go, these look different from textbook OOP because:
//   - Go has no classes or constructors (only structs + functions)
//   - Go has no inheritance (only embedding + interfaces)
//   - Go favors composition over inheritance
//   - The language itself guides you toward certain patterns
//
// Patterns covered:
//   1. Singleton   — sync.Once for thread-safe lazy initialization
//   2. Builder     — fluent builder AND functional options (the Go-idiomatic way)
//   3. Factory     — constructor functions, the New() convention
//   4. Object Pool — sync.Pool for expensive object reuse
//   5. Prototype   — Clone() methods, deep-copy semantics

package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// =============================================================================
// PATTERN 1: SINGLETON
// =============================================================================
//
// Intent: Ensure exactly one instance of a type exists in the process.
//
// When to use:
//   - Database connection pools (one pool, not per-request)
//   - Configuration objects (read once at startup)
//   - Logger instances
//   - Metrics registries
//
// Go-specific challenge: goroutine safety.
// Naive check-then-set is a data race. sync.Once solves it correctly.
//
// Common mistake: using init() — it runs even if the singleton is never needed.
// sync.Once gives you lazy initialization: only initialize when first accessed.

// DatabasePool represents an expensive resource we want only one of.
type DatabasePool struct {
	maxConnections int
	dsn            string
	connections    []*fakeConn
	mu             sync.Mutex
}

type fakeConn struct{ id int }

// package-level variable: the singleton instance.
// Unexported so external packages cannot replace it.
var (
	dbPoolInstance *DatabasePool
	dbPoolOnce    sync.Once // zero value is ready to use — no Init() needed
)

// GetDatabasePool returns the singleton pool.
// First call initializes it; all subsequent calls return the same pointer.
//
// sync.Once guarantees:
//  1. The function runs exactly once, even with thousands of concurrent callers.
//  2. All callers BLOCK until initialization completes (not just the first).
//  3. No double-check locking needed — Once handles the memory model correctly.
func GetDatabasePool() *DatabasePool {
	dbPoolOnce.Do(func() {
		// This closure runs exactly once, ever.
		fmt.Println("  [singleton] initializing database pool (expensive operation)...")
		time.Sleep(10 * time.Millisecond) // simulate slow startup

		dbPoolInstance = &DatabasePool{
			maxConnections: 10,
			dsn:            "postgres://localhost:5432/mydb",
		}
		for i := 0; i < 3; i++ {
			dbPoolInstance.connections = append(dbPoolInstance.connections, &fakeConn{id: i})
		}
	})
	return dbPoolInstance
}

// Query demonstrates using the singleton.
func (p *DatabasePool) Query(sql string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fmt.Sprintf("result of '%s' from pool(dsn=%s)", sql, p.dsn)
}

// Why NOT use init()?
//   init() runs at program startup even if you never call GetDatabasePool().
//   With sync.Once, no connection is made until actually needed (lazy).
//   Also: init() cannot return errors; sync.Once function can set an error variable.

// =============================================================================
// PATTERN 2A: BUILDER (Fluent / Method Chaining Style)
// =============================================================================
//
// Intent: Construct complex objects step-by-step, separating construction
//         from representation. Useful when an object has many optional fields
//         and the constructor would have an unreadable parameter list.
//
// Fluent style uses method chaining: builder.SetX(x).SetY(y).Build()
// Each setter returns *Builder, allowing chaining.
//
// Downside of fluent builder in Go: you can't easily check for errors
// mid-chain. One solution: store the error in the builder itself.

// ServerConfig is the complex object we want to build.
type ServerConfig struct {
	host            string
	port            int
	readTimeout     time.Duration
	writeTimeout    time.Duration
	maxHeaderBytes  int
	tlsEnabled      bool
	compressionLevel int
}

// ServerConfigBuilder holds state while we construct a ServerConfig.
type ServerConfigBuilder struct {
	config ServerConfig
	err    error // accumulated error — set by any step that fails
}

// NewServerConfigBuilder creates a builder with sensible defaults.
// This is the "default configuration" pattern — always provide defaults.
func NewServerConfigBuilder() *ServerConfigBuilder {
	return &ServerConfigBuilder{
		config: ServerConfig{
			host:            "localhost",
			port:            8080,
			readTimeout:     30 * time.Second,
			writeTimeout:    30 * time.Second,
			maxHeaderBytes:  1 << 20, // 1 MiB
			compressionLevel: 0,
		},
	}
}

// Each setter returns *ServerConfigBuilder for chaining.
// Validation errors are stored and surfaced at Build() time.

func (b *ServerConfigBuilder) WithHost(host string) *ServerConfigBuilder {
	if host == "" {
		b.err = fmt.Errorf("host cannot be empty")
		return b
	}
	b.config.host = host
	return b
}

func (b *ServerConfigBuilder) WithPort(port int) *ServerConfigBuilder {
	if port < 1 || port > 65535 {
		b.err = fmt.Errorf("port %d out of range [1, 65535]", port)
		return b
	}
	b.config.port = port
	return b
}

func (b *ServerConfigBuilder) WithTimeouts(read, write time.Duration) *ServerConfigBuilder {
	b.config.readTimeout = read
	b.config.writeTimeout = write
	return b
}

func (b *ServerConfigBuilder) WithTLS() *ServerConfigBuilder {
	b.config.tlsEnabled = true
	return b
}

func (b *ServerConfigBuilder) WithCompression(level int) *ServerConfigBuilder {
	if level < 0 || level > 9 {
		b.err = fmt.Errorf("compression level must be 0-9, got %d", level)
		return b
	}
	b.config.compressionLevel = level
	return b
}

// Build finalizes construction and returns the product.
// This is the only place where the accumulated error is surfaced.
func (b *ServerConfigBuilder) Build() (ServerConfig, error) {
	if b.err != nil {
		return ServerConfig{}, b.err
	}
	return b.config, nil
}

// =============================================================================
// PATTERN 2B: FUNCTIONAL OPTIONS (The Idiomatic Go Way)
// =============================================================================
//
// Invented/popularized by Rob Pike and Dave Cheney.
// Instead of a builder struct, options are functions that modify a config.
//
// Why it's preferred in Go:
//   - No builder struct needed; less boilerplate
//   - New options are added without changing the constructor signature
//   - Options are composable and testable in isolation
//   - The API is self-documenting: WithTLS(), WithTimeout(5s)
//   - Works beautifully with variadic arguments
//
// The pattern:
//   type Option func(*T)            // Option is a function that mutates T
//   func New(opts ...Option) *T {   // constructor accepts variadic options
//       t := &T{defaults}
//       for _, opt := range opts { opt(t) }
//       return t
//   }

// HTTPClient is the object we're configuring with functional options.
type HTTPClient struct {
	baseURL    string
	timeout    time.Duration
	retries    int
	userAgent  string
	debug      bool
}

// Option is the functional option type.
// It's a function that takes a pointer and modifies it in place.
type Option func(*HTTPClient)

// Each With* function is a constructor for an Option.
// They are called "option constructors" or just "options".

func WithBaseURL(url string) Option {
	return func(c *HTTPClient) {
		c.baseURL = url
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *HTTPClient) {
		c.timeout = d
	}
}

func WithRetries(n int) Option {
	return func(c *HTTPClient) {
		c.retries = n
	}
}

func WithUserAgent(ua string) Option {
	return func(c *HTTPClient) {
		c.userAgent = ua
	}
}

func WithDebug() Option {
	return func(c *HTTPClient) {
		c.debug = true
	}
}

// NewHTTPClient creates a client with defaults, then applies all options.
// The caller can pass zero, some, or all options.
func NewHTTPClient(opts ...Option) *HTTPClient {
	// Start with sane defaults.
	client := &HTTPClient{
		timeout:   30 * time.Second,
		retries:   3,
		userAgent: "MyApp/1.0",
	}
	// Apply each option in order. Later options override earlier ones.
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *HTTPClient) Get(path string) string {
	return fmt.Sprintf("GET %s%s (timeout=%v, retries=%d, debug=%v)",
		c.baseURL, path, c.timeout, c.retries, c.debug)
}

// =============================================================================
// PATTERN 3: FACTORY FUNCTION (Constructor Functions)
// =============================================================================
//
// Go has no constructors in the OOP sense.
// Convention: functions named New<Type> or just New return an initialized value.
//
// Why factory functions matter:
//   - Enforce invariants: you can validate and return an error
//   - Hide implementation: return interface, not concrete struct
//   - Control memory: decide pointer vs value
//   - Set up internal state that zero value doesn't provide
//
// The New() convention:
//   - Single type in package: func New(...) *T
//   - Multiple types: func NewFoo(...) *Foo, func NewBar(...) *Bar
//   - Return interface when you want to hide implementation details
//
// Common mistake: returning a value type when you need shared state
//   var m MyMap   // MyMap has unexported sync.Mutex — BROKEN
//   m = MyMap{}   // zero mutex is fine, but map is nil → panic
//   Use New() to ensure the map is initialized.

// Shape is an interface (factory returns this, hiding concrete type).
type Shape interface {
	Area() float64
	Perimeter() float64
	String() string
}

type circle struct {
	radius float64
}

type rectangle struct {
	width, height float64
}

// NewCircle is a factory function.
// Returns the Shape interface — callers don't need to know it's a *circle.
func NewCircle(radius float64) (Shape, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("circle: radius must be positive, got %f", radius)
	}
	return &circle{radius: radius}, nil
}

func NewRectangle(width, height float64) (Shape, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("rectangle: dimensions must be positive")
	}
	return &rectangle{width: width, height: height}, nil
}

func (c *circle) Area() float64      { return 3.14159 * c.radius * c.radius }
func (c *circle) Perimeter() float64 { return 2 * 3.14159 * c.radius }
func (c *circle) String() string     { return fmt.Sprintf("Circle(r=%.2f)", c.radius) }

func (r *rectangle) Area() float64      { return r.width * r.height }
func (r *rectangle) Perimeter() float64 { return 2 * (r.width + r.height) }
func (r *rectangle) String() string {
	return fmt.Sprintf("Rectangle(%.2fx%.2f)", r.width, r.height)
}

// ShapeFactory demonstrates the Abstract Factory variant:
// a factory that creates families of related objects.
type ShapeFactory struct {
	defaultColor string
}

func NewShapeFactory(color string) *ShapeFactory {
	return &ShapeFactory{defaultColor: color}
}

func (f *ShapeFactory) CreateCircle(radius float64) (Shape, error) {
	fmt.Printf("  [factory] creating %s circle\n", f.defaultColor)
	return NewCircle(radius)
}

// =============================================================================
// PATTERN 4: OBJECT POOL (sync.Pool)
// =============================================================================
//
// Intent: Reuse expensive-to-allocate objects instead of creating new ones
//         every time. Reduces GC pressure in high-throughput code.
//
// sync.Pool is Go's built-in pool.
// Key behaviors:
//   - Get() returns a pooled object or calls New() if pool is empty
//   - Put() returns an object to the pool for future reuse
//   - Objects MAY be evicted by the GC at any time (between GC cycles)
//   - NOT suitable for connections or objects with close semantics
//   - IS suitable for: buffers, encoders/decoders, scratch space
//
// Classic real-world use: bytes.Buffer pool in net/http, encoding/json, etc.
//
// Common mistakes:
//   1. Forgetting to Reset() before Put() — stale data leaks between uses
//   2. Using Pool for objects that must be explicitly closed (use sync.Pool
//      for buffers, use a channel-based pool for DB connections)
//   3. Holding a reference to a pooled object after Put() (use-after-free)

// ExpensiveObject simulates something costly to allocate.
type ExpensiveObject struct {
	id     int
	buffer []byte // pre-allocated buffer
	data   map[string]int
}

func newExpensiveObject() *ExpensiveObject {
	return &ExpensiveObject{
		id:     rand.Intn(10000),
		buffer: make([]byte, 0, 4096), // 4 KiB pre-allocated
		data:   make(map[string]int, 64),
	}
}

// Reset clears the object so it's safe to reuse.
// Always call Reset() before putting back into pool.
func (e *ExpensiveObject) Reset() {
	e.buffer = e.buffer[:0]  // reset length, keep capacity
	for k := range e.data {  // clear map (Go 1.21: use clear(e.data))
		delete(e.data, k)
	}
}

// objectPool is the pool. New is called when Get() finds no available object.
var objectPool = &sync.Pool{
	New: func() interface{} {
		fmt.Println("  [pool] allocating new ExpensiveObject")
		return newExpensiveObject()
	},
}

// UsePooledObject demonstrates the acquire → use → release lifecycle.
func UsePooledObject(key string, value int) {
	// Acquire from pool (or allocate via New if pool is empty).
	obj := objectPool.Get().(*ExpensiveObject)

	// CRITICAL: always defer Put() so we return even on panic.
	defer func() {
		obj.Reset()         // clean before returning
		objectPool.Put(obj) // return to pool
	}()

	// Use the object.
	obj.data[key] = value
	msg := fmt.Sprintf("key=%s value=%d", key, value)
	obj.buffer = append(obj.buffer, msg...)
}

// BufferPool is a more focused pool for bytes.Buffer.
// This pattern is used extensively in the standard library.
var bufferPool = &sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func buildString(parts ...string) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset() // important: reset before use
	defer bufferPool.Put(buf)

	for i, p := range parts {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(p)
	}
	return buf.String()
}

// =============================================================================
// PATTERN 5: PROTOTYPE (Clone Methods)
// =============================================================================
//
// Intent: Create new objects by copying (cloning) an existing object.
//         Useful when initialization is expensive or configuration is complex.
//
// In Go: there's no built-in clone mechanism.
// Every struct assignment is a SHALLOW copy by default.
// For deep copies, you must implement Clone() yourself.
//
// Shallow vs Deep copy:
//   - Shallow: copy the struct, shared pointers still point to same data
//   - Deep: recursively copy all referenced data (slices, maps, pointers)
//
// When prototype matters:
//   - Template objects with default configuration
//   - Undo/redo systems (store state snapshots)
//   - Test fixtures (clone a base object, tweak one field per test)

// GameCharacter demonstrates prototype with deep copy semantics.
type GameCharacter struct {
	Name        string
	Level       int
	Stats       map[string]int // must be deep-copied
	Inventory   []string       // must be deep-copied
	Position    *Point         // must be deep-copied (pointer)
}

type Point struct{ X, Y float64 }

// Clone creates a fully independent deep copy of GameCharacter.
// Modifying the clone does NOT affect the original.
func (g *GameCharacter) Clone() *GameCharacter {
	if g == nil {
		return nil
	}

	// Copy primitive fields directly (Name, Level are value types).
	clone := &GameCharacter{
		Name:  g.Name,
		Level: g.Level,
	}

	// Deep copy the map.
	if g.Stats != nil {
		clone.Stats = make(map[string]int, len(g.Stats))
		for k, v := range g.Stats {
			clone.Stats[k] = v
		}
	}

	// Deep copy the slice.
	if g.Inventory != nil {
		clone.Inventory = make([]string, len(g.Inventory))
		copy(clone.Inventory, g.Inventory)
	}

	// Deep copy the pointer (allocate new Point, copy value).
	if g.Position != nil {
		pos := *g.Position // dereference to copy the value
		clone.Position = &pos
	}

	return clone
}

func (g *GameCharacter) String() string {
	return fmt.Sprintf("%s(lvl=%d, pos=%v, stats=%v, inv=%v)",
		g.Name, g.Level, g.Position, g.Stats, g.Inventory)
}

// =============================================================================
// MAIN: Demonstrate all patterns
// =============================================================================

func main() {
	fmt.Println("=== CREATIONAL DESIGN PATTERNS IN GO ===")
	fmt.Println()

	// ------------------------------------------------------------------
	// 1. SINGLETON
	// ------------------------------------------------------------------
	fmt.Println("--- 1. SINGLETON ---")

	// Simulate 5 goroutines all trying to get the pool simultaneously.
	var wg sync.WaitGroup
	results := make([]*DatabasePool, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetDatabasePool()
		}(i)
	}
	wg.Wait()

	// All goroutines got the same pointer.
	fmt.Printf("  Pool pointer from goroutine 0: %p\n", results[0])
	fmt.Printf("  Pool pointer from goroutine 4: %p\n", results[4])
	fmt.Printf("  Same instance? %v\n", results[0] == results[4])
	fmt.Println("  Query result:", results[0].Query("SELECT 1"))
	fmt.Println()

	// ------------------------------------------------------------------
	// 2A. BUILDER (Fluent)
	// ------------------------------------------------------------------
	fmt.Println("--- 2A. BUILDER (Fluent / Method Chaining) ---")

	cfg, err := NewServerConfigBuilder().
		WithHost("api.example.com").
		WithPort(443).
		WithTimeouts(60*time.Second, 60*time.Second).
		WithTLS().
		WithCompression(6).
		Build()
	if err != nil {
		fmt.Println("  Error:", err)
	} else {
		fmt.Printf("  Config: host=%s port=%d tls=%v compression=%d timeouts=%v/%v\n",
			cfg.host, cfg.port, cfg.tlsEnabled, cfg.compressionLevel,
			cfg.readTimeout, cfg.writeTimeout)
	}

	// Demonstrate error accumulation.
	_, err = NewServerConfigBuilder().
		WithPort(99999). // invalid
		WithHost("").    // also invalid
		Build()
	fmt.Println("  Validation error (expected):", err)
	fmt.Println()

	// ------------------------------------------------------------------
	// 2B. FUNCTIONAL OPTIONS
	// ------------------------------------------------------------------
	fmt.Println("--- 2B. FUNCTIONAL OPTIONS (Idiomatic Go) ---")

	// Default client — zero options.
	defaultClient := NewHTTPClient()
	fmt.Println("  Default client:", defaultClient.Get("/ping"))

	// Customized client — only the options you care about.
	apiClient := NewHTTPClient(
		WithBaseURL("https://api.example.com"),
		WithTimeout(5*time.Second),
		WithRetries(1),
		WithUserAgent("bot/2.0"),
		WithDebug(),
	)
	fmt.Println("  API client:", apiClient.Get("/users"))

	// You can compose options.
	productionOpts := []Option{
		WithTimeout(10 * time.Second),
		WithRetries(3),
	}
	prodClient := NewHTTPClient(append(productionOpts, WithBaseURL("https://prod.api.com"))...)
	fmt.Println("  Prod client:", prodClient.Get("/health"))
	fmt.Println()

	// ------------------------------------------------------------------
	// 3. FACTORY FUNCTION
	// ------------------------------------------------------------------
	fmt.Println("--- 3. FACTORY FUNCTION ---")

	shapes := []struct {
		name    string
		factory func() (Shape, error)
	}{
		{"circle r=5", func() (Shape, error) { return NewCircle(5) }},
		{"rect 4x6", func() (Shape, error) { return NewRectangle(4, 6) }},
		{"invalid circle", func() (Shape, error) { return NewCircle(-1) }},
	}

	for _, tc := range shapes {
		s, err := tc.factory()
		if err != nil {
			fmt.Printf("  %-18s → error: %v\n", tc.name, err)
		} else {
			fmt.Printf("  %-18s → area=%.2f perimeter=%.2f\n",
				s.String(), s.Area(), s.Perimeter())
		}
	}

	factory := NewShapeFactory("blue")
	s, _ := factory.CreateCircle(3.0)
	fmt.Println("  Factory created:", s)
	fmt.Println()

	// ------------------------------------------------------------------
	// 4. OBJECT POOL
	// ------------------------------------------------------------------
	fmt.Println("--- 4. OBJECT POOL (sync.Pool) ---")

	// First few calls allocate via New.
	for i := 0; i < 3; i++ {
		UsePooledObject(fmt.Sprintf("key%d", i), i*10)
	}

	// After the first objects are returned to pool, subsequent calls
	// should reuse them (you may see "allocating new" only on first call
	// or after GC; sync.Pool is not a guaranteed cache).
	fmt.Println("  Reusing pooled objects:")
	for i := 3; i < 6; i++ {
		UsePooledObject(fmt.Sprintf("key%d", i), i*10)
	}

	// Buffer pool demo.
	result := buildString("alpha", "beta", "gamma", "delta")
	fmt.Println("  BufferPool result:", result)
	fmt.Println()

	// ------------------------------------------------------------------
	// 5. PROTOTYPE (Clone)
	// ------------------------------------------------------------------
	fmt.Println("--- 5. PROTOTYPE (Clone / Deep Copy) ---")

	// Create the "template" character.
	template := &GameCharacter{
		Name:  "BaseWarrior",
		Level: 1,
		Stats: map[string]int{
			"strength": 10,
			"defense":  8,
			"speed":    6,
		},
		Inventory: []string{"sword", "shield"},
		Position:  &Point{X: 0, Y: 0},
	}

	// Clone it to create two independent characters.
	warrior1 := template.Clone()
	warrior1.Name = "Thor"
	warrior1.Level = 15
	warrior1.Stats["strength"] = 25
	warrior1.Inventory = append(warrior1.Inventory, "hammer")
	warrior1.Position.X = 100

	warrior2 := template.Clone()
	warrior2.Name = "Loki"
	warrior2.Level = 12
	warrior2.Stats["speed"] = 20
	warrior2.Inventory = append(warrior2.Inventory, "staff")
	warrior2.Position.Y = 200

	fmt.Println("  Template :", template)
	fmt.Println("  Warrior 1:", warrior1)
	fmt.Println("  Warrior 2:", warrior2)

	// Verify deep copy: template is unchanged.
	fmt.Printf("  Template strength still 10? %v (was not polluted by clones)\n",
		template.Stats["strength"] == 10)
	fmt.Printf("  Template position still (0,0)? %v\n",
		template.Position.X == 0 && template.Position.Y == 0)

	fmt.Println()
	fmt.Println("=== END CREATIONAL PATTERNS ===")
}
