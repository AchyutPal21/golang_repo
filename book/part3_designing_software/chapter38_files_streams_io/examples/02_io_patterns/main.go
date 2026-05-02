// FILE: book/part3_designing_software/chapter38_files_streams_io/examples/02_io_patterns/main.go
// CHAPTER: 38 — Files, Streams, and Buffered I/O
// TOPIC: io.TeeReader, io.MultiWriter, io.LimitReader, io.SectionReader,
//        io.Pipe, and composing stream transformations.
//
// Run (from the chapter folder):
//   go run ./examples/02_io_patterns

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// io.TeeReader — read from src AND write to a side writer simultaneously
// ─────────────────────────────────────────────────────────────────────────────

func demoTee() {
	fmt.Println("=== io.TeeReader ===")

	src := strings.NewReader("the quick brown fox jumps over the lazy dog")
	var capture strings.Builder

	// tee reads from src; every byte read is also written to capture.
	tee := io.TeeReader(src, &capture)

	// Count words as we read — the bytes also land in capture.
	scanner := bufio.NewScanner(tee)
	scanner.Split(bufio.ScanWords)
	wordCount := 0
	for scanner.Scan() {
		wordCount++
	}

	fmt.Printf("  words: %d\n", wordCount)
	fmt.Printf("  captured: %q\n", capture.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// io.MultiWriter — fan-out: write to multiple destinations simultaneously
// ─────────────────────────────────────────────────────────────────────────────

func demoMultiWriter() {
	fmt.Println()
	fmt.Println("=== io.MultiWriter ===")

	var primary strings.Builder
	var secondary strings.Builder
	var audit strings.Builder

	mw := io.MultiWriter(&primary, &secondary, &audit)

	lines := []string{"event: user.login user=alice", "event: page.view path=/dashboard", "event: user.logout user=alice"}
	for _, line := range lines {
		_, _ = fmt.Fprintln(mw, line)
	}

	fmt.Printf("  primary   (%d bytes): %q...\n", primary.Len(), primary.String()[:20])
	fmt.Printf("  secondary (%d bytes): same content\n", secondary.Len())
	fmt.Printf("  audit     (%d bytes): same content\n", audit.Len())
	fmt.Printf("  all equal: %v\n", primary.String() == secondary.String() && secondary.String() == audit.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// io.LimitReader — prevent reading more than N bytes
// ─────────────────────────────────────────────────────────────────────────────

func demoLimitReader() {
	fmt.Println()
	fmt.Println("=== io.LimitReader ===")

	big := strings.NewReader(strings.Repeat("ABCDE", 20)) // 100 bytes
	limited := io.LimitReader(big, 25)

	data, _ := io.ReadAll(limited)
	fmt.Printf("  source: 100 bytes  limit: 25  read: %d bytes  content: %q\n",
		len(data), string(data))
}

// ─────────────────────────────────────────────────────────────────────────────
// io.SectionReader — random access within a reader
// ─────────────────────────────────────────────────────────────────────────────

func demoSectionReader() {
	fmt.Println()
	fmt.Println("=== io.SectionReader ===")

	// Simulate a file with three records packed together.
	content := "HEADER--RECORD1-RECORD2-RECORD3-FOOTER--"
	rs := strings.NewReader(content)

	// Read only record 1 (bytes 8–15) and record 2 (bytes 16–23).
	for _, region := range []struct {
		name   string
		offset int64
		size   int64
	}{
		{"RECORD1", 8, 7},
		{"RECORD2", 16, 7},
		{"RECORD3", 24, 7},
	} {
		section := io.NewSectionReader(rs, region.offset, region.size)
		data, _ := io.ReadAll(section)
		fmt.Printf("  %s at offset %d: %q\n", region.name, region.offset, string(data))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// io.Pipe — connect a writer to a reader without buffering in memory
// ─────────────────────────────────────────────────────────────────────────────

func demoIoPipe() {
	fmt.Println()
	fmt.Println("=== io.Pipe ===")

	pr, pw := io.Pipe()

	// Producer goroutine — writes to pw.
	go func() {
		defer pw.Close()
		for i := 1; i <= 5; i++ {
			_, _ = fmt.Fprintf(pw, "chunk-%d ", i)
		}
	}()

	// Consumer — reads from pr (blocks until producer writes).
	data, _ := io.ReadAll(pr)
	fmt.Printf("  received: %q\n", strings.TrimSpace(string(data)))
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPOSING TRANSFORMATIONS
//
// Stack io.Reader wrappers to transform a stream without loading it into memory.
// ─────────────────────────────────────────────────────────────────────────────

// uppercaseReader wraps an io.Reader and uppercases every byte on the fly.
type uppercaseReader struct{ inner io.Reader }

func (u *uppercaseReader) Read(p []byte) (int, error) {
	n, err := u.inner.Read(p)
	for i := 0; i < n; i++ {
		if p[i] >= 'a' && p[i] <= 'z' {
			p[i] -= 32
		}
	}
	return n, err
}

// countingReader wraps an io.Reader and counts total bytes read.
type countingReader struct {
	inner io.Reader
	n     int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	c.n += int64(n)
	return n, err
}

func demoComposition() {
	fmt.Println()
	fmt.Println("=== Composed stream transformations ===")

	src := strings.NewReader("hello from the composed stream pipeline")

	// Stack: src → uppercase → count bytes
	counter := &countingReader{inner: &uppercaseReader{inner: src}}

	// Consume through bufio.Scanner word by word.
	scanner := bufio.NewScanner(counter)
	scanner.Split(bufio.ScanWords)
	var words []string
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	fmt.Printf("  words: %v\n", words)
	fmt.Printf("  bytes read: %d\n", counter.n)
}

func main() {
	demoTee()
	demoMultiWriter()
	demoLimitReader()
	demoSectionReader()
	demoIoPipe()
	demoComposition()
}
