# Chapter 49 — Pipelines, Fan-In/Out

## What you will learn

- The pipeline pattern: composing goroutines as stages connected by channels
- Writing each stage as a function that takes `<-chan T` and returns `<-chan T`
- Context-aware cancellation that propagates cleanly through a chain of stages
- Back-pressure: how buffered channels let slow consumers throttle fast producers
- Fan-out: one input channel consumed by N concurrent workers
- Fan-in (merge): N output channels combined into one using a WaitGroup
- Ordered fan-out: preserving input order by writing results into an indexed slice
- Practical multi-stage pipeline: source → parse → validate → enrich → aggregate

---

## Pipeline stage signature

Each stage follows the same contract:

```go
func stageName(ctx context.Context, in <-chan T) <-chan U {
    out := make(chan U)
    go func() {
        defer close(out)         // always close downstream
        for {
            select {
            case v, ok := <-in:
                if !ok { return } // upstream closed — we're done
                result := transform(v)
                select {
                case out <- result:
                case <-ctx.Done():
                    return
                }
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}
```

Stages are composed by passing one stage's output as the next stage's input:

```go
parsed   := parseStage(ctx, sourceStage(ctx))
validated := validateStage(ctx, parsed)
enriched  := enrichStage(ctx, validated)
```

---

## Fan-out

Multiple goroutines drain the same input channel. Each item is processed by exactly one worker:

```go
jobs := make(chan Job, buf)
var wg sync.WaitGroup
for range n {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for job := range jobs { process(job) }
    }()
}
```

When `jobs` is closed, all workers exit. This is the same worker pool from Ch48, but expressed inline.

---

## Fan-in (merge)

Combine N channels into one, closing the output when all inputs are exhausted:

```go
func merge(ctx context.Context, channels ...<-chan T) <-chan T {
    out := make(chan T)
    var wg sync.WaitGroup
    wg.Add(len(channels))
    for _, ch := range channels {
        go func(c <-chan T) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(ch)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

The output order is non-deterministic — whichever worker finishes first sends first.

---

## Ordered fan-out

To preserve input order, attach an index to each item and write results into a pre-allocated slice:

```go
type indexed struct{ i int; v int }
results := make([]int, len(inputs))
out := make(chan indexed, len(inputs))

for i, v := range inputs {
    go func(i, v int) { out <- indexed{i, compute(v)} }(i, v)
}
for range inputs {
    r := <-out
    results[r.i] = r.v
}
```

Because each goroutine writes to a distinct index, no mutex is needed.

---

## Back-pressure

A buffered channel between stages acts as a bounded queue. If the downstream stage is slow, the upstream stage blocks when the buffer is full — this is back-pressure:

```
producer → [buffer:2] → slow consumer
```

The producer cannot outrun the consumer by more than `cap(buffer)` items. This prevents unbounded memory growth without dropping data.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_pipeline_stages/main.go` | Generator, square, filter, take, back-pressure |
| `examples/02_fan_in_out/main.go` | Fan-out + merge, ordered fan-out, parallel fetch, multi-stage |

## Exercise

`exercises/01_csv_pipeline/main.go` — five-stage pipeline: source → parse → validate → enrich (fan-out, 4 workers) → aggregate.
