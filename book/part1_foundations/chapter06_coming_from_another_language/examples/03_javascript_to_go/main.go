// FILE: book/part1_foundations/chapter06_coming_from_another_language/examples/03_javascript_to_go/main.go
// CHAPTER: 06 — Coming From Another Language
// TOPIC: A JavaScript Promise.all program translated to Go goroutines.
//
// Run (from the chapter folder):
//   go run ./examples/03_javascript_to_go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The "fetch N URLs concurrently" pattern is a JavaScript idiom that
//   forces you to learn the goroutine + channel + WaitGroup model. We don't
//   actually fetch URLs — we simulate work with sleeps so the example runs
//   without a network. The pattern transfers verbatim to real HTTP.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

/*
JAVASCRIPT ORIGINAL:

    async function fetchAll(urls) {
        const results = await Promise.all(urls.map(async (url) => {
            const t0 = performance.now();
            await new Promise(r => setTimeout(r, Math.random() * 500));
            return { url, ms: performance.now() - t0 };
        }));
        results.sort((a, b) => b.ms - a.ms);
        return results;
    }

    const urls = ["a", "b", "c", "d", "e"];
    fetchAll(urls).then(results => {
        for (const r of results) {
            console.log(`${r.url}: ${r.ms.toFixed(0)}ms`);
        }
    });
*/

// result is the per-task outcome. JS would use an object literal; Go uses
// a struct.
type result struct {
	url string
	ms  int64
}

// fetchOne simulates the work for one URL: sleep a random time, return
// the elapsed milliseconds. In real code this would be an http.Get.
func fetchOne(url string) result {
	t0 := time.Now()
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return result{url: url, ms: time.Since(t0).Milliseconds()}
}

// fetchAll runs fetchOne concurrently for every URL and returns the
// results sorted by latency descending.
//
// The shape that maps to Promise.all:
//
//   - One goroutine per task (analogous to one async function call).
//   - A sync.WaitGroup tracks "have all goroutines finished?"
//     (analogous to Promise.all's "wait for all to settle").
//   - A buffered channel of `result` collects the outputs in any order
//     (analogous to "the array Promise.all returns to you").
//
// Differences from JS:
//   - Goroutines run on multiple OS threads truly in parallel. JS's
//     Promise.all interleaves on a single thread via the event loop.
//   - We pre-allocate the result channel; JS allocates the result array
//     when Promise.all resolves.
//   - There's no error type here, but in real code each task would
//     return (T, error) and we'd collect both.
func fetchAll(urls []string) []result {
	results := make(chan result, len(urls))

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, url := range urls {
		go func(u string) {
			defer wg.Done()
			results <- fetchOne(u)
		}(url) // pass by value to avoid the loop-variable trap (pre-1.22)
	}

	// Wait for every goroutine to call wg.Done(), then close the channel
	// so the range below terminates.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Drain the channel into a slice. In JS the Promise.all result IS the
	// array; in Go we collect it ourselves.
	out := make([]result, 0, len(urls))
	for r := range results {
		out = append(out, r)
	}

	// Sort descending by ms, then by url alphabetically for ties.
	sort.Slice(out, func(i, j int) bool {
		if out[i].ms != out[j].ms {
			return out[i].ms > out[j].ms
		}
		return out[i].url < out[j].url
	})
	return out
}

func main() {
	urls := []string{"a", "b", "c", "d", "e"}
	for _, r := range fetchAll(urls) {
		fmt.Printf("%s: %dms\n", r.url, r.ms)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Notice:
	//
	//   - The Go version is ~10 lines longer than the JS version. The
	//     extra lines are the WaitGroup + channel ceremony.
	//   - In exchange, the Go version: runs in genuine parallel, has
	//     statically-typed inputs/outputs, and never has a "promise that
	//     swallowed an error" bug.
	//   - For large N (say, 1000+), the Go version would also be much
	//     faster; JS is single-threaded under the hood and would
	//     interleave the sleeps.
	//
	// In Chapter 48 we'll meet errgroup.Group, which collapses the
	// WaitGroup + error-collection + cancellation into one type. The
	// pattern below is the "by hand" version; errgroup is what you'd
	// actually use in production.
	// ─────────────────────────────────────────────────────────────────────
}
