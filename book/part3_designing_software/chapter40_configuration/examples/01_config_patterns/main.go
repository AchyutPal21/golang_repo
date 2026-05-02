// FILE: book/part3_designing_software/chapter40_configuration/examples/01_config_patterns/main.go
// CHAPTER: 40 — Configuration
// TOPIC: Environment variables, JSON config files, functional options,
//        layered config (defaults → file → env → flags).
//
// Run (from the chapter folder):
//   go run ./examples/01_config_patterns

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONFIG STRUCT — typed, validated, immutable after construction
// ─────────────────────────────────────────────────────────────────────────────

type DatabaseConfig struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Name     string        `json:"name"`
	MaxConns int           `json:"max_conns"`
	Timeout  time.Duration `json:"timeout"`
}

type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

type AppConfig struct {
	Env      string         `json:"env"`
	LogLevel string         `json:"log_level"`
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 1 — defaults (always applied first)
// ─────────────────────────────────────────────────────────────────────────────

func defaults() AppConfig {
	return AppConfig{
		Env:      "development",
		LogLevel: "info",
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "app",
			MaxConns: 10,
			Timeout:  3 * time.Second,
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 2 — JSON file (partial: only fields present in the JSON override)
// ─────────────────────────────────────────────────────────────────────────────

// fileConfig is a mirror of AppConfig that uses pointers so we can detect
// which fields were present in the JSON versus absent.
type fileServerConfig struct {
	Host         *string        `json:"host"`
	Port         *int           `json:"port"`
	ReadTimeout  *time.Duration `json:"read_timeout"`
	WriteTimeout *time.Duration `json:"write_timeout"`
}

type fileDatabaseConfig struct {
	Host     *string        `json:"host"`
	Port     *int           `json:"port"`
	Name     *string        `json:"name"`
	MaxConns *int           `json:"max_conns"`
	Timeout  *time.Duration `json:"timeout"`
}

type fileConfig struct {
	Env      *string             `json:"env"`
	LogLevel *string             `json:"log_level"`
	Server   fileServerConfig    `json:"server"`
	Database fileDatabaseConfig  `json:"database"`
}

func applyFile(cfg *AppConfig, jsonData string) error {
	var fc fileConfig
	if err := json.Unmarshal([]byte(jsonData), &fc); err != nil {
		return fmt.Errorf("config file: %w", err)
	}
	if fc.Env != nil {
		cfg.Env = *fc.Env
	}
	if fc.LogLevel != nil {
		cfg.LogLevel = *fc.LogLevel
	}
	if fc.Server.Host != nil {
		cfg.Server.Host = *fc.Server.Host
	}
	if fc.Server.Port != nil {
		cfg.Server.Port = *fc.Server.Port
	}
	if fc.Server.ReadTimeout != nil {
		cfg.Server.ReadTimeout = *fc.Server.ReadTimeout
	}
	if fc.Server.WriteTimeout != nil {
		cfg.Server.WriteTimeout = *fc.Server.WriteTimeout
	}
	if fc.Database.Host != nil {
		cfg.Database.Host = *fc.Database.Host
	}
	if fc.Database.Port != nil {
		cfg.Database.Port = *fc.Database.Port
	}
	if fc.Database.Name != nil {
		cfg.Database.Name = *fc.Database.Name
	}
	if fc.Database.MaxConns != nil {
		cfg.Database.MaxConns = *fc.Database.MaxConns
	}
	if fc.Database.Timeout != nil {
		cfg.Database.Timeout = *fc.Database.Timeout
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 3 — environment variables
// ─────────────────────────────────────────────────────────────────────────────

func applyEnv(cfg *AppConfig, lookup func(string) (string, bool)) {
	set := func(dest *string, key string) {
		if v, ok := lookup(key); ok {
			*dest = v
		}
	}
	setInt := func(dest *int, key string) {
		if v, ok := lookup(key); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	setDur := func(dest *time.Duration, key string) {
		if v, ok := lookup(key); ok {
			if d, err := time.ParseDuration(v); err == nil {
				*dest = d
			}
		}
	}

	set(&cfg.Env, "APP_ENV")
	set(&cfg.LogLevel, "LOG_LEVEL")
	set(&cfg.Server.Host, "SERVER_HOST")
	setInt(&cfg.Server.Port, "SERVER_PORT")
	setDur(&cfg.Server.ReadTimeout, "SERVER_READ_TIMEOUT")
	setDur(&cfg.Server.WriteTimeout, "SERVER_WRITE_TIMEOUT")
	set(&cfg.Database.Host, "DB_HOST")
	setInt(&cfg.Database.Port, "DB_PORT")
	set(&cfg.Database.Name, "DB_NAME")
	setInt(&cfg.Database.MaxConns, "DB_MAX_CONNS")
	setDur(&cfg.Database.Timeout, "DB_TIMEOUT")
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 4 — command-line flags (simple key=value pairs for demo purposes)
// ─────────────────────────────────────────────────────────────────────────────

func applyFlags(cfg *AppConfig, args []string) {
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		switch key {
		case "--env":
			cfg.Env = val
		case "--log-level":
			cfg.LogLevel = val
		case "--server-port":
			if n, err := strconv.Atoi(val); err == nil {
				cfg.Server.Port = n
			}
		case "--db-host":
			cfg.Database.Host = val
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VALIDATION
// ─────────────────────────────────────────────────────────────────────────────

func validate(cfg AppConfig) error {
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid log_level %q: must be one of debug|info|warn|error", cfg.LogLevel)
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port %d out of range", cfg.Server.Port)
	}
	if cfg.Database.MaxConns < 1 {
		return fmt.Errorf("database.max_conns must be >= 1")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LOAD — compose all four layers
// ─────────────────────────────────────────────────────────────────────────────

func Load(jsonData string, envLookup func(string) (string, bool), args []string) (AppConfig, error) {
	cfg := defaults()
	if jsonData != "" {
		if err := applyFile(&cfg, jsonData); err != nil {
			return AppConfig{}, err
		}
	}
	applyEnv(&cfg, envLookup)
	applyFlags(&cfg, args)
	if err := validate(cfg); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO
// ─────────────────────────────────────────────────────────────────────────────

func printConfig(label string, cfg AppConfig) {
	fmt.Printf("  [%s]\n", label)
	fmt.Printf("    env=%-12s log_level=%s\n", cfg.Env, cfg.LogLevel)
	fmt.Printf("    server=%s:%d  read=%s write=%s\n",
		cfg.Server.Host, cfg.Server.Port,
		cfg.Server.ReadTimeout, cfg.Server.WriteTimeout)
	fmt.Printf("    db=%s:%d/%s max_conns=%d timeout=%s\n\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
		cfg.Database.MaxConns, cfg.Database.Timeout)
}

func main() {
	fmt.Println("=== Layered Configuration ===")
	fmt.Println()

	// Layer 1 only — pure defaults.
	cfg, _ := Load("", func(string) (string, bool) { return "", false }, nil)
	printConfig("defaults only", cfg)

	// Layer 1 + 2 — defaults overridden by JSON file.
	fileJSON := `{"env":"staging","log_level":"debug","server":{"port":9090},"database":{"host":"db.internal","max_conns":25}}`
	cfg, _ = Load(fileJSON, func(string) (string, bool) { return "", false }, nil)
	printConfig("defaults + file", cfg)

	// Layer 1 + 2 + 3 — JSON + env vars.
	envMap := map[string]string{
		"APP_ENV":    "production",
		"LOG_LEVEL":  "warn",
		"SERVER_PORT": "443",
		"DB_TIMEOUT": "5s",
	}
	cfg, _ = Load(fileJSON, func(k string) (string, bool) {
		v, ok := envMap[k]
		return v, ok
	}, nil)
	printConfig("defaults + file + env", cfg)

	// Layer 1 + 2 + 3 + 4 — everything including CLI flags.
	cfg, _ = Load(fileJSON, func(k string) (string, bool) {
		v, ok := envMap[k]
		return v, ok
	}, []string{"--log-level=error", "--server-port=8443"})
	printConfig("defaults + file + env + flags", cfg)

	// Validation failure.
	fmt.Println("=== Validation ===")
	_, err := Load(`{"log_level":"verbose"}`, os.LookupEnv, nil)
	fmt.Printf("  bad log_level error: %v\n", err)

	_, err = Load(`{"server":{"port":0}}`, os.LookupEnv, nil)
	fmt.Printf("  bad port error:      %v\n", err)
}
