// FILE: examples/03_v3_production/internal/wc/format.go
// TOPIC: Output formatting, separated from counting.

package wc

import (
	"fmt"
	"strings"
)

// Display selects which counts to render. The flag-parsing logic in cmd/wc
// builds one of these from the user's flags; the formatter then renders.
type Display struct {
	Lines bool
	Words bool
	Bytes bool
	Runes bool
}

// Format renders Stats as a single line of output, similar to GNU wc.
// label is appended after the counts (filename, "total", or "" for stdin).
func Format(s Stats, label string, d Display) string {
	var parts []string
	if d.Lines {
		parts = append(parts, fmt.Sprintf("%7d", s.Lines))
	}
	if d.Words {
		parts = append(parts, fmt.Sprintf("%7d", s.Words))
	}
	if d.Bytes {
		parts = append(parts, fmt.Sprintf("%7d", s.Bytes))
	}
	if d.Runes {
		parts = append(parts, fmt.Sprintf("%7d", s.Runes))
	}

	out := strings.Join(parts, " ")
	if label != "" {
		out += " " + label
	}
	return out
}
