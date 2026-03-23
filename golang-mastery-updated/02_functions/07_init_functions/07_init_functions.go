// 07_init_functions.go
//
// TOPIC: init() — Deep Dive: Multiple init per File, Init Order, Registration
//        Pattern, Init Pitfalls
//
// WHAT IS init()?
//   init() is a special function in Go that is automatically called by the
//   runtime before main() (or before any exported function is called in a
//   library package). You never call init() directly — the runtime handles it.
//
//   Signature: func init() — no parameters, no return values, cannot be called
//   explicitly (the compiler prevents it: "undefined: init").
//
// KEY PROPERTIES:
//   1. A single file can have MULTIPLE init() functions (unusual among languages)
//   2. A single package can have multiple files, each with their own init()s
//   3. init() functions run AFTER all variable declarations in the package are evaluated
//   4. init() functions in the same file run in the order they appear (top to bottom)
//   5. init() functions across files in a package run in the order the build
//      system presents the files (alphabetical by filename is the convention)
//   6. If package A imports package B, all of B's init()s run before A's init()s
//   7. Each init() runs only ONCE, even if the package is imported multiple times
//
// THE INITIALIZATION ORDER (per package):
//   Step 1: Initialize package-level variable declarations (in dependency order)
//   Step 2: Run all init() functions in source file order
//   Step 3: After all imported packages are initialized, the importing package proceeds
//
// WHY DOES GO ALLOW MULTIPLE init()?
//   It allows each file in a package to do its own local setup without
//   coordinating with other files. A file's init() is cohesive with the
//   declarations in that file.
//
// WHEN TO USE init():
//   - Registering drivers, codecs, plugins (the "registration pattern")
//   - Validating configuration or environment variables at startup
//   - Initializing package-level state that cannot be initialized with a simple
//     var declaration (e.g., requires error handling or complex logic)
//   - Computing lookup tables or caches at startup
//
// WHEN NOT TO USE init():
//   - When the initialization can fail in a way you want to report gracefully
//     (init() can only panic, not return an error)
//   - When you need to control initialization order precisely from the caller
//   - When the initialization depends on runtime configuration (use explicit
//     setup functions instead: func Initialize(cfg Config) error)
//   - In library packages where side effects in init() surprise users

package main

import (
	"fmt"
	"strings"
)

// ─── PACKAGE-LEVEL VARIABLES (initialized before init()) ─────────────────────
//
// Variable initializations are computed in dependency order BEFORE any init() runs.
// If var A depends on var B, B is initialized first regardless of declaration order.

var (
	// These are initialized in dependency order:
	// configPath → config → db (even if declared in different order)
	configPath = "/etc/app/config.yaml" // no dependencies
	config     = loadConfig(configPath) // depends on configPath
	db         = connectDB(config)      // depends on config

	// A lookup table computed once at startup
	squareTable = computeSquareTable(10)
)

// These are plain functions used during package-level var initialization.
// They run BEFORE any init(), as part of variable initialization.
func loadConfig(path string) map[string]string {
	// In real code: read a file, parse YAML, etc.
	return map[string]string{
		"dsn":  "postgres://localhost:5432/mydb",
		"path": path,
	}
}

func connectDB(cfg map[string]string) string {
	// In real code: open a database connection
	return fmt.Sprintf("DB<%s>", cfg["dsn"])
}

func computeSquareTable(n int) []int {
	table := make([]int, n+1)
	for i := range table {
		table[i] = i * i
	}
	return table
}

// ─── MULTIPLE init() IN ONE FILE ──────────────────────────────────────────────
//
// Go uniquely allows multiple init() functions in a single file.
// They run in the ORDER THEY APPEAR in the source file.
// This is useful when setup steps have a natural ordering within one file.

var initLog []string // tracks the order init() calls happen

func init() {
	// First init in this file — runs first
	initLog = append(initLog, "init #1: registering metrics")
	registerMetric("requests_total")
	registerMetric("errors_total")
}

func init() {
	// Second init in this file — runs after the first
	initLog = append(initLog, "init #2: validating config")
	if config["dsn"] == "" {
		panic("init: database DSN is required") // init can only panic, not return error
	}
}

func init() {
	// Third init — runs last among this file's inits
	initLog = append(initLog, "init #3: seeding random (simulated)")
	// In real code: rand.Seed(time.Now().UnixNano()) — though Go 1.20+ auto-seeds
}

// ─── REGISTRATION PATTERN ─────────────────────────────────────────────────────
//
// The most important and idiomatic use of init() in Go.
//
// Problem: How does database/sql know about "postgres", "mysql", "sqlite3" drivers
//          when it's a standard library package and drivers are third-party?
//
// Answer: The driver package registers itself in its init():
//
//   // In github.com/lib/pq (the postgres driver):
//   func init() {
//       sql.Register("postgres", &Driver{})
//   }
//
// The user imports the driver with a blank import:
//   import _ "github.com/lib/pq"
//
// The blank import "_" means: "I don't use any exported names from this package,
// but I want its init() to run." This triggers the side effect (registration)
// without polluting the namespace.
//
// Go's image package uses the same pattern:
//   import _ "image/png"  // registers the PNG decoder
//   import _ "image/jpeg" // registers the JPEG decoder

// A registry simulating the database/sql driver registry pattern.
var driverRegistry = make(map[string]DBDriver)

type DBDriver interface {
	Connect(dsn string) string
}

// RegisterDriver is called by drivers in their init() functions.
func RegisterDriver(name string, driver DBDriver) {
	if _, exists := driverRegistry[name]; exists {
		panic(fmt.Sprintf("RegisterDriver: driver %q already registered", name))
	}
	driverRegistry[name] = driver
	initLog = append(initLog, fmt.Sprintf("driver registered: %s", name))
}

// -- Simulated "postgres" driver package --
// In real code this would be a separate package with its own init()

type postgresDriver struct{}

func (d postgresDriver) Connect(dsn string) string {
	return "postgres connection: " + dsn
}

// This init() simulates what a driver package's init() does.
// In real Go code, this would be in a DIFFERENT package,
// and users would trigger it via:  import _ "myapp/internal/postgres"
func init() {
	RegisterDriver("postgres", postgresDriver{})
	initLog = append(initLog, "init #4: postgres driver registered")
}

// -- Simulated "mysql" driver --
type mysqlDriver struct{}

func (d mysqlDriver) Connect(dsn string) string {
	return "mysql connection: " + dsn
}

func init() {
	RegisterDriver("mysql", mysqlDriver{})
	initLog = append(initLog, "init #5: mysql driver registered")
}

// ─── METRIC REGISTRY ──────────────────────────────────────────────────────────
// Simulates Prometheus-style metric registration in init()

var metrics = make(map[string]int)

func registerMetric(name string) {
	metrics[name] = 0
}

func incrementMetric(name string) {
	metrics[name]++
}

// ─── INIT PITFALLS ────────────────────────────────────────────────────────────
//
// PITFALL 1: init() cannot return an error.
//   If initialization can fail, you must panic (which is severe) or use an
//   explicit setup function:
//     func Setup() error { ... }   ← called from main(), can return error
//
// PITFALL 2: Hard to test.
//   init() runs automatically when the package is imported. This means if your
//   init() has side effects (opens files, connects to DB), your tests will too.
//   This makes unit tests fragile and environment-dependent.
//   Mitigation: keep init() minimal — only register things, don't do I/O.
//
// PITFALL 3: Circular init dependencies.
//   Go detects initialization cycles and will fail to compile:
//     var a = b + 1
//     var b = a + 1  // compile error: initialization cycle
//
// PITFALL 4: Global state is hard to reason about.
//   init() typically sets up global (package-level) state. Global state makes
//   functions hard to test and reason about. Consider dependency injection instead.
//
// PITFALL 5: Import order surprises.
//   The order init()s run across packages depends on import order, which can
//   be surprising. Don't rely on one package's init() having run before another's
//   unless there's an explicit import relationship.

// explicitSetup demonstrates the "init() alternative" for testable code.
// This function can be called from main() or from tests, and returns an error.
type AppConfig struct {
	DSN     string
	Port    int
	LogPath string
}

func NewApp(cfg AppConfig) (*AppConfig, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("NewApp: DSN is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 8080 // default
	}
	// In real code: open DB connection, setup logger, etc.
	return &cfg, nil
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  init() FUNCTIONS — DEEP DIVE")
	fmt.Println(sep)

	// Show that package-level vars are initialized before init()
	fmt.Println("\n── Package-Level Variable Initialization ──")
	fmt.Println("  configPath:", configPath)
	fmt.Println("  config:    ", config)
	fmt.Println("  db:        ", db)
	fmt.Println("  squareTable[1..5]:", squareTable[1:6])

	// Show init() execution order
	fmt.Println("\n── init() Execution Order ──")
	fmt.Println("  The following init()s ran (in order) before main():")
	for i, entry := range initLog {
		fmt.Printf("    [%d] %s\n", i+1, entry)
	}

	// Registration pattern
	fmt.Println("\n── Registration Pattern (driver registry) ──")
	fmt.Println("  Registered drivers:")
	for name, driver := range driverRegistry {
		conn := driver.Connect("localhost:5432/mydb")
		fmt.Printf("    %-10s → %s\n", name, conn)
	}

	// Metric registry
	fmt.Println("\n── Metric Registry (registered via init) ──")
	incrementMetric("requests_total")
	incrementMetric("requests_total")
	incrementMetric("requests_total")
	incrementMetric("errors_total")
	for name, count := range metrics {
		fmt.Printf("    %-20s = %d\n", name, count)
	}

	// Explicit setup alternative
	fmt.Println("\n── Explicit Setup (Testable Alternative to init) ──")
	app, err := NewApp(AppConfig{DSN: "postgres://localhost/test", Port: 9090})
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  app created: dsn=%s port=%d\n", app.DSN, app.Port)
	}

	_, err = NewApp(AppConfig{}) // missing DSN
	fmt.Println("  missing DSN error:", err)

	fmt.Println("\n── Blank Import Pattern (conceptual demo) ──")
	fmt.Println("  In real code:")
	fmt.Println(`    import _ "github.com/lib/pq"      // registers postgres driver`)
	fmt.Println(`    import _ "image/png"               // registers PNG decoder`)
	fmt.Println("  The '_' triggers init() without importing any names.")
	fmt.Println("  This file simulates that with inline init() functions.")

	fmt.Println("\n── Initialization Cycle Detection ──")
	fmt.Println("  Go PREVENTS this at compile time:")
	fmt.Println("    var a = b + 1")
	fmt.Println("    var b = a + 1  // error: initialization cycle detected")

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • init() runs automatically: pkg vars → init() → main()")
	fmt.Println("  • Multiple init()s per file allowed; run top-to-bottom")
	fmt.Println("  • Imported package's init() always runs before importer's")
	fmt.Println("  • Registration pattern: driver/plugin registers itself in init()")
	fmt.Println("  • Blank import  _ 'pkg'  triggers init() without naming anything")
	fmt.Println("  • init() can only panic, not return error — keep it minimal")
	fmt.Println("  • For testable, configurable setup: use explicit func Setup() error")
	fmt.Println("  • Avoid heavy I/O or logic in init() — makes testing painful")
	fmt.Println(sep)
}
