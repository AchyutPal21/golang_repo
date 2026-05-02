// FILE: book/part4_concurrency_systems/chapter48_worker_pools/examples/02_errgroup_semaphore/main.go
// CHAPTER: 48 — Worker Pools
// TOPIC: errgroup.Group (golang.org/x/sync), semaphore-based concurrency
//        limit, and the scatter-gather pattern.
//        (We vendor errgroup-equivalent here to avoid external deps.)
//
// Run (from the chapter folder):
//   go run ./examples/02_errgroup_semaphore

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MINIMAL errgroup — equivalent to golang.org/x/sync/errgroup
// ─────────────────────────────────────────────────────────────────────────────

type Group struct {
	cancel  func(error)
	wg      sync.WaitGroup
	once    sync.Once
	err     error
}

func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{cancel: cancel}, ctx
}

func (g *Group) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.once.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(err)
				}
			})
		}
	}()
}

func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel(g.err)
	}
	return g.err
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: errgroup — all succeed
// ─────────────────────────────────────────────────────────────────────────────

func fetch(ctx context.Context, url string, delay time.Duration) (string, error) {
	select {
	case <-time.After(delay):
		return fmt.Sprintf("body of %s", url), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func demoErrGroup() {
	fmt.Println("=== errgroup: all succeed ===")

	ctx := context.Background()
	g, ctx := WithContext(ctx)

	urls := []string{"/api/users", "/api/orders", "/api/products"}
	bodies := make([]string, len(urls))
	delays := []time.Duration{20, 15, 25}

	for i, url := range urls {
		i, url, delay := i, url, delays[i]
		g.Go(func() error {
			body, err := fetch(ctx, url, delay*time.Millisecond)
			if err != nil {
				return fmt.Errorf("fetch %s: %w", url, err)
			}
			bodies[i] = body
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("  error:", err)
	} else {
		for _, b := range bodies {
			fmt.Println(" ", b)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: errgroup — first error cancels the group
// ─────────────────────────────────────────────────────────────────────────────

func fetchWithError(ctx context.Context, url string, delay time.Duration, fail bool) error {
	select {
	case <-time.After(delay):
		if fail {
			return fmt.Errorf("%s: service unavailable", url)
		}
		fmt.Printf("  fetched: %s\n", url)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", url, ctx.Err())
	}
}

func demoErrGroupCancel() {
	fmt.Println()
	fmt.Println("=== errgroup: first error cancels group ===")

	g, ctx := WithContext(context.Background())

	type task struct {
		url   string
		delay time.Duration
		fail  bool
	}
	tasks := []task{
		{"/api/a", 10 * time.Millisecond, false},
		{"/api/b", 20 * time.Millisecond, true},  // this will fail
		{"/api/c", 60 * time.Millisecond, false}, // will be cancelled
		{"/api/d", 70 * time.Millisecond, false}, // will be cancelled
	}

	for _, t := range tasks {
		t := t
		g.Go(func() error {
			return fetchWithError(ctx, t.url, t.delay, t.fail)
		})
	}

	fmt.Printf("  group error: %v\n", g.Wait())
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: semaphore-based concurrency limit
//
// Use a buffered channel as a counting semaphore to limit fan-out.
// ─────────────────────────────────────────────────────────────────────────────

func demoSemaphorePool() {
	fmt.Println()
	fmt.Println("=== Semaphore pool (max 3 concurrent of 10) ===")

	sem := make(chan struct{}, 3) // max 3 concurrent
	var wg sync.WaitGroup
	var mu sync.Mutex
	peak := 0
	active := 0

	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{} // acquire
			defer func() { <-sem }() // release

			mu.Lock()
			active++
			if active > peak {
				peak = active
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			active--
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("  peak concurrent: %d  (limit was 3)\n", peak)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 4: scatter-gather — fan-out work, gather results
// ─────────────────────────────────────────────────────────────────────────────

type SearchResult struct {
	Source string
	Items  []string
}

func searchSource(ctx context.Context, source string, query string, delay time.Duration) (SearchResult, error) {
	select {
	case <-time.After(delay):
		return SearchResult{
			Source: source,
			Items:  []string{fmt.Sprintf("%s-result-1", source), fmt.Sprintf("%s-result-2", source)},
		}, nil
	case <-ctx.Done():
		return SearchResult{}, ctx.Err()
	}
}

func demoScatterGather() {
	fmt.Println()
	fmt.Println("=== Scatter-gather: parallel search ===")

	type source struct {
		name  string
		delay time.Duration
	}
	sources := []source{
		{"database", 30 * time.Millisecond},
		{"cache", 10 * time.Millisecond},
		{"elasticsearch", 40 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := make([]SearchResult, len(sources))
	var wg sync.WaitGroup
	var firstErr error
	var once sync.Once

	for i, s := range sources {
		i, s := i, s
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := searchSource(ctx, s.name, "golang", s.delay)
			if err != nil {
				once.Do(func() { firstErr = err })
				return
			}
			results[i] = r
		}()
	}

	wg.Wait()

	if firstErr != nil {
		fmt.Printf("  partial failure: %v\n", firstErr)
	}

	total := 0
	for _, r := range results {
		if r.Source != "" {
			fmt.Printf("  %s: %v\n", r.Source, r.Items)
			total += len(r.Items)
		}
	}
	fmt.Printf("  total results: %d\n", total)
}

func main() {
	demoErrGroup()
	demoErrGroupCancel()
	demoSemaphorePool()
	demoScatterGather()

	_ = errors.New // suppress unused import
}
