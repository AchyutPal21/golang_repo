// FILE: 08_standard_library/05_io_package.go
// TOPIC: io Package — Reader, Writer, Copy, bufio, pipes
//
// Run: go run 08_standard_library/05_io_package.go

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// custom Writer that counts bytes written
type CountingWriter struct {
	w     io.Writer
	count int64
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.count += int64(n)
	return
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: io Package")
	fmt.Println("════════════════════════════════════════")

	// ── io.Reader and io.Writer — THE most important interfaces ───────────
	// io.Reader: { Read(p []byte) (n int, err error) }
	// io.Writer: { Write(p []byte) (n int, err error) }
	// Everything in Go I/O is built around these two interfaces.
	// Files, HTTP bodies, buffers, gzip streams — all implement Reader/Writer.

	// ── strings.NewReader — string as io.Reader ────────────────────────────
	fmt.Println("\n── strings.NewReader ──")
	r := strings.NewReader("Hello, Go io package!")
	buf := make([]byte, 5)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Printf("  Read %d bytes: %q\n", n, buf[:n])
		}
		if err == io.EOF {
			break
		}
	}

	// ── io.ReadAll — read everything from a Reader ──────────────────────────
	fmt.Println("\n── io.ReadAll ──")
	r2 := strings.NewReader("Read everything at once")
	data, _ := io.ReadAll(r2)
	fmt.Printf("  ReadAll: %q\n", data)

	// ── io.Copy — copy from Reader to Writer ──────────────────────────────
	fmt.Println("\n── io.Copy ──")
	src := strings.NewReader("Copying data efficiently")
	var dst bytes.Buffer
	n, _ := io.Copy(&dst, src)
	fmt.Printf("  Copied %d bytes: %q\n", n, dst.String())

	// ── io.MultiWriter — write to multiple destinations ────────────────────
	fmt.Println("\n── io.MultiWriter ──")
	var buf1, buf2 bytes.Buffer
	mw := io.MultiWriter(&buf1, &buf2)
	fmt.Fprintf(mw, "Written to both buffers")
	fmt.Printf("  buf1: %q\n  buf2: %q\n", buf1.String(), buf2.String())

	// ── io.TeeReader — read AND copy simultaneously ────────────────────────
	fmt.Println("\n── io.TeeReader ──")
	var teeOut bytes.Buffer
	teeReader := io.TeeReader(strings.NewReader("tee'd data"), &teeOut)
	result, _ := io.ReadAll(teeReader)
	fmt.Printf("  Read: %q\n  Tee: %q\n", result, teeOut.String())

	// ── io.LimitReader — limit how much can be read ─────────────────────────
	fmt.Println("\n── io.LimitReader ──")
	limited := io.LimitReader(strings.NewReader("Hello, World!"), 5)
	limited_data, _ := io.ReadAll(limited)
	fmt.Printf("  LimitReader(5): %q\n", limited_data)

	// ── io.Discard — black hole writer ────────────────────────────────────
	// Useful for draining a body you don't need (must drain HTTP responses!)
	n2, _ := io.Copy(io.Discard, strings.NewReader("discarded content"))
	fmt.Printf("\n── io.Discard: discarded %d bytes ──\n", n2)

	// ── Custom io.Writer ─────────────────────────────────────────────────
	fmt.Println("\n── Custom Writer (CountingWriter) ──")
	var underlying bytes.Buffer
	cw := &CountingWriter{w: &underlying}
	fmt.Fprintf(cw, "Hello, %s!", "World")
	fmt.Printf("  Wrote %d bytes: %q\n", cw.count, underlying.String())

	// ── bufio.Scanner — line-by-line reading ──────────────────────────────
	fmt.Println("\n── bufio.Scanner ──")
	input := "line one\nline two\nline three"
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		fmt.Printf("  Line: %q\n", scanner.Text())
	}

	// ── bufio.Writer — buffered writing ───────────────────────────────────
	fmt.Println("\n── bufio.Writer ──")
	var rawBuf bytes.Buffer
	bw := bufio.NewWriter(&rawBuf)
	fmt.Fprint(bw, "buffered ")
	fmt.Fprint(bw, "output")
	// Data is in bufio's internal buffer — NOT yet in rawBuf:
	fmt.Printf("  Before Flush: rawBuf=%q\n", rawBuf.String())
	bw.Flush()  // MUST flush to write buffered data to underlying writer
	fmt.Printf("  After Flush:  rawBuf=%q\n", rawBuf.String())

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  io.Reader / io.Writer — the universal I/O interfaces")
	fmt.Println("  io.Copy(dst, src)   — efficient transfer, uses 32KB buffer")
	fmt.Println("  io.ReadAll(r)       — read everything (watch memory for large data)")
	fmt.Println("  io.MultiWriter      — fan-out to multiple writers")
	fmt.Println("  io.TeeReader        — read + copy simultaneously")
	fmt.Println("  io.LimitReader      — prevent reading too much")
	fmt.Println("  io.Discard          — drain reader (HTTP body must be drained!)")
	fmt.Println("  bufio.Scanner       — line/word scanning with custom split functions")
	fmt.Println("  bufio.Writer        — buffer writes, always Flush() at end")
}
