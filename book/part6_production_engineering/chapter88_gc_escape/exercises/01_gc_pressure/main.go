// FILE: book/part6_production_engineering/chapter88_gc_escape/exercises/01_gc_pressure/main.go
// CHAPTER: 88 — GC & Escape Analysis
// EXERCISE: Identify and fix GC pressure hotspots in a simulated HTTP
//   middleware pipeline. Measure before/after GC cycles and pause time.
//
// Run:
//   go run ./part6_production_engineering/chapter88_gc_escape/exercises/01_gc_pressure/

package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED REQUEST / RESPONSE
// ─────────────────────────────────────────────────────────────────────────────

type Header struct{ Key, Value string }

type Request struct {
	Method  string
	Path    string
	Headers []Header
	Body    []byte
}

type Response struct {
	Status  int
	Headers []Header
	Body    []byte
}

// ─────────────────────────────────────────────────────────────────────────────
// BEFORE: allocations in every hot-path step
// ─────────────────────────────────────────────────────────────────────────────

// parsePathSlow splits a URL path and returns a new []string each call.
func parsePathSlow(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

// buildLogLineSlow concatenates with fmt.Sprintf — always allocates.
func buildLogLineSlow(req *Request, status int, dur time.Duration) string {
	return fmt.Sprintf("%s %s %d %v", req.Method, req.Path, status, dur)
}

// processRequestSlow creates a new Response{} for every request.
func processRequestSlow(req *Request) *Response {
	segments := parsePathSlow(req.Path)
	var body strings.Builder
	body.WriteString(`{"path":"`)
	body.WriteString(segments[0])
	body.WriteString(`","ok":true}`)

	return &Response{
		Status: 200,
		Headers: []Header{
			{"Content-Type", "application/json"},
			{"X-Segments", strconv.Itoa(len(segments))},
		},
		Body: []byte(body.String()),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AFTER: allocation-reduced versions
// ─────────────────────────────────────────────────────────────────────────────

// parsePathFast writes path segments into caller-provided slice.
// Returns number of segments written.
func parsePathFast(path string, out []string) int {
	path = strings.TrimPrefix(path, "/")
	n := 0
	for {
		i := strings.IndexByte(path, '/')
		if i < 0 {
			if len(path) > 0 && n < len(out) {
				out[n] = path
				n++
			}
			break
		}
		if n < len(out) {
			out[n] = path[:i]
			n++
		}
		path = path[i+1:]
	}
	return n
}

// buildLogLineFast appends into caller-provided buffer — zero alloc.
func buildLogLineFast(dst []byte, method, path string, status int, dur time.Duration) []byte {
	dst = append(dst, method...)
	dst = append(dst, ' ')
	dst = append(dst, path...)
	dst = append(dst, ' ')
	dst = strconv.AppendInt(dst, int64(status), 10)
	dst = append(dst, ' ')
	dst = strconv.AppendInt(dst, dur.Milliseconds(), 10)
	dst = append(dst, "ms"...)
	return dst
}

// ResponsePool recycles Response objects.
var responsePool = responsePoolT{pool: make(chan *Response, 128)}

type responsePoolT struct{ pool chan *Response }

func (p *responsePoolT) Get() *Response {
	select {
	case r := <-p.pool:
		return r
	default:
		return &Response{
			Headers: make([]Header, 0, 4),
			Body:    make([]byte, 0, 256),
		}
	}
}

func (p *responsePoolT) Put(r *Response) {
	r.Status = 0
	r.Headers = r.Headers[:0]
	r.Body = r.Body[:0]
	select {
	case p.pool <- r:
	default:
		// pool full — discard
	}
}

// processRequestFast reuses pooled response and stack-local path segments.
func processRequestFast(req *Request, logBuf []byte, segs []string) (*Response, []byte) {
	// Parse path into stack-local slice
	n := parsePathFast(req.Path, segs)
	if n == 0 {
		segs[0] = ""
		n = 1
	}

	resp := responsePool.Get()
	resp.Status = 200
	resp.Headers = append(resp.Headers[:0],
		Header{"Content-Type", "application/json"},
		Header{"X-Segments", strconv.Itoa(n)},
	)
	resp.Body = resp.Body[:0]
	resp.Body = append(resp.Body, `{"path":"`...)
	resp.Body = append(resp.Body, segs[0]...)
	resp.Body = append(resp.Body, `","ok":true}`...)

	logBuf = buildLogLineFast(logBuf[:0], req.Method, req.Path, resp.Status, 500*time.Microsecond)
	return resp, logBuf
}

// ─────────────────────────────────────────────────────────────────────────────
// MEASUREMENT
// ─────────────────────────────────────────────────────────────────────────────

type gcsnap struct {
	numGC uint32
	pause time.Duration
	alloc uint64
}

func gcsnap_read() gcsnap {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return gcsnap{m.NumGC, time.Duration(m.PauseTotalNs), m.TotalAlloc}
}
func (a gcsnap) diff(b gcsnap) gcsnap {
	return gcsnap{b.numGC - a.numGC, b.pause - a.pause, b.alloc - a.alloc}
}

const reqCount = 20_000

func benchSlow() gcsnap {
	reqs := makeFakeRequests(reqCount)
	runtime.GC()
	before := gcsnap_read()
	for _, req := range reqs {
		resp := processRequestSlow(req)
		_ = buildLogLineSlow(req, resp.Status, 500*time.Microsecond)
	}
	runtime.GC()
	return before.diff(gcsnap_read())
}

func benchFast() gcsnap {
	reqs := makeFakeRequests(reqCount)
	logBuf := make([]byte, 0, 128)
	segs := make([]string, 8)
	runtime.GC()
	before := gcsnap_read()
	for _, req := range reqs {
		resp, lb := processRequestFast(req, logBuf, segs)
		logBuf = lb
		_ = logBuf
		responsePool.Put(resp)
	}
	runtime.GC()
	return before.diff(gcsnap_read())
}

func makeFakeRequests(n int) []*Request {
	paths := []string{"/api/users", "/api/orders/42", "/health", "/metrics"}
	reqs := make([]*Request, n)
	for i := range reqs {
		reqs[i] = &Request{
			Method: "GET",
			Path:   paths[i%len(paths)],
			Headers: []Header{
				{"Accept", "application/json"},
				{"X-Request-ID", strconv.Itoa(i)},
			},
		}
	}
	return reqs
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 88 Exercise: GC Pressure Reduction ===")
	fmt.Println()

	// ── UNIT TEST ─────────────────────────────────────────────────────────────
	fmt.Println("--- Correctness check ---")
	req := &Request{Method: "GET", Path: "/api/users/42"}

	slowResp := processRequestSlow(req)
	logBuf := make([]byte, 0, 128)
	segs := make([]string, 8)
	fastResp, lb := processRequestFast(req, logBuf, segs)
	defer responsePool.Put(fastResp)

	fmt.Printf("  slow resp: status=%d body=%s\n", slowResp.Status, slowResp.Body)
	fmt.Printf("  fast resp: status=%d body=%s\n", fastResp.Status, fastResp.Body)
	fmt.Printf("  log line: %s\n", lb)
	fmt.Println()

	// ── PATH PARSING ──────────────────────────────────────────────────────────
	fmt.Println("--- Path parsing ---")
	segsOut := make([]string, 8)
	paths := []string{"/api/users/42/profile", "/health", "/a/b/c/d"}
	for _, p := range paths {
		n := parsePathFast(p, segsOut)
		fmt.Printf("  %q → %v (n=%d)\n", p, segsOut[:n], n)
	}
	fmt.Println()

	// ── GC PRESSURE COMPARISON ────────────────────────────────────────────────
	fmt.Println("--- GC pressure comparison (20k requests) ---")
	// warm up
	benchSlow()
	benchFast()

	ds := benchSlow()
	df := benchFast()
	fmt.Printf("  BEFORE fix: GC cycles=%-4d  pause=%-10v  alloc=%d KB\n",
		ds.numGC, ds.pause.Round(time.Microsecond), ds.alloc/1024)
	fmt.Printf("  AFTER  fix: GC cycles=%-4d  pause=%-10v  alloc=%d KB\n",
		df.numGC, df.pause.Round(time.Microsecond), df.alloc/1024)
	if ds.numGC > 0 {
		fmt.Printf("  GC cycle reduction: %.1f%%\n",
			100*(1-float64(df.numGC)/float64(ds.numGC)))
	}
	fmt.Println()

	fmt.Println("Fixes applied:")
	fmt.Println("  1. parsePathFast — caller-provided slice, no allocation")
	fmt.Println("  2. buildLogLineFast — strconv.AppendInt into caller buffer")
	fmt.Println("  3. responsePool — channel-based Response recycling")
	fmt.Println("  4. Response.Body reuse — append into pre-allocated slice")
}
