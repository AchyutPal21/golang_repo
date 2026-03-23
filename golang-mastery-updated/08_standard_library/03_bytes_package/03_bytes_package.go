// 03_bytes_package.go
//
// The bytes package: working with mutable byte slices efficiently.
//
// WHY bytes EXISTS (and why it mirrors strings):
// Go has two ways to represent text:
//   string  — immutable, can be used as map keys, safe to share across goroutines
//   []byte  — mutable, can be written into, ideal for I/O and protocol parsing
//
// Most I/O in Go (os.File, net.Conn, http body) works with []byte.
// The bytes package gives you the same rich operations as strings but for []byte.
//
// THE CORE DECISION:
//   Use strings when you have text that won't change (keys, labels, config values).
//   Use []byte when you're building or parsing data that will be read/written to I/O.
//
// ALLOCATION COST of string ↔ []byte conversion:
//   []byte(s) — always copies; O(n)
//   string(b) — always copies; O(n)  (with one special compiler optimization, see below)

package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: bytes vs strings — mirror functions
// ─────────────────────────────────────────────────────────────────────────────

func mirrorFunctionsDemo() {
	fmt.Println("═══ SECTION 1: bytes mirrors strings ═══")

	// Almost every function in strings has a counterpart in bytes.
	// The difference: strings take/return string; bytes take/return []byte.

	b := []byte("Hello, Gophers! Go is great.")

	// Searching
	fmt.Println(bytes.Contains(b, []byte("Go")))       // true
	fmt.Println(bytes.HasPrefix(b, []byte("Hello")))   // true
	fmt.Println(bytes.HasSuffix(b, []byte("great.")))  // true
	fmt.Println(bytes.Count(b, []byte("o")))            // 3
	fmt.Println(bytes.Index(b, []byte("Go")))           // 7

	// Splitting
	parts := bytes.Split(b, []byte(" "))
	fmt.Printf("Split into %d parts\n", len(parts))

	// Fields (split on whitespace)
	fields := bytes.Fields([]byte("  hello   world  "))
	for _, f := range fields {
		fmt.Printf("  field: %q\n", f)
	}

	// Trimming
	padded := []byte("   hello   ")
	fmt.Printf("TrimSpace: %q\n", bytes.TrimSpace(padded))
	fmt.Printf("Trim: %q\n", bytes.Trim([]byte("***hi***"), "*"))
	fmt.Printf("TrimLeft: %q\n", bytes.TrimLeft([]byte("000123"), "0"))

	// Replace
	result := bytes.ReplaceAll(b, []byte("Go"), []byte("Rust"))
	fmt.Printf("ReplaceAll: %s\n", result)

	// ToUpper / ToLower
	fmt.Printf("ToUpper: %s\n", bytes.ToUpper([]byte("hello")))
	fmt.Printf("ToLower: %s\n", bytes.ToLower([]byte("HELLO")))

	// EqualFold (case-insensitive equality)
	fmt.Println(bytes.EqualFold([]byte("Go"), []byte("go"))) // true

	// Equal — for []byte equality (can't use == !)
	// WHY: Slices can't be compared with == in Go (only arrays can).
	// bytes.Equal is the correct way to compare two []byte values.
	a1 := []byte("hello")
	a2 := []byte("hello")
	a3 := []byte("world")
	// a1 == a2 // COMPILE ERROR: slice can only be compared to nil
	fmt.Println(bytes.Equal(a1, a2)) // true
	fmt.Println(bytes.Equal(a1, a3)) // false

	// Join
	words := [][]byte{[]byte("one"), []byte("two"), []byte("three")}
	joined := bytes.Join(words, []byte(", "))
	fmt.Printf("Join: %s\n", joined)

	// Compare — returns -1, 0, 1
	fmt.Println(bytes.Compare([]byte("a"), []byte("b"))) // -1

	// Cut (Go 1.20)
	before, after, found := bytes.Cut([]byte("user:pass"), []byte(":"))
	fmt.Printf("Cut: before=%q after=%q found=%v\n", before, after, found)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: bytes.Buffer — the mutable byte buffer
// ─────────────────────────────────────────────────────────────────────────────

func bufferDemo() {
	fmt.Println("═══ SECTION 2: bytes.Buffer ═══")

	// bytes.Buffer is a variable-sized buffer of bytes.
	// It implements: io.Reader, io.Writer, io.ByteScanner, io.RuneScanner, io.WriterTo
	//
	// WHY bytes.Buffer vs strings.Builder?
	//
	// strings.Builder:
	//   + Write-only (you can only add to it, not read from it mid-stream)
	//   + Slightly faster for pure string building
	//   - Cannot be used as io.Reader
	//
	// bytes.Buffer:
	//   + Implements BOTH io.Reader and io.Writer
	//   + You can Write to it and Read from it — it's a queue/pipe
	//   + Bytes/String methods to get current content
	//   + Can pre-load with bytes.NewBufferString(s)
	//
	// RULE:
	//   Use strings.Builder when building a string result to return.
	//   Use bytes.Buffer when you need an intermediate buffer that will be
	//   passed to something that reads from io.Reader (e.g., http.NewRequest body,
	//   json.NewDecoder, or any io.Copy call).

	var buf bytes.Buffer

	// Writing to a Buffer
	buf.WriteString("Hello")
	buf.WriteByte(',')
	buf.WriteRune(' ')
	fmt.Fprintf(&buf, "%s!", "World") // Buffer implements io.Writer

	fmt.Printf("Buffer contents: %q\n", buf.String())
	fmt.Printf("Buffer length: %d\n", buf.Len())

	// Reading from a Buffer (consuming bytes)
	readBuf := make([]byte, 5)
	n, _ := buf.Read(readBuf)
	fmt.Printf("Read %d bytes: %q\n", n, readBuf[:n]) // "Hello"
	fmt.Printf("Remaining: %q\n", buf.String())        // ", World!"

	// ReadString / ReadBytes — read until delimiter
	var buf2 bytes.Buffer
	buf2.WriteString("line1\nline2\nline3")
	line, err := buf2.ReadString('\n')
	fmt.Printf("ReadString: %q err=%v\n", line, err) // "line1\n"
	line, err = buf2.ReadString('\n')
	fmt.Printf("ReadString: %q err=%v\n", line, err) // "line2\n"

	// ReadByte / ReadRune
	buf2.ReadByte() // consume 'l'

	// Bytes() — returns current unread portion WITHOUT copying (careful: modifying
	// the returned slice modifies the buffer!)
	var buf3 bytes.Buffer
	buf3.WriteString("immutable-view")
	view := buf3.Bytes() // no copy
	fmt.Printf("Bytes(): %q\n", view)

	// Reset — clear the buffer but keep capacity (cheap reuse)
	buf3.Reset()
	fmt.Printf("After Reset, len: %d\n", buf3.Len()) // 0

	// bytes.NewBuffer — initialize from existing []byte
	// WHY: Lets you wrap existing data for reading, without copying.
	existingData := []byte("pre-loaded data")
	buf4 := bytes.NewBuffer(existingData)
	fmt.Printf("NewBuffer: %q\n", buf4.String())

	// bytes.NewBufferString — same but from string
	buf5 := bytes.NewBufferString("from string")
	all, _ := buf5.ReadString(0) // read all (delimiter 0 not found)
	_ = all

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: bytes.NewReader — a read-only io.Reader over []byte
// ─────────────────────────────────────────────────────────────────────────────

func readerDemo() {
	fmt.Println("═══ SECTION 3: bytes.NewReader ═══")

	// bytes.NewReader is like strings.NewReader but for []byte.
	// Implements: io.Reader, io.ReaderAt, io.Seeker, io.WriterTo, io.ByteScanner, io.RuneScanner
	//
	// WHY use NewReader instead of Buffer?
	// NewReader is for READ-ONLY access to existing data.
	// Buffer is for READ-WRITE intermediate buffering.
	// NewReader is slightly more efficient for read-only because it doesn't
	// need the overhead of tracking write position separately from read position.

	data := []byte("Hello, Go bytes!")
	reader := bytes.NewReader(data)

	fmt.Printf("Size: %d\n", reader.Len()) // 17

	buf := make([]byte, 5)
	n, _ := reader.Read(buf)
	fmt.Printf("Read: %q (remaining: %d)\n", buf[:n], reader.Len())

	// Seek back to start
	reader.Seek(0, 0)

	// ReadAt — read without advancing position
	buf2 := make([]byte, 2)
	reader.ReadAt(buf2, 7) // read at offset 7
	fmt.Printf("ReadAt(7): %q\n", buf2) // "Go"
	fmt.Printf("Position unchanged: %d remaining\n", reader.Len())

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: string ↔ []byte conversion costs
// ─────────────────────────────────────────────────────────────────────────────

func conversionCostDemo() {
	fmt.Println("═══ SECTION 4: string ↔ []byte Conversion Costs ═══")

	// FUNDAMENTAL RULE: every []byte(s) and string(b) makes a COPY.
	// Go strings are immutable; []byte is mutable. They cannot share memory
	// safely (unless the compiler can prove safety — see optimization below).

	// []byte(s) — allocates and copies
	s := "hello world"
	b := []byte(s) // copy
	b[0] = 'H'
	fmt.Println(s)       // "hello world" — original unchanged
	fmt.Println(string(b)) // "Hello world" — our modified copy

	// string(b) — normally allocates and copies
	b2 := []byte{72, 101, 108, 108, 111}
	s2 := string(b2) // copy
	fmt.Println(s2) // "Hello"

	// COMPILER OPTIMIZATION — zero-copy string(b) in certain contexts:
	// The Go compiler avoids the copy in string(b) when the result is used
	// IMMEDIATELY in a comparison or map lookup, because it can prove no
	// mutation can occur in between.
	//
	// These do NOT allocate (compiler optimizes):
	m := map[string]int{"hello": 1, "world": 2}
	key := []byte("hello")
	_ = m[string(key)]         // optimized: no copy
	_ = string(key) == "hello" // optimized: no copy
	//
	// These DO allocate (compiler cannot guarantee safety):
	stored := string(key)  // stored in a variable — must copy
	fmt.Println(stored)

	fmt.Println()
	fmt.Println("Allocation strategy:")
	fmt.Println("  []byte(s) → always copies (O(n))")
	fmt.Println("  string(b) → always copies EXCEPT map lookup / direct compare")
	fmt.Println("  TIP: Stay in one type for as long as possible; convert only at the boundary")

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Performance — bytes.Buffer vs strings.Builder
// ─────────────────────────────────────────────────────────────────────────────

func performanceComparison() {
	fmt.Println("═══ SECTION 5: Performance Comparison ═══")

	const N = 100_000

	// bytes.Buffer
	start := time.Now()
	var buf bytes.Buffer
	buf.Grow(N)
	for i := 0; i < N; i++ {
		buf.WriteByte('x')
	}
	_ = buf.String()
	bufferTime := time.Since(start)

	// strings.Builder
	start = time.Now()
	var sb strings.Builder
	sb.Grow(N)
	for i := 0; i < N; i++ {
		sb.WriteByte('x')
	}
	_ = sb.String()
	builderTime := time.Since(start)

	fmt.Printf("bytes.Buffer:    %v\n", bufferTime)
	fmt.Printf("strings.Builder: %v\n", builderTime)
	fmt.Println()
	fmt.Println("Both are fast. Difference is usually negligible.")
	fmt.Println("Choose based on what interface you need (Reader vs Writer-only).")
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: bytes.IndexFunc / bytes.Map / bytes.TrimFunc
// ─────────────────────────────────────────────────────────────────────────────

func funcVariantsDemo() {
	fmt.Println("═══ SECTION 6: Func Variants ═══")

	// bytes.IndexFunc — find first byte matching predicate
	data := []byte("Hello123World")
	idx := bytes.IndexFunc(data, unicode.IsDigit)
	fmt.Printf("IndexFunc (digit): %d -> %q\n", idx, data[idx:idx+3])

	// bytes.Map — transform each rune
	upper := bytes.Map(unicode.ToUpper, []byte("hello world"))
	fmt.Printf("Map ToUpper: %s\n", upper)

	// bytes.TrimFunc
	trimmed := bytes.TrimFunc([]byte("123hello456"), unicode.IsDigit)
	fmt.Printf("TrimFunc: %q\n", trimmed)

	// bytes.ContainsFunc (Go 1.21+)
	hasPunct := bytes.ContainsFunc([]byte("Hello!"), unicode.IsPunct)
	fmt.Printf("ContainsFunc (punct): %v\n", hasPunct)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Real-world patterns
// ─────────────────────────────────────────────────────────────────────────────

func realWorldPatterns() {
	fmt.Println("═══ SECTION 7: Real-World Patterns ═══")

	// Pattern 1: Parse an HTTP-style header line
	// "Content-Type: application/json"
	headerLine := []byte("Content-Type: application/json")
	name, value, _ := bytes.Cut(headerLine, []byte(": "))
	fmt.Printf("Header name:  %s\n", name)
	fmt.Printf("Header value: %s\n", value)

	// Pattern 2: Count newlines in a file's content (mock)
	fileContent := []byte("line1\nline2\nline3\nline4\n")
	lineCount := bytes.Count(fileContent, []byte("\n"))
	fmt.Printf("Lines: %d\n", lineCount) // 4

	// Pattern 3: Buffer as a reusable scratch pad
	// Allocate once, reset between uses — common in request handlers
	type Handler struct {
		buf bytes.Buffer
	}
	h := &Handler{}
	for i := 0; i < 3; i++ {
		h.buf.Reset() // reuse without allocation
		fmt.Fprintf(&h.buf, "request-%d processed", i)
		// h.buf.Bytes() could be written to response writer
		fmt.Printf("  %s\n", h.buf.String())
	}

	// Pattern 4: Build binary protocol frame
	// Many network protocols have length-prefixed frames.
	// Using bytes.Buffer to construct the frame:
	var frame bytes.Buffer
	payload := []byte("PING")
	// Write 4-byte big-endian length prefix
	length := uint32(len(payload))
	frame.WriteByte(byte(length >> 24))
	frame.WriteByte(byte(length >> 16))
	frame.WriteByte(byte(length >> 8))
	frame.WriteByte(byte(length))
	frame.Write(payload)
	fmt.Printf("Frame: % x\n", frame.Bytes()) // 00 00 00 04 50 49 4e 47

	// Pattern 5: Use bytes.Buffer as http.Request body
	// import "net/http"
	// body := bytes.NewBufferString(`{"key":"value"}`)
	// req, _ := http.NewRequest("POST", url, body)
	// This is idiomatic and efficient.
	fmt.Println("(See comments for http.NewRequest body pattern)")

	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║          Go Standard Library: bytes Package           ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	mirrorFunctionsDemo()
	bufferDemo()
	readerDemo()
	conversionCostDemo()
	performanceComparison()
	funcVariantsDemo()
	realWorldPatterns()

	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. bytes mirrors strings but for []byte — same API, different type")
	fmt.Println("  2. Use bytes.Equal for []byte equality (== doesn't compile for slices)")
	fmt.Println("  3. bytes.Buffer: read+write, strings.Builder: write-only")
	fmt.Println("  4. Use bytes.Buffer as io.Reader for http bodies, json decoders, etc.")
	fmt.Println("  5. string(b) is optimized (no copy) in map lookups and direct comparisons")
	fmt.Println("  6. Stay in one type domain as long as possible; convert only at I/O boundary")
}
