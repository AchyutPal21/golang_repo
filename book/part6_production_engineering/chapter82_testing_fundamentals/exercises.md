# Chapter 82 Exercises — Testing Fundamentals

## Exercise 1 — URL Shortener Test Suite (`exercises/01_test_suite`)

Build a complete test suite for a URL shortener service demonstrating every testing pattern from this chapter.

### System under test

**`Shortener`**:
```go
func NewShortener() *Shortener
func (s *Shortener) Shorten(long string) (string, error)
func (s *Shortener) ShortenWithCode(long, customCode string) (string, error)
func (s *Shortener) Resolve(code string) (string, error)
func (s *Shortener) Clicks(code string) (int64, error)
func (s *Shortener) Count() int
```

### Error types

- `ErrInvalidURL` — URL doesn't start with `http://` or `https://`
- `ErrNotFound` — short code doesn't exist
- `ErrCodeTaken` — custom code already registered

### Test groups to implement

**Shorten validation** (table-driven):
- Valid `https://` URL → no error
- Valid `http://` URL → no error
- URL without scheme → `ErrInvalidURL`
- FTP URL → `ErrInvalidURL`
- Empty string → `ErrInvalidURL`

**Resolve**:
- Found → returns original URL
- Custom code → resolves correctly
- Not found → `ErrNotFound`
- Each resolve increments click counter

**Custom codes**:
- Custom code accepted and returned verbatim
- Duplicate custom code → `ErrCodeTaken`

**Idempotency**:
- Shortening the same URL twice returns the same code and doesn't create two entries

**Property-based**:
- `resolve(shorten(url)) == url` for any valid URL
- 10 concurrent goroutines shortening the same URL produce the same code
- Click count equals exact number of Resolve calls

### Hints

- Use a shared fixture (`newFixture()`) that pre-seeds two entries; each test group reads from it
- The idempotency test verifies `Count() == 1` after two identical Shorten calls
- For the concurrency property test, compare all 10 results against `results[0]`
