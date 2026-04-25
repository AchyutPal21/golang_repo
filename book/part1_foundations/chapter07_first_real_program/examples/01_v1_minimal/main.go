// FILE: book/part1_foundations/chapter07_first_real_program/examples/01_v1_minimal/main.go
// CHAPTER: 07 — Your First Real Program
// TOPIC: A minimal `wc` clone — single file, no flags, no stdin.
//
// Run (from the chapter folder):
//   go run ./examples/01_v1_minimal /etc/passwd
//   go run ./examples/01_v1_minimal ../../../README.md
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The smallest possible useful `wc`. Reads one file. Counts lines, words,
//   bytes. Prints. That's the whole spec. ~30 lines of code, plus comments.
//   Use it as the baseline against which v2 and v3 are improvements.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// ─── Argument check ────────────────────────────────────────────────
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: v1 <file>")
		os.Exit(2) // 2 = usage error, by convention (matches grep, wc)
	}
	name := os.Args[1]

	// ─── Open the file ─────────────────────────────────────────────────
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // 1 = runtime error
	}
	defer f.Close()

	// ─── Scan line by line ─────────────────────────────────────────────
	//
	// bufio.Scanner streams the file in chunks; the default split function
	// is ScanLines, which strips the trailing newline. We extend the
	// internal buffer to 1 MiB to handle long lines without panicking.
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var lines, words, bytes int
	for sc.Scan() {
		line := sc.Bytes()
		lines++
		bytes += len(line) + 1 // +1 because ScanLines stripped the \n

		// A "word" is a maximal run of non-whitespace characters. We
		// detect transitions from whitespace to non-whitespace.
		inWord := false
		for _, b := range line {
			if b == ' ' || b == '\t' {
				inWord = false
			} else if !inWord {
				inWord = true
				words++
			}
		}
	}
	if err := sc.Err(); err != nil {
		// Scanner errors are silent unless we check Err(). Always check.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// ─── Output ────────────────────────────────────────────────────────
	//
	// GNU wc format: lines, words, bytes, name. Tab-separated would be
	// strictly correct; spaces are easier to read.
	fmt.Printf("%d %d %d %s\n", lines, words, bytes, name)
}
