// EXERCISE 22.1 — io.Reader/Writer pipeline.
//
// Build a pipeline of io.Reader wrappers:
//   - limitedReader: reads at most N bytes total
//   - rot13Reader: ROT-13 encodes letter bytes
// Chain them: rot13(limited(source)) and read the result.
//
// Run (from the chapter folder):
//   go run ./exercises/01_io_pipeline

package main

import (
	"fmt"
	"io"
	"strings"
)

// limitedReader reads at most limit bytes from r.
type limitedReader struct {
	r     io.Reader
	limit int64
	read  int64
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.read >= l.limit {
		return 0, io.EOF
	}
	remaining := l.limit - l.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err = l.r.Read(p)
	l.read += int64(n)
	return
}

// rot13Reader applies ROT-13 to alphabetic bytes.
type rot13Reader struct{ r io.Reader }

func rot13(b byte) byte {
	switch {
	case b >= 'a' && b <= 'z':
		return 'a' + (b-'a'+13)%26
	case b >= 'A' && b <= 'Z':
		return 'A' + (b-'A'+13)%26
	}
	return b
}

func (r *rot13Reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	for i := range p[:n] {
		p[i] = rot13(p[i])
	}
	return
}

func readAll(r io.Reader) string {
	var sb strings.Builder
	buf := make([]byte, 32)
	for {
		n, err := r.Read(buf)
		sb.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
	}
	return sb.String()
}

func main() {
	text := "Hello, World! Attack at dawn."

	// Just limited
	lr := &limitedReader{r: strings.NewReader(text), limit: 13}
	fmt.Println("limited(13):", readAll(lr))

	// Just rot13
	rr := &rot13Reader{r: strings.NewReader(text)}
	encoded := readAll(rr)
	fmt.Println("rot13:      ", encoded)

	// Verify ROT-13 is its own inverse
	rr2 := &rot13Reader{r: strings.NewReader(encoded)}
	fmt.Println("rot13 again:", readAll(rr2))

	// Pipeline: rot13 of limited
	pipeline := &rot13Reader{r: &limitedReader{
		r:     strings.NewReader(text),
		limit: 20,
	}}
	fmt.Println("rot13(limit(20)):", readAll(pipeline))
}
