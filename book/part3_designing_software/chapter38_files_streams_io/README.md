# Chapter 38 — Files, Streams, and Buffered I/O

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Go's I/O model is built on two interfaces — `io.Reader` and `io.Writer` — that everything else composes. Understanding buffering, streaming, and the standard composing utilities (`TeeReader`, `MultiWriter`, `Pipe`) lets you write memory-efficient programs that process arbitrarily large data.

---

## 38.1 — File operations

| Operation | Function |
|---|---|
| Read entire file | `os.ReadFile(path)` |
| Write entire file | `os.WriteFile(path, data, perm)` |
| Open for reading | `os.Open(path)` |
| Open/create for writing | `os.Create(path)` |
| Append | `os.OpenFile(path, os.O_APPEND\|os.O_CREATE\|os.O_WRONLY, 0644)` |

Always `defer f.Close()` immediately after a successful `Open` or `Create`.

---

## 38.2 — Buffering

Without a buffer, each `fmt.Fprintf(f, ...)` call is a syscall. `bufio.Writer` accumulates bytes in memory and flushes in large chunks:

```go
w := bufio.NewWriter(f)
defer w.Flush() // or call Flush() explicitly before Close
```

**`Flush()` is mandatory** — bytes still in the buffer are lost if you close the file without flushing.

`bufio.Scanner` reads line by line without loading the whole file:

```go
scanner := bufio.NewScanner(f)
for scanner.Scan() {
    process(scanner.Text())
}
if err := scanner.Err(); err != nil { /* handle */ }
```

---

## 38.3 — io.TeeReader

Reads from `src` and simultaneously copies every byte to a side `io.Writer`:

```go
tee := io.TeeReader(src, &sideCapture)
// Reading from tee also writes to sideCapture
```

Used for: checksumming while transferring, logging raw bytes while parsing, capturing request bodies in middleware.

---

## 38.4 — io.MultiWriter

Fans a single write out to multiple `io.Writer` destinations:

```go
mw := io.MultiWriter(file, &logBuffer, os.Stdout)
fmt.Fprintln(mw, "written to all three")
```

---

## 38.5 — io.LimitReader and io.SectionReader

`io.LimitReader(r, n)` — reads at most `n` bytes. Essential for handling untrusted input.

`io.NewSectionReader(rs, offset, size)` — random access into an `io.ReaderAt`. Read any region of a file without seeking.

---

## 38.6 — io.Pipe

Connects a writer to a reader without buffering through memory:

```go
pr, pw := io.Pipe()
go func() { defer pw.Close(); /* write to pw */ }()
data, _ := io.ReadAll(pr) // reads what the goroutine wrote
```

---

## 38.7 — Composing Reader/Writer wrappers

Wrap readers and writers to transform streams lazily:

```go
counter := &countingReader{inner: &uppercaseReader{inner: src}}
// Every byte read is counted AND uppercased as it passes through
```

The transformation happens as bytes flow — no intermediate copy of the whole stream.

---

## 38.8 — Atomic file writes

Write to a temp file, then `os.Rename` to the final path. Rename is atomic — readers never see a partial file:

```go
tmp, _ := os.CreateTemp("", "atomic-*")
tmp.WriteString(content)
tmp.Close()
os.Rename(tmp.Name(), finalPath)
```

---

## Running the examples

```bash
cd book/part3_designing_software/chapter38_files_streams_io

go run ./examples/01_files_buffered  # os.ReadFile, bufio.Scanner, bufio.Writer, atomic write
go run ./examples/02_io_patterns     # TeeReader, MultiWriter, LimitReader, SectionReader, Pipe, composition

go run ./exercises/01_log_processor  # streaming log filter pipeline with multi-sink
```

---

## Key takeaways

1. **`defer f.Close()`** immediately after opening — ensures cleanup on all return paths.
2. **`bufio.Writer.Flush()`** is mandatory — bytes in the buffer are lost without it.
3. **`bufio.Scanner`** is the idiomatic line-by-line reader — memory efficient for large files.
4. **`io.TeeReader`** — read and capture simultaneously.
5. **`io.MultiWriter`** — fan one write to multiple destinations.
6. **Compose readers/writers** to transform streams without loading everything into memory.
7. **Atomic write** = temp file + `os.Rename`.

---

## Cross-references

- **Chapter 39** — Encoding: `encoding/json` reads and writes `io.Reader`/`io.Writer`
- **Chapter 32** — Structural Patterns: Decorator applied to `io.Reader`/`io.Writer`
