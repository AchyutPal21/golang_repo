# Chapter 40 — Configuration

> **Part III · Designing Software** | Estimated reading time: 18 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Configuration is the boundary between your program and its environment. Done well, it gives you a single typed, validated `Config` struct that the rest of the program trusts implicitly. Done poorly, it scatters `os.Getenv` calls throughout the codebase and leaves secrets in log files.

---

## 40.1 — Layered configuration

The four layers, applied in order (each overrides the previous):

| Layer | Source | Notes |
|---|---|---|
| 1 | Code defaults | Always present; safe startup state |
| 2 | Config file | JSON/YAML/TOML; optional |
| 3 | Environment variables | 12-factor; per-deployment |
| 4 | Command-line flags | Highest priority; per-run |

The result is one immutable `Config` struct validated before any application code runs.

```go
func Load(jsonData string, envLookup func(string) (string, bool), args []string) (AppConfig, error) {
    cfg := defaults()
    applyFile(&cfg, jsonData)
    applyEnv(&cfg, envLookup)
    applyFlags(&cfg, args)
    return cfg, validate(cfg)
}
```

---

## 40.2 — Partial JSON override

Use pointer fields to distinguish "was present in JSON" from "was absent":

```go
type partialConfig struct {
    Host    *string        `json:"host"`    // nil = not in file, keep default
    Port    *int           `json:"port"`
    Timeout *time.Duration `json:"timeout"`
}
```

After unmarshalling, only apply non-nil fields to the config struct. This means your JSON file only needs to contain the fields you want to override.

---

## 40.3 — Functional options

Functional options let callers provide a variable number of named overrides without requiring all fields:

```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) { c.Timeout = d }
}

cfg := NewConfig("https://api.example.com",
    WithTimeout(30*time.Second),
    WithMaxRetries(5),
)
```

Rules:
- The constructor sets safe defaults before applying options.
- Each option is a function — not a value — so options can be computed, stored, and composed.
- Options with validation can return an error: `type Option func(*Config) error`.

---

## 40.4 — Secrets

Never log or marshal secrets. Wrap them in a type whose `String()` redacts:

```go
type Secret string

func (s Secret) String() string { return "<redacted>" }
func (s Secret) Value() string  { return string(s) }
```

Rules:
- Secrets should come only from environment variables (not config files checked into version control).
- Tag secret fields with `json:"-"` to prevent accidental serialisation.
- Only call `.Value()` where the raw string is genuinely needed (e.g., DSN construction).

---

## 40.5 — Feature flags

Lightweight boolean gates for in-process feature toggles:

```go
flags := NewFeatureFlags(map[string]bool{
    "new_checkout_flow": false,
    "dark_mode":         true,
})

if flags.Enabled("new_checkout_flow") {
    // new path
}
```

Protect with `sync.RWMutex` if flags can be updated at runtime (hot-reload from a remote config store).

---

## 40.6 — Config watchers

A `WatchableConfig` notifies registered callbacks when a key changes:

```go
cfg.OnChange(func(key, newValue string) {
    log.Printf("config updated: %s = %s", key, newValue)
})
cfg.Update("log_level", "debug") // triggers all callbacks
```

Used with remote config systems (Consul, etcd, AWS Parameter Store) that push updates at runtime.

---

## 40.7 — Validation

Always validate after all layers are applied, before returning the config:

```go
func validate(cfg Config) error {
    if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
        return fmt.Errorf("server.port %d out of range", cfg.Server.Port)
    }
    // ...
    return nil
}
```

A `Config` struct that survives `validate` is trusted throughout the program — no defensive checks elsewhere.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter40_configuration

go run ./examples/01_config_patterns  # layered load, JSON partial override, env, flags, validation
go run ./examples/02_advanced_config  # functional options, secrets, feature flags, watcher

go run ./exercises/01_app_config      # full app config: layered + functional options + secrets
```

---

## Key takeaways

1. **Four layers** in priority order: defaults → file → env → flags. Load once, validate once, then pass the immutable struct.
2. **Pointer fields** for partial JSON: nil means "not set, keep the default."
3. **Functional options** give callers a clean API without exposing every field in a constructor.
4. **Secrets** belong only in environment variables; wrap in a `Secret` type whose `String()` redacts.
5. **Feature flags** are just `map[string]bool` with an `RWMutex` — no library needed for simple use cases.
6. **Validation** lives in `Load` — the application never sees an invalid config.

---

## Cross-references

- **Chapter 39** — Encoding: `encoding/json` parses the config file
- **Chapter 28** — Dependency Injection: inject `Config` into services via constructor
- **Chapter 31** — Creational Patterns: functional options is the Builder pattern specialised for config
