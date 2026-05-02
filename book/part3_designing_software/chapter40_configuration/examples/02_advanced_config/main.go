// FILE: book/part3_designing_software/chapter40_configuration/examples/02_advanced_config/main.go
// CHAPTER: 40 — Configuration
// TOPIC: Functional options pattern for config, secrets redaction,
//        config watchers (hot-reload simulation), and feature flags.
//
// Run (from the chapter folder):
//   go run ./examples/02_advanced_config

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// FUNCTIONAL OPTIONS
// ─────────────────────────────────────────────────────────────────────────────

type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	APIKey      string
	UserAgent   string
	RateLimit   int // requests per second; 0 = unlimited
	TLSInsecure bool
}

type ClientOption func(*ClientConfig)

func WithTimeout(d time.Duration) ClientOption {
	return func(c *ClientConfig) { c.Timeout = d }
}

func WithMaxRetries(n int) ClientOption {
	return func(c *ClientConfig) { c.MaxRetries = n }
}

func WithAPIKey(key string) ClientOption {
	return func(c *ClientConfig) { c.APIKey = key }
}

func WithUserAgent(ua string) ClientOption {
	return func(c *ClientConfig) { c.UserAgent = ua }
}

func WithRateLimit(rps int) ClientOption {
	return func(c *ClientConfig) { c.RateLimit = rps }
}

// NewClientConfig applies functional options on top of safe defaults.
func NewClientConfig(baseURL string, opts ...ClientOption) ClientConfig {
	cfg := ClientConfig{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		UserAgent:  "go-client/1.0",
		RateLimit:  100,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// ─────────────────────────────────────────────────────────────────────────────
// SECRETS — redacted String() so secrets never appear in logs
// ─────────────────────────────────────────────────────────────────────────────

type Secret string

func (s Secret) String() string {
	if len(s) == 0 {
		return "<empty>"
	}
	return "<redacted>"
}

func (s Secret) Value() string { return string(s) }

type DBCredentials struct {
	Host     string
	Port     int
	User     string
	Password Secret
	DBName   string
}

func (c DBCredentials) String() string {
	return fmt.Sprintf("postgres://%s:***@%s:%d/%s", c.User, c.Host, c.Port, c.DBName)
}

// DSN returns the actual connection string — only called at connection time.
func (c DBCredentials) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		c.User, c.Password.Value(), c.Host, c.Port, c.DBName)
}

// ─────────────────────────────────────────────────────────────────────────────
// FEATURE FLAGS — lightweight boolean flags with description
// ─────────────────────────────────────────────────────────────────────────────

type FeatureFlags struct {
	mu    sync.RWMutex
	flags map[string]bool
}

func NewFeatureFlags(initial map[string]bool) *FeatureFlags {
	flags := make(map[string]bool, len(initial))
	for k, v := range initial {
		flags[k] = v
	}
	return &FeatureFlags{flags: flags}
}

func (f *FeatureFlags) Enabled(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.flags[name]
}

func (f *FeatureFlags) Set(name string, value bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flags[name] = value
}

func (f *FeatureFlags) All() map[string]bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make(map[string]bool, len(f.flags))
	for k, v := range f.flags {
		out[k] = v
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// CONFIG WATCHER — hot-reload simulation
// ─────────────────────────────────────────────────────────────────────────────

type WatchableConfig struct {
	mu       sync.RWMutex
	current  map[string]string
	watchers []func(key, newValue string)
}

func NewWatchableConfig(initial map[string]string) *WatchableConfig {
	snap := make(map[string]string, len(initial))
	for k, v := range initial {
		snap[k] = v
	}
	return &WatchableConfig{current: snap}
}

func (w *WatchableConfig) Get(key string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.current[key]
}

func (w *WatchableConfig) OnChange(fn func(key, newValue string)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.watchers = append(w.watchers, fn)
}

// Update simulates a remote config push (e.g., from Consul or etcd).
func (w *WatchableConfig) Update(key, value string) {
	w.mu.Lock()
	w.current[key] = value
	watchers := make([]func(string, string), len(w.watchers))
	copy(watchers, w.watchers)
	w.mu.Unlock()

	for _, fn := range watchers {
		fn(key, value)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO
// ─────────────────────────────────────────────────────────────────────────────

func demoFunctionalOptions() {
	fmt.Println("=== Functional Options ===")

	// Minimal — just a base URL.
	minimal := NewClientConfig("https://api.example.com")
	fmt.Printf("  minimal:  timeout=%s retries=%d rateLimit=%d\n",
		minimal.Timeout, minimal.MaxRetries, minimal.RateLimit)

	// Custom overrides.
	custom := NewClientConfig("https://api.payments.io",
		WithTimeout(30*time.Second),
		WithMaxRetries(5),
		WithAPIKey("sk-live-abc123"),
		WithRateLimit(50),
	)
	fmt.Printf("  custom:   timeout=%s retries=%d rateLimit=%d apiKey=%s\n",
		custom.Timeout, custom.MaxRetries, custom.RateLimit,
		// Redact the key in log output — print only first 7 chars.
		custom.APIKey[:7]+"…",
	)
}

func demoSecrets() {
	fmt.Println()
	fmt.Println("=== Secrets Redaction ===")

	creds := DBCredentials{
		Host:     "db.prod.internal",
		Port:     5432,
		User:     "app_user",
		Password: "s3cr3t-p4ssw0rd",
		DBName:   "commerce",
	}

	// This is what would end up in logs — password never visible.
	fmt.Printf("  logged:  %s\n", creds)
	fmt.Printf("  password field: %s\n", creds.Password)

	dsn := creds.DSN()
	// Show partial DSN (would never log the full DSN in production).
	idx := strings.Index(dsn, "@")
	fmt.Printf("  DSN host part: %s\n", dsn[idx+1:])
}

func demoFeatureFlags() {
	fmt.Println()
	fmt.Println("=== Feature Flags ===")

	flags := NewFeatureFlags(map[string]bool{
		"new_checkout_flow":   false,
		"dark_mode":           true,
		"experimental_search": false,
	})

	fmt.Printf("  new_checkout_flow:   %v\n", flags.Enabled("new_checkout_flow"))
	fmt.Printf("  dark_mode:           %v\n", flags.Enabled("dark_mode"))
	fmt.Printf("  experimental_search: %v\n", flags.Enabled("experimental_search"))

	// Simulate a remote flag flip.
	flags.Set("new_checkout_flow", true)
	fmt.Printf("  [flipped] new_checkout_flow: %v\n", flags.Enabled("new_checkout_flow"))
}

func demoWatcher() {
	fmt.Println()
	fmt.Println("=== Config Watcher (hot-reload) ===")

	cfg := NewWatchableConfig(map[string]string{
		"log_level": "info",
		"rate_limit": "100",
	})

	var changes []string
	cfg.OnChange(func(key, newValue string) {
		changes = append(changes, fmt.Sprintf("%s→%s", key, newValue))
	})

	fmt.Printf("  initial log_level: %s\n", cfg.Get("log_level"))

	// Simulate remote updates arriving.
	cfg.Update("log_level", "debug")
	cfg.Update("rate_limit", "50")

	fmt.Printf("  updated log_level: %s\n", cfg.Get("log_level"))
	fmt.Printf("  updated rate_limit: %s\n", cfg.Get("rate_limit"))
	fmt.Printf("  change events: %v\n", changes)
}

func main() {
	demoFunctionalOptions()
	demoSecrets()
	demoFeatureFlags()
	demoWatcher()
}
