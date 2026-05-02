# Chapter 40 — Exercises

## 40.1 — Full application config

Run [`exercises/01_app_config`](exercises/01_app_config/main.go).

Layered config with `CacheConfig`, `AuthConfig`, and feature flags. JWT secret comes only from the environment. HTTP client uses functional options.

Try:
- Add a `RateLimitConfig` struct with `RequestsPerSecond int` and `BurstSize int`. Wire it into the layered loader so it can be overridden from JSON and from the `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST` environment variables.
- Add an `WithInsecureTLS() HTTPClientOption` that sets a flag and make `validate` return an error if `InsecureTLS` is true and `Env == "production"`.
- Add a `Redact() Config` method that returns a copy of the config with all `Secret` fields zeroed — useful for structured logging of the startup configuration.

## 40.2 ★ — Config from multiple sources

Build a `MultiLoader` that accepts a slice of `Loader` values, each implementing:

```go
type Loader interface {
    Load() (map[string]string, error)
}
```

Apply them left-to-right (later loaders override earlier). Implement:
- `EnvLoader` — reads from `os.Environ()` filtered by a prefix
- `JSONFileLoader` — reads a flat key-value JSON file
- `DefaultsLoader` — returns a hardcoded map of defaults

Produce a final `map[string]string` and convert it to a typed `Config` with explicit parsing.

## 40.3 ★★ — Remote config with watch

Implement a `RemoteConfig` that simulates a remote key-value store (backed by a `map[string]string` and a `time.Ticker`). Every 2 seconds it checks for changes and fires registered `OnChange(key, value)` callbacks. Use it to hot-reload `LogLevel` and a feature flag during a short demo loop that prints the current values every 500ms for 5 seconds.
