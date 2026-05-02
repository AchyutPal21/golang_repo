// FILE: book/part3_designing_software/chapter38_files_streams_io/examples/01_files_buffered/main.go
// CHAPTER: 38 — Files, Streams, and Buffered I/O
// TOPIC: os.File, bufio.Scanner, bufio.Writer, io.Reader, io.Writer,
//        defer-close pattern, and why buffering matters.
//
// Run (from the chapter folder):
//   go run ./examples/01_files_buffered

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// READING FILES
// ─────────────────────────────────────────────────────────────────────────────

// readAll reads the full content of a file into memory.
// Good for small files; use streaming for large ones.
func readAll(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("readAll: %w", err)
	}
	return string(data), nil
}

// readLines reads a file line by line using bufio.Scanner.
// Memory-efficient for large files — never loads the whole file.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("readLines: %w", err)
	}
	defer f.Close() // always close; defer ensures cleanup on any return path

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("readLines: scan: %w", err)
	}
	return lines, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// WRITING FILES
// ─────────────────────────────────────────────────────────────────────────────

// writeAll atomically writes content to a file.
func writeAll(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writeAll: %w", err)
	}
	return nil
}

// writeBuffered writes many small pieces efficiently using bufio.Writer.
// Without buffering, each Fprintf call is a syscall — very slow.
func writeBuffered(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writeBuffered: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("writeBuffered: write: %w", err)
		}
	}
	// Flush MUST be called — bytes in the buffer are not written otherwise.
	if err := w.Flush(); err != nil {
		return fmt.Errorf("writeBuffered: flush: %w", err)
	}
	return nil
}

// appendLine opens a file in append mode.
func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("appendLine: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// io.Reader / io.Writer — the universal stream interfaces
// ─────────────────────────────────────────────────────────────────────────────

// copyWithProgress copies from src to dst, printing progress every 20 bytes.
func copyWithProgress(dst io.Writer, src io.Reader, label string) (int64, error) {
	buf := make([]byte, 20)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
			fmt.Printf("  [%s] copied %d bytes (total=%d)\n", label, n, total)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// countLines counts lines in any io.Reader without loading it into memory.
func countLines(r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// TEMP FILES — write to temp, rename to final (atomic replace)
// ─────────────────────────────────────────────────────────────────────────────

func atomicWrite(finalPath, content string) error {
	// Write to a temp file in the same directory.
	tmp, err := os.CreateTemp("", "atomic-*")
	if err != nil {
		return fmt.Errorf("atomicWrite: create temp: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp on failure.
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicWrite: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomicWrite: close: %w", err)
	}

	// Rename is atomic on most OSes — readers never see a partial file.
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("atomicWrite: rename: %w", err)
	}
	success = true
	return nil
}

func main() {
	// Set up a temp directory.
	dir, err := os.MkdirTemp("", "ch38-*")
	if err != nil {
		fmt.Println("setup error:", err)
		return
	}
	defer os.RemoveAll(dir)

	samplePath := dir + "/sample.txt"
	sample := "line one\nline two\nline three\nline four\nline five\n"

	fmt.Println("=== Write and read a file ===")
	_ = writeAll(samplePath, sample)
	content, err := readAll(samplePath)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("  read %d bytes\n", len(content))
	}

	fmt.Println()
	fmt.Println("=== Buffered line-by-line read ===")
	lines, _ := readLines(samplePath)
	for i, l := range lines {
		fmt.Printf("  [%d] %q\n", i+1, l)
	}

	fmt.Println()
	fmt.Println("=== Buffered write ===")
	buffPath := dir + "/buffered.txt"
	words := []string{"apple", "banana", "cherry", "date", "elderberry"}
	_ = writeBuffered(buffPath, words)
	written, _ := readAll(buffPath)
	fmt.Printf("  wrote %d bytes: %q\n", len(written), strings.TrimSpace(written))

	fmt.Println()
	fmt.Println("=== Append mode ===")
	logPath := dir + "/app.log"
	for _, msg := range []string{"server started", "user login", "request processed"} {
		_ = appendLine(logPath, msg)
	}
	logLines, _ := readLines(logPath)
	fmt.Printf("  log has %d lines\n", len(logLines))

	fmt.Println()
	fmt.Println("=== io.Reader / io.Writer pipeline ===")
	src := strings.NewReader("Hello, io.Reader and io.Writer world!")
	var dst strings.Builder
	n, _ := copyWithProgress(&dst, src, "copy")
	fmt.Printf("  total bytes copied: %d\n", n)
	fmt.Printf("  result: %q\n", dst.String())

	fmt.Println()
	fmt.Println("=== countLines on any Reader ===")
	multiLine := strings.NewReader("one\ntwo\nthree\nfour\nfive\n")
	count, _ := countLines(multiLine)
	fmt.Printf("  line count: %d\n", count)

	fmt.Println()
	fmt.Println("=== Atomic write ===")
	finalPath := dir + "/config.json"
	_ = atomicWrite(finalPath, `{"version":"1.0","debug":false}`)
	cfg, _ := readAll(finalPath)
	fmt.Printf("  config: %s\n", cfg)
}
