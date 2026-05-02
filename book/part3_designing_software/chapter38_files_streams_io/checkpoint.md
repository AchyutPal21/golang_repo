# Chapter 38 — Revision Checkpoint

## Questions

1. Why is `bufio.Writer.Flush()` mandatory and what happens if you skip it?
2. What does `bufio.Scanner.Err()` return and when must you check it?
3. What problem does `io.TeeReader` solve and give one real-world use case.
4. What is the difference between `io.LimitReader` and `io.SectionReader`?
5. Why is temp-file + `os.Rename` safer than writing directly to the final path?

## Answers

1. `bufio.Writer` accumulates writes in a memory buffer and does not call the
   underlying `io.Writer` until the buffer is full or `Flush()` is called. If you
   close the file without flushing, the bytes still in the buffer are silently
   discarded — no error is returned from `Close()` about the missing data. Always
   call `Flush()` before closing. In production code, check the error from `Flush()`
   because the underlying write can fail (disk full, network error) even if earlier
   writes succeeded.

2. `scanner.Err()` returns the first non-EOF error encountered during scanning. After
   the `for scanner.Scan()` loop ends, check `scanner.Err()` to distinguish a clean
   EOF from a read error. If you skip the check, you silently accept partial data
   (e.g., a file that was truncated mid-read) as complete.

3. `io.TeeReader(src, side)` returns a reader that, when read, simultaneously writes
   every byte to `side`. It solves the problem of "I need to process the stream AND
   keep a copy of the raw bytes" without buffering the whole stream first.
   Real-world use case: HTTP middleware that logs request bodies — the handler reads
   the body through a TeeReader, so the raw bytes are captured to a log buffer while
   the handler processes the decoded JSON, with no extra memory allocation for the
   whole body.

4. `io.LimitReader(r, n)` wraps any `io.Reader` and stops after `n` bytes —
   sequential access only, no seeking. Used to cap untrusted input.
   `io.NewSectionReader(rs, offset, size)` wraps an `io.ReaderAt` (which supports
   random access) and exposes a window starting at `offset` of length `size`. Used
   to read a specific region of a file (e.g., a record in a packed binary format)
   without seeking the file pointer.

5. Writing directly to the final path leaves readers in a race with the writer:
   they may see a partially-written file at any moment. If the process crashes
   mid-write, the file is corrupted and the old content is gone. With temp-file +
   `os.Rename`, the writer produces a complete file atomically — `Rename` is a
   single syscall that replaces the old path with the new file in one step on most
   operating systems. Readers either see the old complete file or the new complete
   file, never a partial one. The old file is not deleted until the rename succeeds,
   so a crash during writing leaves the original intact.
