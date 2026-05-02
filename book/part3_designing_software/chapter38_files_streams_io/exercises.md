# Chapter 38 — Exercises

## 38.1 — Streaming log processor

Run [`exercises/01_log_processor`](exercises/01_log_processor/main.go).

`LogProcessor` filters a log stream by minimum level and writes to any `io.Writer`.

Try:
- Add a `RateLimitedWriter` that allows at most N bytes per call to `Write`, discarding the excess. Verify the log processor handles short writes correctly.
- Add a `TimestampedWriter` that prepends `[HH:MM:SS]` to each line. Wrap the output writer with it: `NewLogProcessor(INFO).Process(src, NewTimestampedWriter(dst))`.
- Count lines in a file using `countLines(r io.Reader)` without reading the file into memory. Test with a 1000-line synthetic input from `strings.NewReader`.

## 38.2 ★ — Word frequency from stream

Build a `WordCounter` that reads from any `io.Reader` and returns `map[string]int` of word frequencies. Use `bufio.Scanner` with `ScanWords`. Test on a `strings.NewReader` and on a `bytes.Buffer`.

## 38.3 ★★ — Chunked upload simulator

Implement a `ChunkedWriter` that splits a stream into fixed-size chunks and calls an `Upload(chunk []byte, part int) error` function for each. After all chunks, call `Finalize() error`. Simulate upload failure on part 3 and verify no partial data is committed.
