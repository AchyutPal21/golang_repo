// 07_functional_options.go
//
// THE FUNCTIONAL OPTIONS PATTERN
//
// Problem: How do you create a struct with many optional configuration fields
// without forcing callers to provide values for everything?
//
// Bad solutions:
//   1. Telescoping constructors: NewServer(), NewServerWithTimeout(),
//      NewServerWithTimeoutAndMaxRetries(), ... → combinatorial explosion
//   2. Config struct passed to constructor: NewServer(ServerConfig{...})
//      → zero values are ambiguous (did you mean 0 retries, or "use default"?)
//   3. Exported fields on the struct: server.Timeout = 30
//      → not atomic, not validated, allows partial construction
//
// The FUNCTIONAL OPTIONS pattern (coined by Rob Pike, popularized by Dave Cheney):
//   - Define an Option type: type Option func(*Server)
//   - Define constructor functions that RETURN Options: WithTimeout(d) Option
//   - Constructor takes variadic ...Option and applies each in order
//   - Unoprovided options keep their default values (set before applying options)
//
// ADVANTAGES:
//   + Self-documenting: WithTimeout(5*time.Second) is clear
//   + Extensible: add new options without breaking existing callers
//   + Validated: each WithXxx function can validate its argument
//   + Default values are obvious and centralized
//   + Thread safety can be baked into individual options
//   + The option functions themselves are composable
//
// Used by many major Go libraries: grpc-go, zap, cobra, etc.

package main

import (
	"fmt"
	"log"
	"time"
)

// ─── 1. The Server Type We Want to Configure ──────────────────────────────────

// Server represents an HTTP server with many optional configuration fields.
// Most fields have sensible defaults that most callers should not need to change.
type Server struct {
	host            string
	port            int
	timeout         time.Duration
	maxConnections  int
	maxRetries      int
	retryDelay      time.Duration
	enableTLS       bool
	tlsCertFile     string
	tlsKeyFile      string
	logger          Logger
	rateLimitRPS    int   // requests per second, 0 = disabled
	readBufferSize  int
	writeBufferSize int
}

// Logger is a small interface for pluggable logging (enables testing with mocks).
type Logger interface {
	Printf(format string, args ...any)
}

// stdLogger is a Logger backed by the standard log package.
type stdLogger struct{}

func (s stdLogger) Printf(format string, args ...any) {
	log.Printf(format, args...)
}

// noopLogger discards all log messages (useful in tests).
type noopLogger struct{}

func (n noopLogger) Printf(format string, args ...any) {}

// ─── 2. The Option Type ───────────────────────────────────────────────────────
//
// An Option is a function that modifies a *Server in place.
// Each WithXxx function constructs and returns one of these.

type Option func(*Server)

// ─── 3. Option Constructor Functions ─────────────────────────────────────────
//
// Convention: WithXxx returns an Option.
// Each validates its argument and panics or returns a no-op for invalid values.
// Prefer returning errors for production APIs; panic is sometimes used for
// programmer errors (invalid option values detected at startup).

// WithHost sets the server's bind host.
func WithHost(host string) Option {
	return func(s *Server) {
		if host == "" {
			panic("functional_options: WithHost: host cannot be empty")
		}
		s.host = host
	}
}

// WithPort sets the server's port.
func WithPort(port int) Option {
	return func(s *Server) {
		if port < 1 || port > 65535 {
			panic(fmt.Sprintf("functional_options: WithPort: invalid port %d", port))
		}
		s.port = port
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(s *Server) {
		if d <= 0 {
			panic("functional_options: WithTimeout: duration must be positive")
		}
		s.timeout = d
	}
}

// WithMaxConnections sets the maximum number of concurrent connections.
func WithMaxConnections(max int) Option {
	return func(s *Server) {
		if max <= 0 {
			panic("functional_options: WithMaxConnections: must be positive")
		}
		s.maxConnections = max
	}
}

// WithMaxRetries sets how many times a failed operation is retried.
func WithMaxRetries(n int) Option {
	return func(s *Server) {
		if n < 0 {
			panic("functional_options: WithMaxRetries: cannot be negative")
		}
		s.maxRetries = n
	}
}

// WithRetryDelay sets the delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(s *Server) {
		if d < 0 {
			panic("functional_options: WithRetryDelay: cannot be negative")
		}
		s.retryDelay = d
	}
}

// WithTLS enables TLS with the given certificate and key files.
// This is a "compound" option that sets multiple fields atomically.
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) {
		if certFile == "" || keyFile == "" {
			panic("functional_options: WithTLS: cert and key file paths required")
		}
		s.enableTLS = true
		s.tlsCertFile = certFile
		s.tlsKeyFile = keyFile
	}
}

// WithLogger replaces the default logger with a custom implementation.
// This enables injecting test loggers or structured loggers (zap, slog).
func WithLogger(l Logger) Option {
	return func(s *Server) {
		if l == nil {
			panic("functional_options: WithLogger: logger cannot be nil")
		}
		s.logger = l
	}
}

// WithRateLimit sets the requests-per-second rate limit (0 = disabled).
func WithRateLimit(rps int) Option {
	return func(s *Server) {
		if rps < 0 {
			panic("functional_options: WithRateLimit: cannot be negative")
		}
		s.rateLimitRPS = rps
	}
}

// WithBufferSizes sets the read and write buffer sizes.
func WithBufferSizes(read, write int) Option {
	return func(s *Server) {
		if read <= 0 || write <= 0 {
			panic("functional_options: WithBufferSizes: both sizes must be positive")
		}
		s.readBufferSize = read
		s.writeBufferSize = write
	}
}

// ─── 4. The Constructor ───────────────────────────────────────────────────────
//
// NewServer sets sensible defaults, then applies each Option in order.
// Callers only provide the options they want to change.

func NewServer(opts ...Option) *Server {
	// 1. Set defaults — any unspecified option keeps these values.
	//    All defaults are in one place → easy to audit and change.
	s := &Server{
		host:            "0.0.0.0",
		port:            8080,
		timeout:         30 * time.Second,
		maxConnections:  1000,
		maxRetries:      3,
		retryDelay:      500 * time.Millisecond,
		enableTLS:       false,
		logger:          stdLogger{},
		rateLimitRPS:    0, // disabled by default
		readBufferSize:  4096,
		writeBufferSize: 4096,
	}

	// 2. Apply each option function in order.
	//    Options are applied left-to-right, so later options override earlier ones.
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// ─── 5. Methods on Server ─────────────────────────────────────────────────────

func (s *Server) Address() string {
	scheme := "http"
	if s.enableTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, s.host, s.port)
}

func (s *Server) String() string {
	tls := "no"
	if s.enableTLS {
		tls = fmt.Sprintf("yes (cert=%s)", s.tlsCertFile)
	}
	rateLimit := "disabled"
	if s.rateLimitRPS > 0 {
		rateLimit = fmt.Sprintf("%d rps", s.rateLimitRPS)
	}
	return fmt.Sprintf(
		"Server{addr=%s, timeout=%v, maxConn=%d, retries=%d, tls=%s, rateLimit=%s, buf=%d/%d}",
		s.Address(),
		s.timeout,
		s.maxConnections,
		s.maxRetries,
		tls,
		rateLimit,
		s.readBufferSize,
		s.writeBufferSize,
	)
}

// ─── 6. Option Composition ────────────────────────────────────────────────────
//
// Because options are just functions, you can compose them into "preset" options
// that apply a group of settings at once.

// ProductionDefaults returns an Option that applies production-grade settings.
// This is an "option bundle" — a single option that expands into multiple.
func ProductionDefaults() Option {
	return func(s *Server) {
		// Apply each sub-option directly to s
		WithTimeout(10 * time.Second)(s)
		WithMaxConnections(10000)(s)
		WithMaxRetries(5)(s)
		WithRetryDelay(200 * time.Millisecond)(s)
		WithRateLimit(1000)(s)
		WithBufferSizes(8192, 8192)(s)
	}
}

// DevelopmentDefaults returns an Option suitable for local development.
func DevelopmentDefaults() Option {
	return func(s *Server) {
		WithTimeout(60 * time.Second)(s)
		WithMaxConnections(50)(s)
		WithMaxRetries(1)(s)
		WithLogger(noopLogger{})(s) // silence logs in dev
	}
}

// ─── 7. Variant: Options with Error Return ────────────────────────────────────
//
// Some libraries use func(*Config) error for options so that invalid inputs
// return errors rather than panicking. This is safer for library code
// where panics are undesirable. Shown here as a pattern.

type DatabaseOption func(*DatabaseConfig) error

type DatabaseConfig struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
	MaxLifetime time.Duration
}

func WithDSN(dsn string) DatabaseOption {
	return func(cfg *DatabaseConfig) error {
		if dsn == "" {
			return fmt.Errorf("WithDSN: DSN cannot be empty")
		}
		cfg.DSN = dsn
		return nil
	}
}

func WithMaxOpen(n int) DatabaseOption {
	return func(cfg *DatabaseConfig) error {
		if n <= 0 {
			return fmt.Errorf("WithMaxOpen: must be positive, got %d", n)
		}
		cfg.MaxOpenConn = n
		return nil
	}
}

func NewDatabaseConfig(opts ...DatabaseOption) (*DatabaseConfig, error) {
	cfg := &DatabaseConfig{
		MaxOpenConn: 25,
		MaxIdleConn: 5,
		MaxLifetime: 5 * time.Minute,
	}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("NewDatabaseConfig: %w", err)
		}
	}
	return cfg, nil
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Functional Options Pattern")
	fmt.Println("========================================")

	// ── Default Server ───────────────────────────────────────────────────────
	fmt.Println("\n── Default Server (no options) ──────────────────────")
	s1 := NewServer()
	fmt.Println(s1)

	// ── Custom Server ────────────────────────────────────────────────────────
	fmt.Println("\n── Custom Server (selective options) ────────────────")
	s2 := NewServer(
		WithHost("192.168.1.1"),
		WithPort(9090),
		WithTimeout(15*time.Second),
		WithMaxRetries(5),
	)
	fmt.Println(s2)

	// ── TLS Server ───────────────────────────────────────────────────────────
	fmt.Println("\n── TLS-Enabled Server ───────────────────────────────")
	s3 := NewServer(
		WithPort(443),
		WithTLS("/etc/ssl/cert.pem", "/etc/ssl/key.pem"),
		WithTimeout(20*time.Second),
		WithRateLimit(500),
	)
	fmt.Println(s3)
	fmt.Printf("Address: %s\n", s3.Address())

	// ── Production Server with Preset ────────────────────────────────────────
	fmt.Println("\n── Production Server (preset option) ────────────────")
	s4 := NewServer(
		ProductionDefaults(),         // apply bundle first
		WithHost("10.0.0.1"),         // then override specific values
		WithTLS("/ssl/cert", "/ssl/key"),
	)
	fmt.Println(s4)

	// ── Development Server ───────────────────────────────────────────────────
	fmt.Println("\n── Development Server (preset option) ───────────────")
	s5 := NewServer(
		DevelopmentDefaults(),
		WithPort(3000),
	)
	fmt.Println(s5)

	// ── Demonstrating Option Order (later overrides earlier) ─────────────────
	fmt.Println("\n── Option Order: Later Overrides Earlier ─────────────")
	s6 := NewServer(
		WithTimeout(10*time.Second), // set to 10s
		WithTimeout(60*time.Second), // override to 60s — this wins
	)
	fmt.Printf("Timeout: %v (expected 60s)\n", s6.timeout)

	// ── Options with Error Return ─────────────────────────────────────────────
	fmt.Println("\n── Variant: Options Returning Error ─────────────────")

	dbCfg, err := NewDatabaseConfig(
		WithDSN("postgres://user:pass@host/db"),
		WithMaxOpen(50),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("DB Config: DSN=%s, MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v\n",
			dbCfg.DSN, dbCfg.MaxOpenConn, dbCfg.MaxIdleConn, dbCfg.MaxLifetime)
	}

	// Error case
	_, err = NewDatabaseConfig(
		WithDSN(""),     // invalid — empty DSN
		WithMaxOpen(10),
	)
	fmt.Printf("Expected error: %v\n", err)

	// ── Summary ──────────────────────────────────────────────────────────────
	fmt.Println("\n── Pattern Summary ──────────────────────────────────")
	fmt.Println(`
  type Option func(*Server)

  func WithTimeout(d time.Duration) Option {
      return func(s *Server) { s.timeout = d }
  }

  func NewServer(opts ...Option) *Server {
      s := &Server{ /* defaults */ }
      for _, opt := range opts {
          opt(s)
      }
      return s
  }

  Benefits:
    + Self-documenting call sites: NewServer(WithTimeout(5s), WithTLS(...))
    + Backward compatible: new options never break existing callers
    + Defaults are centralized and auditable
    + Options are composable (bundle them into presets)
    + Each option can validate its input
    + Easy to test: pass a different option to change behavior

  Used by: grpc-go, uber-go/zap, spf13/cobra, google/go-cloud, ...
  `)
}
