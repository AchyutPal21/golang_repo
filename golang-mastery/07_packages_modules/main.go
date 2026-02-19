package main

// =============================================================================
// MODULE 07: PACKAGES & MODULES — Organizing Go code
// =============================================================================
// Run: go run 07_packages_modules/main.go
//
// This module explains:
//   - Package system (visibility, init, blank imports)
//   - Go modules (go.mod, go.sum, versioning)
//   - Toolchain commands (go build, test, vet, fmt, doc)
//   - Build tags
//   - Internal packages
// =============================================================================

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

// =============================================================================
// PACKAGE RULES — The most important things to know
// =============================================================================
//
// 1. PACKAGE NAME: first line of every .go file
//    package main    ← executable (must have func main())
//    package utils   ← library (cannot be run directly)
//
// 2. VISIBILITY: controlled by case of first letter
//    Uppercase = EXPORTED (public)  — Foo, Bar, MaxRetries
//    Lowercase = UNEXPORTED (private) — foo, bar, maxRetries
//    This applies to: types, functions, methods, variables, constants
//
// 3. PACKAGE vs DIRECTORY: package name should match directory name
//    But can differ (e.g., package main inside cmd/server/)
//
// 4. ONE package per directory (usually)
//    Exception: _test package for external tests
//
// 5. CIRCULAR imports are NOT allowed
//    A → B → A is a compile error
//    Solution: extract shared code into a third package C
//
// 6. IMPORT PATH: module_name/path/to/package
//    e.g., "github.com/gin-gonic/gin"
//         "golang-mastery/utils" (local package in same module)

// =============================================================================
// BLANK IMPORT — side effects only
// =============================================================================
// import _ "package/path"
// This runs the package's init() function but doesn't use any exports.
// Common use: registering database drivers, image decoders

// Example: import _ "github.com/lib/pq" — registers PostgreSQL driver

// =============================================================================
// ALIASED IMPORTS
// =============================================================================
// import (
//     "fmt"
//     myfmt "fmt"    ← alias
//     . "fmt"        ← dot import — use without package prefix (avoid!)
// )

// =============================================================================
// INIT FUNCTION — package-level initialization
// =============================================================================
var globalConfig map[string]string

func init() {
	// init() runs before main(), after all imports are initialized
	// Multiple init() per file/package — all run in order
	// Cannot be called manually
	globalConfig = map[string]string{
		"host": "localhost",
		"port": "8080",
		"env":  "development",
	}
	fmt.Println("[init] Global config initialized")
}

// =============================================================================
// DIRECTORY STRUCTURE FOR A REAL PROJECT
// =============================================================================
// Standard Go project layout:
//
// myproject/
// ├── go.mod              ← module definition
// ├── go.sum              ← dependency checksums (don't edit manually)
// ├── main.go             ← or cmd/myapp/main.go
// ├── cmd/                ← multiple executables
// │   ├── server/
// │   │   └── main.go
// │   └── cli/
// │       └── main.go
// ├── internal/           ← private packages (only importable within module)
// │   └── db/
// │       └── db.go
// ├── pkg/                ← public reusable packages
// │   └── utils/
// │       └── utils.go
// ├── api/                ← API definitions (proto, swagger, etc.)
// ├── configs/            ← configuration files
// ├── scripts/            ← build/deploy scripts
// └── tests/              ← integration/e2e tests

// =============================================================================
// GO MODULES — dependency management
// =============================================================================
// Modules are the unit of versioning and distribution.
//
// go mod init github.com/yourname/yourproject
//   → creates go.mod
//
// go.mod format:
//   module github.com/yourname/yourproject
//   go 1.22
//   require (
//       github.com/gin-gonic/gin v1.9.1
//       github.com/stretchr/testify v1.8.4
//   )
//
// COMMANDS:
//   go get package@version  ← add/update dependency
//   go mod tidy             ← add missing, remove unused dependencies
//   go mod download         ← download all dependencies
//   go mod verify           ← verify checksums
//   go mod graph            ← dependency graph
//   go mod vendor           ← copy deps to vendor/ directory
//   go list -m all          ← list all dependencies

// =============================================================================
// BUILD TAGS — conditional compilation
// =============================================================================
// Old style (Go < 1.17):
// // +build linux darwin
//
// New style (Go 1.17+):
// //go:build linux || darwin
//
// Common tags: linux, darwin, windows, amd64, arm64
// Custom tags: go build -tags production ./...

// =============================================================================
// TOOLCHAIN — essential commands
// =============================================================================
// go build ./...           ← compile everything
// go run main.go           ← compile and run
// go test ./...            ← run all tests
// go test -v ./...         ← verbose
// go test -race ./...      ← race detector
// go test -bench=. ./...   ← benchmarks
// go test -cover ./...     ← coverage
// go vet ./...             ← static analysis (catches bugs)
// go fmt ./...             ← format code (or use gofmt/goimports)
// go doc fmt.Println       ← view documentation
// go generate ./...        ← run //go:generate directives
// go clean -cache          ← clear build cache
// go env                   ← print Go environment
// go version               ← Go version
// go install tool@latest   ← install a tool

// =============================================================================
// DEMONSTRATING PACKAGE-LEVEL VISIBILITY
// =============================================================================

// Exported — visible from other packages
type Config struct {
	Host    string
	Port    int
	Debug   bool
	maxConn int // unexported — only visible inside this package
}

// NewConfig — constructor function (common Go pattern)
func NewConfig(host string, port int) *Config {
	return &Config{
		Host:    host,
		Port:    port,
		maxConn: 100, // can set unexported field from inside package
	}
}

// Exported method
func (c *Config) String() string {
	return fmt.Sprintf("%s:%d (debug=%v)", c.Host, c.Port, c.Debug)
}

// Unexported method — only callable within this package
func (c *Config) validate() bool {
	return c.Port > 0 && c.Port < 65536
}

// =============================================================================
// INTERNAL PACKAGE EXAMPLE
// =============================================================================
// Files in internal/ can only be imported by code in the parent directory.
// e.g., mymodule/internal/db can only be imported by mymodule/...
// External packages cannot import internal/ packages.

// =============================================================================
// PRACTICAL: Working with os and environment
// =============================================================================

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// =============================================================================
// PRACTICAL: Demonstrating how package design works
// =============================================================================

// Functional table — shows how to organize related functions
type StringProcessor struct {
	transforms []func(string) string
}

func NewStringProcessor() *StringProcessor {
	return &StringProcessor{}
}

func (sp *StringProcessor) Add(fn func(string) string) *StringProcessor {
	sp.transforms = append(sp.transforms, fn)
	return sp
}

func (sp *StringProcessor) Process(s string) string {
	for _, fn := range sp.transforms {
		s = fn(s)
	}
	return s
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 07: PACKAGES & MODULES ===")

	fmt.Println("\n--- Global Config (set by init) ---")
	for k, v := range globalConfig {
		fmt.Printf("  %s = %s\n", k, v)
	}

	// -------------------------------------------------------------------------
	// SECTION 1: Package visibility
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Package Visibility ---")

	cfg := NewConfig("localhost", 8080)
	cfg.Debug = true
	fmt.Println("Config:", cfg)
	fmt.Println("Valid:", cfg.validate()) // accessible from same package

	// -------------------------------------------------------------------------
	// SECTION 2: Environment variables
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Environment Variables ---")

	host := getEnv("APP_HOST", "localhost")
	port := getEnv("APP_PORT", "8080")
	env := getEnv("APP_ENV", "development")

	fmt.Printf("App running on %s:%s in %s mode\n", host, port, env)

	// os.Getenv — simpler but returns "" for missing (can't tell missing from empty)
	home := os.Getenv("HOME")
	fmt.Println("HOME:", home)

	// All environment variables
	// envVars := os.Environ() // returns []string of "KEY=VALUE"

	// -------------------------------------------------------------------------
	// SECTION 3: The rand package — show package usage
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Using Packages ---")

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 5; i++ {
		fmt.Printf("random: %d\n", r.Intn(100))
	}

	// -------------------------------------------------------------------------
	// SECTION 4: String processor — package-like design
	// -------------------------------------------------------------------------
	fmt.Println("\n--- String Processor (Package Design Demo) ---")

	sp := NewStringProcessor().
		Add(strings.TrimSpace).
		Add(strings.ToLower).
		Add(func(s string) string {
			return strings.ReplaceAll(s, " ", "_")
		})

	inputs := []string{
		"  Hello World  ",
		"  GO IS AWESOME  ",
		"  Package Design  ",
	}
	for _, input := range inputs {
		fmt.Printf("%q → %q\n", input, sp.Process(input))
	}

	// -------------------------------------------------------------------------
	// SECTION 5: Key commands reference
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Essential Go Commands ---")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"go mod init <module>", "Create a new module"},
		{"go mod tidy", "Add missing, remove unused deps"},
		{"go get pkg@version", "Add/update a dependency"},
		{"go build ./...", "Build all packages"},
		{"go run main.go", "Build and run"},
		{"go test ./...", "Run all tests"},
		{"go test -race ./...", "Run tests with race detector"},
		{"go test -cover ./...", "Run tests with coverage"},
		{"go test -bench=. ./...", "Run benchmarks"},
		{"go vet ./...", "Static analysis"},
		{"go fmt ./...", "Format all code"},
		{"go doc <pkg>.<symbol>", "View documentation"},
		{"go clean -cache", "Clear build cache"},
		{"go list -m all", "List all dependencies"},
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].cmd < commands[j].cmd
	})

	for _, c := range commands {
		fmt.Printf("  %-35s %s\n", c.cmd, c.desc)
	}

	// -------------------------------------------------------------------------
	// SECTION 6: go:generate directive
	// -------------------------------------------------------------------------
	fmt.Println("\n--- go:generate ---")
	fmt.Println(`
//go:generate stringer -type=Direction
type Direction int
const ( North Direction = iota; South; East; West )

Running 'go generate' executes: stringer -type=Direction
This generates a String() method for the Direction type.

Common generators:
  stringer        — String() methods for enums
  mockgen         — mock interfaces for testing
  protoc          — Protocol Buffer code
  wire            — dependency injection
  sqlc            — type-safe SQL code
`)

	// -------------------------------------------------------------------------
	// SECTION 7: Module proxy and GOPATH
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Module Concepts ---")
	fmt.Println(`
GOPATH (legacy):
  ~/go/src   — source code
  ~/go/pkg   — compiled packages
  ~/go/bin   — installed binaries

Module cache: ~/go/pkg/mod/

GOPROXY:
  Go downloads modules through a proxy (default: proxy.golang.org)
  GOPROXY=direct — bypass proxy (downloads directly from VCS)
  GOPROXY=off    — disallow network downloads

GONOSUMCHECK / GONOSUMDB / GOFLAGS — other module env vars

Private modules:
  GONOSUMCHECK=github.com/mycompany/*
  GOPRIVATE=github.com/mycompany/*
`)

	fmt.Println("=== MODULE 07 COMPLETE ===")
}
