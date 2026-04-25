// FILE: book/part1_foundations/chapter07_first_real_program/examples/02_v2_flags/main.go
// CHAPTER: 07 — Your First Real Program
// TOPIC: `wc` with flags, multi-file support, and stdin fallback.
//
// Run (from the chapter folder):
//   go run ./examples/02_v2_flags -l -w README.md
//   go run ./examples/02_v2_flags -l -w main.go README.md
//   echo "hello world" | go run ./examples/02_v2_flags -w
//   go run ./examples/02_v2_flags -h
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The same logic as v1, plus three production-grade ergonomics:
//     1. Flag-controlled output (-l, -w, -c, -m).
//     2. Multiple file arguments. Each gets its own line; a "total" is
//        appended when N > 1 (matches GNU wc).
//     3. Reads from stdin when no file is given. Lets you pipe data in.
//
//   The logic is still inline; v3 splits it into a package.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// stats holds the four counts wc tracks.
type stats struct {
	lines, words, bytes, runes int64
}

func (s *stats) addLine(line []byte) {
	s.lines++
	s.bytes += int64(len(line)) + 1
	s.runes += int64(utf8.RuneCount(line)) + 1

	inWord := false
	for _, b := range line {
		if b == ' ' || b == '\t' {
			inWord = false
		} else if !inWord {
			inWord = true
			s.words++
		}
	}
}

func count(r io.Reader) (stats, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var s stats
	for sc.Scan() {
		s.addLine(sc.Bytes())
	}
	return s, sc.Err()
}

func main() {
	// ─── Flags ─────────────────────────────────────────────────────────
	var showLines, showWords, showBytes, showRunes bool
	flag.BoolVar(&showLines, "l", false, "show line count")
	flag.BoolVar(&showWords, "w", false, "show word count")
	flag.BoolVar(&showBytes, "c", false, "show byte count")
	flag.BoolVar(&showRunes, "m", false, "show rune count")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: v2 [-lwcm] [file ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// If the user specified no flags, default to -l -w -c (matches GNU wc).
	if !(showLines || showWords || showBytes || showRunes) {
		showLines, showWords, showBytes = true, true, true
	}

	// ─── Inputs ────────────────────────────────────────────────────────
	files := flag.Args()
	var (
		total stats
		anyFails bool
	)

	if len(files) == 0 {
		// No file args → read stdin.
		s, err := count(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "stdin:", err)
			os.Exit(1)
		}
		printStats(s, "", showLines, showWords, showBytes, showRunes)
		return
	}

	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			anyFails = true
			continue
		}
		s, err := count(f)
		_ = f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, name+":", err)
			anyFails = true
			continue
		}
		total.lines += s.lines
		total.words += s.words
		total.bytes += s.bytes
		total.runes += s.runes
		printStats(s, name, showLines, showWords, showBytes, showRunes)
	}

	if len(files) > 1 {
		printStats(total, "total", showLines, showWords, showBytes, showRunes)
	}
	if anyFails {
		os.Exit(1)
	}
}

func printStats(s stats, label string, l, w, c, m bool) {
	first := true
	out := func(n int64) {
		if !first {
			fmt.Print(" ")
		}
		fmt.Printf("%7d", n)
		first = false
	}
	if l {
		out(s.lines)
	}
	if w {
		out(s.words)
	}
	if c {
		out(s.bytes)
	}
	if m {
		out(s.runes)
	}
	if label != "" {
		fmt.Printf(" %s", label)
	}
	fmt.Println()
}
