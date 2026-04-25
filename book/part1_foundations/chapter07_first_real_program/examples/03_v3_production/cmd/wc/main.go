// FILE: examples/03_v3_production/cmd/wc/main.go
// CHAPTER: 07 — Your First Real Program
// TOPIC: Production layout: thin main, all logic in internal/wc.
//
// Run (from the chapter folder):
//   go run ./examples/03_v3_production/cmd/wc -l -w README.md
//   go run ./examples/03_v3_production/cmd/wc --version
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   This is the template every Go CLI you write will follow:
//     1. main() returns nothing; it calls run() and exits with the right code.
//     2. run() parses flags, decides what to do, dispatches into internal/wc.
//     3. The actual counting logic lives in internal/wc.Count, where it can
//        be tested without invoking the CLI.
//   30 lines here; the work is in internal/wc.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/upskill-go/book/part1_foundations/chapter07_first_real_program/examples/03_v3_production/internal/wc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run is the real entry point. We separate it from main() so we can return
// errors explicitly and so tests could exercise it without spawning a
// subprocess. (We don't write that test here, but the structure permits it.)
func run() error {
	var (
		showLines, showWords, showBytes, showRunes bool
		showVersion                                bool
	)
	flag.BoolVar(&showLines, "l", false, "show line count")
	flag.BoolVar(&showWords, "w", false, "show word count")
	flag.BoolVar(&showBytes, "c", false, "show byte count")
	flag.BoolVar(&showRunes, "m", false, "show rune count")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: wc [-lwcm] [--version] [file ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		printVersion()
		return nil
	}

	if !(showLines || showWords || showBytes || showRunes) {
		// Default to lines, words, bytes — matches GNU wc.
		showLines, showWords, showBytes = true, true, true
	}
	display := wc.Display{Lines: showLines, Words: showWords, Bytes: showBytes, Runes: showRunes}

	files := flag.Args()
	if len(files) == 0 {
		// Stdin mode.
		s, err := wc.Count(os.Stdin)
		if err != nil {
			return fmt.Errorf("stdin: %w", err)
		}
		fmt.Println(wc.Format(s, "", display))
		return nil
	}

	var total wc.Stats
	var firstErr error
	for _, name := range files {
		s, err := countFile(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		total.Lines += s.Lines
		total.Words += s.Words
		total.Bytes += s.Bytes
		total.Runes += s.Runes
		fmt.Println(wc.Format(s, name, display))
	}

	if len(files) > 1 {
		fmt.Println(wc.Format(total, "total", display))
	}
	return firstErr
}

func countFile(name string) (wc.Stats, error) {
	f, err := os.Open(name)
	if err != nil {
		return wc.Stats{}, err
	}
	defer f.Close()

	s, err := wc.Count(io.Reader(f))
	if err != nil {
		return wc.Stats{}, fmt.Errorf("%s: %w", name, err)
	}
	return s, nil
}
