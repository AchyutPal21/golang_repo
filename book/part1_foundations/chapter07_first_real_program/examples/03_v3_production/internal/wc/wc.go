// Package wc is the pure logic of the word counter. No I/O, no flag
// parsing, no exit codes — only "given an io.Reader, compute Stats." This
// is the unit-test surface: given known input bytes, expect known counts.
//
// The package is under internal/ so external modules cannot import it; the
// only consumer is cmd/wc in this same module.
package wc

import (
	"bufio"
	"io"
	"unicode/utf8"
)

// Stats is the four-count summary of a stream of text. The fields are
// exported so callers can read them and so we can compose Stats values
// across multiple files (e.g. computing a total).
type Stats struct {
	Lines int64
	Words int64
	Bytes int64
	Runes int64
}

// Count reads r to EOF and returns the Lines/Words/Bytes/Runes counts. It
// streams the input via bufio.Scanner with a 1 MiB max line size, which
// is enough for any realistic text. Returns the underlying reader's error
// if Scan terminates abnormally.
func Count(r io.Reader) (Stats, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var s Stats
	for sc.Scan() {
		line := sc.Bytes()
		s.Lines++
		s.Bytes += int64(len(line)) + 1 // +1 for the stripped newline
		s.Runes += int64(utf8.RuneCount(line)) + 1

		inWord := false
		for _, b := range line {
			if b == ' ' || b == '\t' {
				inWord = false
			} else if !inWord {
				inWord = true
				s.Words++
			}
		}
	}
	return s, sc.Err()
}
