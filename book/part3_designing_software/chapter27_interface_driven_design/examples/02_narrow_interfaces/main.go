// FILE: book/part3_designing_software/chapter27_interface_driven_design/examples/02_narrow_interfaces/main.go
// CHAPTER: 27 — Interface-Driven Design
// TOPIC: Prefer narrow (1-2 method) interfaces. Interface composition.
//        The role model — stdlib interfaces as archetypes.
//
// Run (from the chapter folder):
//   go run ./examples/02_narrow_interfaces

package main

import (
	"fmt"
	"io"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// The stdlib interface model: small, single-purpose, composable.
//
//   io.Reader:   Read(p []byte) (n int, err error)
//   io.Writer:   Write(p []byte) (n int, err error)
//   io.Closer:   Close() error
//   io.Seeker:   Seek(offset int64, whence int) (int64, error)
//   io.ReadWriter    = Reader + Writer
//   io.ReadWriteCloser = Reader + Writer + Closer
//
// Design your own interfaces the same way.
// ─────────────────────────────────────────────────────────────────────────────

// ─── Cache system: narrow interfaces ─────────────────────────────────────────

type Getter interface {
	Get(key string) ([]byte, bool)
}

type Setter interface {
	Set(key string, value []byte)
}

type Deleter interface {
	Delete(key string)
}

// ReadCache is what a read-only consumer needs.
type ReadCache interface {
	Getter
}

// WriteCache is what a write-only consumer needs (e.g., cache warmer).
type WriteCache interface {
	Setter
}

// Cache is the full interface — used by consumers that need all operations.
type Cache interface {
	Getter
	Setter
	Deleter
}

// ─── Concrete implementation ──────────────────────────────────────────────────

type memCache struct {
	data map[string][]byte
}

func newMemCache() *memCache {
	return &memCache{data: make(map[string][]byte)}
}

func (c *memCache) Get(key string) ([]byte, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *memCache) Set(key string, value []byte) {
	c.data[key] = value
}

func (c *memCache) Delete(key string) {
	delete(c.data, key)
}

// ─── Consumers using narrow interfaces ───────────────────────────────────────

// renderPage only reads — receives ReadCache.
func renderPage(key string, cache ReadCache) string {
	if data, ok := cache.Get(key); ok {
		return "[cached] " + string(data)
	}
	return "[miss] " + key
}

// warmCache only writes — receives WriteCache.
func warmCache(pages map[string]string, cache WriteCache) {
	for k, v := range pages {
		cache.Set(k, []byte(v))
	}
}

// invalidate deletes — receives Deleter.
func invalidate(keys []string, d Deleter) {
	for _, k := range keys {
		d.Delete(k)
	}
}

// ─── io.Writer pipeline: narrow interface enables composition ─────────────────

// prefixWriter wraps any io.Writer and prepends a prefix to each line.
type prefixWriter struct {
	w      io.Writer
	prefix string
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(p.w, "%s%s\n", p.prefix, line); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}

// countingWriter counts bytes written.
type countingWriter struct {
	w     io.Writer
	count int
}

func (c *countingWriter) Write(data []byte) (int, error) {
	n, err := c.w.Write(data)
	c.count += n
	return n, err
}

func main() {
	mc := newMemCache()

	// Warm via WriteCache interface
	warmCache(map[string]string{
		"/home":  "<html>Home Page</html>",
		"/about": "<html>About Us</html>",
	}, mc)

	// Render via ReadCache interface
	fmt.Println(renderPage("/home", mc))
	fmt.Println(renderPage("/contact", mc))

	// Invalidate via Deleter interface
	invalidate([]string{"/home"}, mc)
	fmt.Println(renderPage("/home", mc)) // now a miss

	fmt.Println()

	// io.Writer pipeline
	var buf strings.Builder
	cw := &countingWriter{w: &buf}
	pw := &prefixWriter{w: cw, prefix: ">> "}

	fmt.Fprintln(pw, "first line")
	fmt.Fprintln(pw, "second line")
	fmt.Fprintln(pw, "third line")

	fmt.Print(buf.String())
	fmt.Println("bytes counted:", cw.count)

	fmt.Println()
	fmt.Println("Design rule: the smaller the interface, the more types can satisfy it.")
	fmt.Println("  io.Reader is satisfied by 50+ types in the stdlib alone.")
}
