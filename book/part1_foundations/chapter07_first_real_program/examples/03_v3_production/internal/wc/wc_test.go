// FILE: examples/03_v3_production/internal/wc/wc_test.go
// TOPIC: Table-driven tests for the pure Count function.
//
// Run (from the chapter folder):
//   go test ./examples/03_v3_production/internal/wc
//   go test -v ./examples/03_v3_production/internal/wc
//   go test -run TestCount/utf8 ./examples/03_v3_production/internal/wc

package wc

import (
	"strings"
	"testing"
)

// TestCount exercises the Count function across a representative spread of
// inputs. Table-driven tests are the idiomatic Go pattern: one test
// function loops over a list of cases. Each case becomes a subtest via
// t.Run, so a failure shows the case name in the output.
func TestCount(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  Stats
	}{
		{
			// The empty string has no newlines; Scanner produces no tokens.
			name:  "empty",
			input: "",
			want:  Stats{},
		},
		{
			// "hello\n" → one line, one word, six bytes (h-e-l-l-o-\n).
			// The scanner strips the \n, but Count adds 1 back per line.
			name:  "one line",
			input: "hello\n",
			want:  Stats{Lines: 1, Words: 1, Bytes: 6, Runes: 6},
		},
		{
			name:  "two lines",
			input: "hello world\nfoo bar\n",
			want:  Stats{Lines: 2, Words: 4, Bytes: 20, Runes: 20},
		},
		{
			// UTF-8: "héllo" is 6 bytes (h-é=2-l-l-o) but 5 runes, plus the
			// newline gives 7 bytes / 6 runes total.
			name:  "utf8",
			input: "héllo\n",
			want:  Stats{Lines: 1, Words: 1, Bytes: 7, Runes: 6},
		},
		{
			// Whitespace-only line still counts as one line, no words.
			name:  "whitespace line",
			input: "   \n",
			want:  Stats{Lines: 1, Words: 0, Bytes: 4, Runes: 4},
		},
		{
			// Tabs are word separators just like spaces.
			name:  "tabs separate words",
			input: "one\ttwo\tthree\n",
			want:  Stats{Lines: 1, Words: 3, Bytes: 14, Runes: 14},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Count(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Count(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("Count(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
		})
	}
}
