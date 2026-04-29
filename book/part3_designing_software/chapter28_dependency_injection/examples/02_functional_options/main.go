// FILE: book/part3_designing_software/chapter28_dependency_injection/examples/02_functional_options/main.go
// CHAPTER: 28 — Dependency Injection
// TOPIC: Functional options pattern for flexible constructors.
//        Options as first-class values, option validation, default values.
//
// Run (from the chapter folder):
//   go run ./examples/02_functional_options

package main

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Problem: a struct with many optional fields.
// Option A: long constructor signature — breaks on every addition.
// Option B: config struct — caller must know all field names.
// Option C: functional options — extensible, self-documenting, testable.
// ─────────────────────────────────────────────────────────────────────────────

// ─── Server with functional options ──────────────────────────────────────────

type Logger interface {
	Log(msg string)
}

type Server struct {
	host        string
	port        int
	timeout     time.Duration
	maxConns    int
	logger      Logger
	tlsEnabled  bool
	retries     int
}

// Option is a function that configures a Server.
type Option func(*Server) error

// WithHost sets the server host.
func WithHost(host string) Option {
	return func(s *Server) error {
		if host == "" {
			return fmt.Errorf("host cannot be empty")
		}
		s.host = host
		return nil
	}
}

// WithPort sets the server port.
func WithPort(port int) Option {
	return func(s *Server) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %d", port)
		}
		s.port = port
		return nil
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(s *Server) error {
		if d <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		s.timeout = d
		return nil
	}
}

// WithMaxConns sets the connection limit.
func WithMaxConns(n int) Option {
	return func(s *Server) error {
		if n <= 0 {
			return fmt.Errorf("maxConns must be positive")
		}
		s.maxConns = n
		return nil
	}
}

// WithLogger injects a logger.
func WithLogger(l Logger) Option {
	return func(s *Server) error {
		if l == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		s.logger = l
		return nil
	}
}

// WithTLS enables TLS.
func WithTLS() Option {
	return func(s *Server) error {
		s.tlsEnabled = true
		return nil
	}
}

// WithRetries sets retry count.
func WithRetries(n int) Option {
	return func(s *Server) error {
		s.retries = n
		return nil
	}
}

// NewServer applies defaults, then options, then validates.
func NewServer(opts ...Option) (*Server, error) {
	s := &Server{
		host:     "localhost",
		port:     8080,
		timeout:  30 * time.Second,
		maxConns: 100,
		retries:  3,
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("NewServer: %w", err)
		}
	}
	return s, nil
}

func (s *Server) String() string {
	scheme := "http"
	if s.tlsEnabled {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d (timeout=%s maxConns=%d retries=%d)",
		scheme, s.host, s.port, s.timeout, s.maxConns, s.retries)
}

func (s *Server) log(msg string) {
	if s.logger != nil {
		s.logger.Log(msg)
	}
}

// ─── Simple logger implementations ───────────────────────────────────────────

type stdoutLogger struct{}

func (l *stdoutLogger) Log(msg string) { fmt.Println("[LOG]", msg) }

type noopLogger struct{}

func (l *noopLogger) Log(_ string) {}

func main() {
	// ── defaults only ──
	s1, _ := NewServer()
	fmt.Println("defaults:", s1)

	// ── production config ──
	s2, err := NewServer(
		WithHost("api.example.com"),
		WithPort(443),
		WithTLS(),
		WithTimeout(10*time.Second),
		WithMaxConns(1000),
		WithLogger(&stdoutLogger{}),
	)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("production:", s2)
		s2.log("server configured")
	}

	fmt.Println()

	// ── test config: minimal, noop logger ──
	s3, _ := NewServer(
		WithHost("testhost"),
		WithLogger(&noopLogger{}),
	)
	fmt.Println("test:", s3)

	fmt.Println()

	// ── validation catches bad options ──
	_, err = NewServer(WithPort(-1))
	fmt.Println("bad port:", err)

	_, err = NewServer(WithTimeout(-1 * time.Second))
	fmt.Println("bad timeout:", err)

	fmt.Println()

	// ── options are first-class: compose a preset ──
	productionPreset := []Option{
		WithTLS(),
		WithTimeout(10 * time.Second),
		WithMaxConns(1000),
		WithRetries(5),
	}
	s4, _ := NewServer(append(productionPreset, WithHost("prod.example.com"), WithPort(443))...)
	fmt.Println("preset:", s4)
}
