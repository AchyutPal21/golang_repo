// FILE: book/part3_designing_software/chapter40_configuration/exercises/01_app_config/main.go
// CHAPTER: 40 — Configuration
// EXERCISE: Full application config: layered loading (defaults → JSON → env),
//           functional options for an HTTP client, secrets, and feature flags.
//
// Run (from the chapter folder):
//   go run ./exercises/01_app_config

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Secret string

func (s Secret) String() string {
	if len(s) == 0 {
		return "<empty>"
	}
	return "<redacted>"
}
func (s Secret) Value() string { return string(s) }

type CacheConfig struct {
	Host    string        `json:"host"`
	Port    int           `json:"port"`
	TTL     time.Duration `json:"ttl"`
	MaxKeys int           `json:"max_keys"`
}

type AuthConfig struct {
	JWTSecret     Secret        `json:"-"` // never in JSON; from env only
	TokenDuration time.Duration `json:"token_duration"`
	BcryptCost    int           `json:"bcrypt_cost"`
}

type Config struct {
	Env          string            `json:"env"`
	LogLevel     string            `json:"log_level"`
	Cache        CacheConfig       `json:"cache"`
	Auth         AuthConfig        `json:"auth"`
	FeatureFlags map[string]bool   `json:"feature_flags"`
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYERED LOADING
// ─────────────────────────────────────────────────────────────────────────────

func defaultConfig() Config {
	return Config{
		Env:      "development",
		LogLevel: "info",
		Cache: CacheConfig{
			Host:    "localhost",
			Port:    6379,
			TTL:     5 * time.Minute,
			MaxKeys: 10000,
		},
		Auth: AuthConfig{
			TokenDuration: 24 * time.Hour,
			BcryptCost:    12,
		},
		FeatureFlags: map[string]bool{
			"v2_api":     false,
			"rate_limit": true,
		},
	}
}

type partialCacheConfig struct {
	Host    *string        `json:"host"`
	Port    *int           `json:"port"`
	TTL     *time.Duration `json:"ttl"`
	MaxKeys *int           `json:"max_keys"`
}

type partialAuthConfig struct {
	TokenDuration *time.Duration `json:"token_duration"`
	BcryptCost    *int           `json:"bcrypt_cost"`
}

type partialConfig struct {
	Env          *string            `json:"env"`
	LogLevel     *string            `json:"log_level"`
	Cache        partialCacheConfig `json:"cache"`
	Auth         partialAuthConfig  `json:"auth"`
	FeatureFlags map[string]bool    `json:"feature_flags"`
}

func applyJSON(cfg *Config, data string) error {
	if strings.TrimSpace(data) == "" {
		return nil
	}
	var p partialConfig
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return fmt.Errorf("parse config JSON: %w", err)
	}
	if p.Env != nil {
		cfg.Env = *p.Env
	}
	if p.LogLevel != nil {
		cfg.LogLevel = *p.LogLevel
	}
	if p.Cache.Host != nil {
		cfg.Cache.Host = *p.Cache.Host
	}
	if p.Cache.Port != nil {
		cfg.Cache.Port = *p.Cache.Port
	}
	if p.Cache.TTL != nil {
		cfg.Cache.TTL = *p.Cache.TTL
	}
	if p.Cache.MaxKeys != nil {
		cfg.Cache.MaxKeys = *p.Cache.MaxKeys
	}
	if p.Auth.TokenDuration != nil {
		cfg.Auth.TokenDuration = *p.Auth.TokenDuration
	}
	if p.Auth.BcryptCost != nil {
		cfg.Auth.BcryptCost = *p.Auth.BcryptCost
	}
	for k, v := range p.FeatureFlags {
		cfg.FeatureFlags[k] = v
	}
	return nil
}

func applyEnv(cfg *Config, lookup func(string) (string, bool)) {
	str := func(dest *string, key string) {
		if v, ok := lookup(key); ok {
			*dest = v
		}
	}
	integer := func(dest *int, key string) {
		if v, ok := lookup(key); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	dur := func(dest *time.Duration, key string) {
		if v, ok := lookup(key); ok {
			if d, err := time.ParseDuration(v); err == nil {
				*dest = d
			}
		}
	}

	str(&cfg.Env, "APP_ENV")
	str(&cfg.LogLevel, "LOG_LEVEL")
	str(&cfg.Cache.Host, "CACHE_HOST")
	integer(&cfg.Cache.Port, "CACHE_PORT")
	dur(&cfg.Cache.TTL, "CACHE_TTL")
	integer(&cfg.Cache.MaxKeys, "CACHE_MAX_KEYS")

	// JWT secret always comes from environment — never from a file.
	if v, ok := lookup("JWT_SECRET"); ok {
		cfg.Auth.JWTSecret = Secret(v)
	}
	dur(&cfg.Auth.TokenDuration, "AUTH_TOKEN_DURATION")
	integer(&cfg.Auth.BcryptCost, "AUTH_BCRYPT_COST")

	// Feature flag overrides: FF_V2_API=true / FF_RATE_LIMIT=false
	for _, pair := range []struct{ env, flag string }{
		{"FF_V2_API", "v2_api"},
		{"FF_RATE_LIMIT", "rate_limit"},
	} {
		if v, ok := lookup(pair.env); ok {
			cfg.FeatureFlags[pair.flag] = v == "true" || v == "1"
		}
	}
}

func validate(cfg Config) error {
	valid := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !valid[cfg.LogLevel] {
		return fmt.Errorf("invalid log_level %q", cfg.LogLevel)
	}
	if cfg.Cache.Port < 1 || cfg.Cache.Port > 65535 {
		return fmt.Errorf("cache.port %d out of range", cfg.Cache.Port)
	}
	if cfg.Auth.BcryptCost < 4 || cfg.Auth.BcryptCost > 31 {
		return fmt.Errorf("auth.bcrypt_cost %d out of range [4,31]", cfg.Auth.BcryptCost)
	}
	return nil
}

func Load(jsonData string, lookup func(string) (string, bool)) (Config, error) {
	cfg := defaultConfig()
	if err := applyJSON(&cfg, jsonData); err != nil {
		return Config{}, err
	}
	applyEnv(&cfg, lookup)
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP CLIENT WITH FUNCTIONAL OPTIONS
// ─────────────────────────────────────────────────────────────────────────────

type HTTPClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	APIKey     Secret
	Headers    map[string]string
}

type HTTPClientOption func(*HTTPClientConfig)

func WithTimeout(d time.Duration) HTTPClientOption {
	return func(c *HTTPClientConfig) { c.Timeout = d }
}

func WithRetries(n int) HTTPClientOption {
	return func(c *HTTPClientConfig) { c.MaxRetries = n }
}

func WithAPIKey(key string) HTTPClientOption {
	return func(c *HTTPClientConfig) { c.APIKey = Secret(key) }
}

func WithHeader(k, v string) HTTPClientOption {
	return func(c *HTTPClientConfig) { c.Headers[k] = v }
}

func NewHTTPClientConfig(baseURL string, opts ...HTTPClientOption) HTTPClientConfig {
	cfg := HTTPClientConfig{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		Headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// 1. Load with defaults only.
	fmt.Println("=== Load: defaults only ===")
	cfg, err := Load("", func(string) (string, bool) { return "", false })
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("  env=%s log=%s cache=%s:%d ttl=%s\n",
		cfg.Env, cfg.LogLevel, cfg.Cache.Host, cfg.Cache.Port, cfg.Cache.TTL)
	fmt.Printf("  auth: duration=%s bcrypt_cost=%d jwt=%s\n",
		cfg.Auth.TokenDuration, cfg.Auth.BcryptCost, cfg.Auth.JWTSecret)
	fmt.Printf("  flags: %v\n\n", cfg.FeatureFlags)

	// 2. Load with JSON + env.
	fmt.Println("=== Load: JSON + env ===")
	fileJSON := `{"env":"production","log_level":"warn","cache":{"host":"redis.prod","max_keys":50000},"feature_flags":{"v2_api":true}}`
	envMap := map[string]string{
		"JWT_SECRET":          "super-secret-key",
		"AUTH_TOKEN_DURATION": "12h",
		"CACHE_PORT":          "6380",
		"FF_RATE_LIMIT":       "false",
	}
	cfg, err = Load(fileJSON, func(k string) (string, bool) {
		v, ok := envMap[k]
		return v, ok
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("  env=%s log=%s cache=%s:%d\n",
		cfg.Env, cfg.LogLevel, cfg.Cache.Host, cfg.Cache.Port)
	fmt.Printf("  auth: duration=%s jwt=%s\n", cfg.Auth.TokenDuration, cfg.Auth.JWTSecret)
	fmt.Printf("  flags: %v\n\n", cfg.FeatureFlags)

	// 3. HTTP client with functional options.
	fmt.Println("=== HTTP Client Functional Options ===")
	client := NewHTTPClientConfig("https://payments.api.io",
		WithTimeout(20*time.Second),
		WithRetries(5),
		WithAPIKey("sk-prod-xyz"),
		WithHeader("X-Request-ID", "req-001"),
		WithHeader("Accept", "application/json"),
	)
	fmt.Printf("  base=%s timeout=%s retries=%d\n",
		client.BaseURL, client.Timeout, client.MaxRetries)
	fmt.Printf("  api_key=%s headers=%v\n", client.APIKey, client.Headers)

	// 4. Validation failure.
	fmt.Println()
	fmt.Println("=== Validation failures ===")
	_, err = Load(`{"log_level":"trace"}`, func(string) (string, bool) { return "", false })
	fmt.Printf("  bad level: %v\n", err)

	_, err = Load(`{"auth":{"bcrypt_cost":100}}`, func(string) (string, bool) { return "", false })
	fmt.Printf("  bad bcrypt_cost: %v\n", err)
}
