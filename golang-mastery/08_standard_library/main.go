package main

// =============================================================================
// MODULE 08: STANDARD LIBRARY — Essential packages every Go dev must know
// =============================================================================
// Run: go run 08_standard_library/main.go
// =============================================================================

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// =============================================================================
// TYPES FOR JSON DEMO
// =============================================================================

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip,omitempty"`
}

type Person struct {
	Name      string  `json:"name"`
	Age       int     `json:"age"`
	Email     string  `json:"email,omitempty"`
	Address   Address `json:"address"`
	Tags      []string `json:"tags,omitempty"`
	IsActive  bool    `json:"is_active"`
	Score     float64 `json:"score"`
	internal  string  // unexported — not marshaled
}

// Custom JSON marshaling
type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) // "1h30m0s"
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

func main() {
	fmt.Println("=== MODULE 08: STANDARD LIBRARY ===")

	// =========================================================================
	// fmt — formatted I/O
	// =========================================================================
	fmt.Println("\n--- fmt ---")

	// Printing
	fmt.Print("no newline")
	fmt.Println(" ← Print")
	fmt.Printf("formatted: %d %.2f %s %t\n", 42, 3.14, "go", true)

	// Formatting to string
	s := fmt.Sprintf("[%05d]", 42) // [00042]
	fmt.Println("Sprintf:", s)

	// Printing to writer
	fmt.Fprintln(os.Stdout, "to stdout")
	fmt.Fprintln(os.Stderr, "to stderr") // visible in terminal

	// Scanning (reading input)
	// var name string
	// fmt.Scan(&name)       // reads word
	// fmt.Scanln(&name)     // reads line
	// fmt.Scanf("%s", &name)

	// Errorf
	err := fmt.Errorf("something went wrong: %d", 42)
	fmt.Println("error:", err)

	// =========================================================================
	// strings — string manipulation
	// =========================================================================
	fmt.Println("\n--- strings ---")

	s1 := "Hello, World!"

	fmt.Println("ToUpper:", strings.ToUpper(s1))
	fmt.Println("ToLower:", strings.ToLower(s1))
	fmt.Println("Title:", strings.Title("hello world")) // deprecated but common
	fmt.Println("Contains:", strings.Contains(s1, "World"))
	fmt.Println("HasPrefix:", strings.HasPrefix(s1, "Hello"))
	fmt.Println("HasSuffix:", strings.HasSuffix(s1, "!"))
	fmt.Println("Count:", strings.Count(s1, "l"))     // 3
	fmt.Println("Index:", strings.Index(s1, "World")) // 7
	fmt.Println("Replace:", strings.Replace(s1, "l", "L", 2)) // replaces first 2
	fmt.Println("ReplaceAll:", strings.ReplaceAll(s1, "l", "L"))
	fmt.Println("TrimSpace:", strings.TrimSpace("  hello  "))
	fmt.Println("Trim:", strings.Trim("***hello***", "*"))
	fmt.Println("TrimLeft:", strings.TrimLeft("...hello...", "."))
	fmt.Println("TrimRight:", strings.TrimRight("...hello...", "."))
	fmt.Println("TrimPrefix:", strings.TrimPrefix("Hello, World", "Hello, "))
	fmt.Println("TrimSuffix:", strings.TrimSuffix("hello.go", ".go"))

	// Split and Join
	parts := strings.Split("a,b,c,d", ",")
	fmt.Println("Split:", parts)
	fmt.Println("Join:", strings.Join(parts, " | "))

	// SplitN — split into at most N substrings
	fmt.Println("SplitN:", strings.SplitN("a:b:c:d", ":", 3))

	// Fields — split by whitespace
	fmt.Println("Fields:", strings.Fields("  hello   world   go  "))

	// Repeat
	fmt.Println("Repeat:", strings.Repeat("go", 3))

	// EqualFold — case-insensitive comparison
	fmt.Println("EqualFold:", strings.EqualFold("Go", "go"))

	// ContainsAny
	fmt.Println("ContainsAny:", strings.ContainsAny("hello", "aeiou"))

	// Map — transform each rune
	rot13 := func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		}
		return r
	}
	fmt.Println("ROT13:", strings.Map(rot13, "Hello, World!"))

	// strings.Builder — efficient string building (avoid + concatenation in loops)
	var sb strings.Builder
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "item%d", i)
		if i < 4 {
			sb.WriteString(", ")
		}
	}
	fmt.Println("Builder:", sb.String())

	// strings.Reader — treat string as io.Reader
	reader := strings.NewReader("hello world")
	buf := make([]byte, 5)
	n, _ := reader.Read(buf)
	fmt.Printf("Read %d bytes: %s\n", n, buf[:n])

	// =========================================================================
	// strconv — string/number conversions
	// =========================================================================
	fmt.Println("\n--- strconv ---")

	// Int to string and back
	i := 42
	str1 := strconv.Itoa(i) // int → string "42"
	fmt.Println("Itoa:", str1)

	num, err2 := strconv.Atoi("123") // string → int
	fmt.Println("Atoi:", num, err2)

	_, err3 := strconv.Atoi("abc") // error case
	fmt.Println("Atoi error:", err3)

	// ParseInt — more control
	n2, err4 := strconv.ParseInt("FF", 16, 64) // base 16, 64-bit
	fmt.Println("ParseInt hex:", n2, err4)      // 255

	// ParseFloat
	f, _ := strconv.ParseFloat("3.14159", 64)
	fmt.Println("ParseFloat:", f)

	// ParseBool
	b, _ := strconv.ParseBool("true")
	fmt.Println("ParseBool:", b)

	b2, _ := strconv.ParseBool("1") // "1", "t", "T", "true", "TRUE", "True"
	fmt.Println("ParseBool '1':", b2)

	// FormatInt
	fmt.Println("FormatInt binary:", strconv.FormatInt(42, 2))  // "101010"
	fmt.Println("FormatInt hex:", strconv.FormatInt(255, 16))   // "ff"
	fmt.Println("FormatFloat:", strconv.FormatFloat(3.14, 'f', 2, 64)) // "3.14"

	// Quote/Unquote
	fmt.Println("Quote:", strconv.Quote("Hello\nWorld"))
	unq, _ := strconv.Unquote(`"Hello\nWorld"`)
	fmt.Println("Unquote:", unq)

	// =========================================================================
	// encoding/json — JSON marshaling/unmarshaling
	// =========================================================================
	fmt.Println("\n--- encoding/json ---")

	// Marshaling (Go → JSON)
	person := Person{
		Name:     "Achyut",
		Age:      25,
		Email:    "achyut@example.com",
		IsActive: true,
		Score:    98.5,
		Address: Address{
			Street: "123 Main St",
			City:   "Kolkata",
			Zip:    "700001",
		},
		Tags:     []string{"developer", "go"},
		internal: "secret", // will NOT be in JSON
	}

	jsonBytes, err5 := json.Marshal(person)
	if err5 != nil {
		fmt.Println("marshal error:", err5)
	} else {
		fmt.Println("JSON:", string(jsonBytes))
	}

	// Pretty print
	prettyBytes, _ := json.MarshalIndent(person, "", "  ")
	fmt.Println("Pretty JSON:\n", string(prettyBytes))

	// Unmarshaling (JSON → Go)
	jsonStr := `{"name":"Bob","age":30,"is_active":true,"address":{"street":"456 Oak Ave","city":"Mumbai"},"score":85.5}`
	var p2 Person
	if err := json.Unmarshal([]byte(jsonStr), &p2); err != nil {
		fmt.Println("unmarshal error:", err)
	} else {
		fmt.Printf("Unmarshaled: %+v\n", p2)
	}

	// Unknown structure — unmarshal to map
	var raw map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &raw)
	fmt.Println("raw map:", raw)

	// json.Decoder — for streams (HTTP bodies, files)
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	var p3 Person
	decoder.Decode(&p3)
	fmt.Println("decoded:", p3.Name, p3.Age)

	// Unmarshal to slice
	jsonArr := `[{"name":"Alice","age":25,"is_active":true,"address":{},"score":0},{"name":"Bob","age":30,"is_active":false,"address":{},"score":0}]`
	var people []Person
	json.Unmarshal([]byte(jsonArr), &people)
	fmt.Println("array:", people[0].Name, people[1].Name)

	// =========================================================================
	// time — date and time
	// =========================================================================
	fmt.Println("\n--- time ---")

	// Current time
	now := time.Now()
	fmt.Println("Now:", now)
	fmt.Println("UTC:", now.UTC())
	fmt.Println("Unix:", now.Unix())       // seconds
	fmt.Println("UnixMilli:", now.UnixMilli()) // milliseconds
	fmt.Println("UnixNano:", now.UnixNano()) // nanoseconds

	// Components
	fmt.Printf("Year=%d Month=%s Day=%d Hour=%d Min=%d Sec=%d\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	// Weekday
	fmt.Println("Weekday:", now.Weekday())

	// Formatting — Go uses a REFERENCE TIME: Mon Jan 2 15:04:05 MST 2006
	// This specific time is used as the format template — unique to Go!
	fmt.Println("Formatted:", now.Format("2006-01-02 15:04:05"))
	fmt.Println("Date only:", now.Format("2006-01-02"))
	fmt.Println("Time only:", now.Format("15:04:05"))
	fmt.Println("Custom:", now.Format("Mon, 02 Jan 2006 15:04:05 MST"))

	// Parsing
	t, _ := time.Parse("2006-01-02", "2024-01-15")
	fmt.Println("Parsed:", t)

	// Duration
	d := 2*time.Hour + 30*time.Minute + 15*time.Second
	fmt.Println("Duration:", d)
	fmt.Printf("Hours: %.1f\n", d.Hours())
	fmt.Printf("Minutes: %.0f\n", d.Minutes())
	fmt.Printf("Seconds: %.0f\n", d.Seconds())

	// Add and Sub
	tomorrow := now.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	fmt.Println("Tomorrow:", tomorrow.Format("2006-01-02"))
	fmt.Println("Yesterday:", yesterday.Format("2006-01-02"))

	// Difference between times
	diff := tomorrow.Sub(now)
	fmt.Println("Diff:", diff)

	// After, Before, Equal
	fmt.Println("Tomorrow after now:", tomorrow.After(now))
	fmt.Println("Yesterday before now:", yesterday.Before(now))

	// time.Since / time.Until
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	elapsed := time.Since(start)
	fmt.Printf("Elapsed: %v\n", elapsed)

	// Timer and Ticker
	// time.NewTimer(d) — fires once after d
	// time.NewTicker(d) — fires every d
	// time.After(d) — channel version of timer
	// time.Sleep(d) — blocks goroutine

	// Timezones
	loc, _ := time.LoadLocation("Asia/Kolkata")
	kolkataTime := now.In(loc)
	fmt.Println("Kolkata time:", kolkataTime.Format("15:04:05 MST"))

	// =========================================================================
	// os — operating system interface
	// =========================================================================
	fmt.Println("\n--- os ---")

	// Process info
	fmt.Println("PID:", os.Getpid())
	fmt.Println("Args:", os.Args) // [program_name, arg1, arg2, ...]

	// Environment
	fmt.Println("HOME:", os.Getenv("HOME"))
	os.Setenv("MY_VAR", "hello") // set env var
	fmt.Println("MY_VAR:", os.Getenv("MY_VAR"))

	// File operations — write and read
	tmpFile := "/tmp/go_mastery_test.txt"

	// WriteFile — simple write
	content := []byte("Hello, Go File System!\nLine 2\nLine 3\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		fmt.Println("write error:", err)
	} else {
		fmt.Println("file written:", tmpFile)
	}

	// ReadFile — simple read
	data, err6 := os.ReadFile(tmpFile)
	if err6 != nil {
		fmt.Println("read error:", err6)
	} else {
		fmt.Printf("file content:\n%s", data)
	}

	// os.Open and manual reading
	file, err7 := os.Open(tmpFile)
	if err7 != nil {
		fmt.Println("open error:", err7)
	} else {
		defer file.Close() // ALWAYS close files!
		// Read with bufio.Scanner — line by line
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			fmt.Printf("line %d: %s\n", lineNum, scanner.Text())
		}
	}

	// File info
	info, _ := os.Stat(tmpFile)
	fmt.Println("file size:", info.Size(), "bytes")
	fmt.Println("file mode:", info.Mode())
	fmt.Println("modified:", info.ModTime().Format("2006-01-02 15:04:05"))

	// Directory operations
	os.Mkdir("/tmp/go_mastery_dir", 0755)
	os.MkdirAll("/tmp/go_mastery_dir/sub1/sub2", 0755) // create all parents
	os.Remove(tmpFile)                                  // remove file
	os.RemoveAll("/tmp/go_mastery_dir")                 // remove directory tree

	// Working directory
	wd, _ := os.Getwd()
	fmt.Println("Working dir:", wd)

	// Exit — exits the program immediately, defers do NOT run
	// os.Exit(0) // 0 = success, non-zero = failure

	// =========================================================================
	// io — I/O primitives
	// =========================================================================
	fmt.Println("\n--- io ---")

	// io.Reader and io.Writer are THE most important interfaces in Go
	// Everything that reads or writes implements them.

	// io.Copy — copy from reader to writer
	src2 := strings.NewReader("Hello, World!")
	var dst strings.Builder
	bytesWritten, _ := io.Copy(&dst, src2)
	fmt.Printf("Copied %d bytes: %s\n", bytesWritten, dst.String())

	// io.ReadAll — read everything from a reader
	r2 := strings.NewReader("read all of this")
	allBytes, _ := io.ReadAll(r2)
	fmt.Println("ReadAll:", string(allBytes))

	// io.MultiReader — concatenate readers
	r3 := io.MultiReader(
		strings.NewReader("first "),
		strings.NewReader("second "),
		strings.NewReader("third"),
	)
	allBytes2, _ := io.ReadAll(r3)
	fmt.Println("MultiReader:", string(allBytes2))

	// io.TeeReader — read and simultaneously write to another writer
	var logBuf strings.Builder
	teeReader := io.TeeReader(strings.NewReader("tee this"), &logBuf)
	io.ReadAll(teeReader) // reads from source
	fmt.Println("TeeReader log:", logBuf.String()) // also wrote to logBuf

	// io.LimitReader — limit how many bytes can be read
	limited := io.LimitReader(strings.NewReader("hello world"), 5)
	limitedBytes, _ := io.ReadAll(limited)
	fmt.Println("LimitReader:", string(limitedBytes)) // "hello"

	// bufio — buffered I/O
	bReader := bufio.NewReader(strings.NewReader("hello\nworld\ngo\n"))
	for {
		line, err := bReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("error:", err)
			}
			if line != "" {
				fmt.Println("bufio line:", strings.TrimRight(line, "\n"))
			}
			break
		}
		fmt.Println("bufio line:", strings.TrimRight(line, "\n"))
	}

	// =========================================================================
	// sort — sorting
	// =========================================================================
	fmt.Println("\n--- sort ---")

	ints2 := []int{5, 2, 8, 1, 9, 3, 7, 4, 6}
	sort.Ints(ints2)
	fmt.Println("sorted ints:", ints2)

	strs2 := []string{"banana", "apple", "cherry"}
	sort.Strings(strs2)
	fmt.Println("sorted strings:", strs2)

	floats := []float64{3.14, 1.41, 2.71, 1.73}
	sort.Float64s(floats)
	fmt.Println("sorted floats:", floats)

	// Reverse sort
	sort.Sort(sort.Reverse(sort.IntSlice(ints2)))
	fmt.Println("reverse sorted:", ints2)

	// Search — binary search on sorted slice
	sort.Ints(ints2) // sort again
	idx := sort.SearchInts(ints2, 5)
	fmt.Println("index of 5:", idx, "value:", ints2[idx])

	// =========================================================================
	// regexp — regular expressions
	// =========================================================================
	fmt.Println("\n--- regexp ---")

	// Compile — parse regex (panics if invalid, use MustCompile for literals)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	fmt.Println("valid email:", emailRegex.MatchString("user@example.com"))
	fmt.Println("invalid email:", emailRegex.MatchString("not-an-email"))

	// FindString
	numRegex := regexp.MustCompile(`\d+`)
	fmt.Println("first number:", numRegex.FindString("abc 123 def 456"))

	// FindAllString
	fmt.Println("all numbers:", numRegex.FindAllString("abc 123 def 456", -1)) // -1 = all

	// FindStringSubmatch — capture groups
	dateRegex := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	matches := dateRegex.FindStringSubmatch("Date: 2024-01-15")
	if matches != nil {
		fmt.Println("full match:", matches[0])
		fmt.Println("year:", matches[1], "month:", matches[2], "day:", matches[3])
	}

	// ReplaceAllString
	result2 := regexp.MustCompile(`\s+`).ReplaceAllString("hello   world   go", " ")
	fmt.Println("cleaned:", result2)

	// Split
	parts2 := regexp.MustCompile(`[,;\s]+`).Split("a, b; c  d", -1)
	fmt.Println("split:", parts2)

	// =========================================================================
	// net/url — URL parsing
	// =========================================================================
	fmt.Println("\n--- net/url ---")

	rawURL := "https://user:pass@example.com:8080/api/v1/users?page=1&limit=10#anchor"
	u, _ := url.Parse(rawURL)

	fmt.Println("Scheme:", u.Scheme)
	fmt.Println("Host:", u.Host)
	fmt.Println("Hostname:", u.Hostname())
	fmt.Println("Port:", u.Port())
	fmt.Println("Path:", u.Path)
	fmt.Println("RawQuery:", u.RawQuery)
	fmt.Println("Fragment:", u.Fragment)
	fmt.Println("User:", u.User.Username())

	// Query params
	params := u.Query() // returns url.Values (map[string][]string)
	fmt.Println("page:", params.Get("page"))
	fmt.Println("limit:", params.Get("limit"))

	// Build URL
	params.Set("page", "2")
	params.Add("filter", "active")
	u.RawQuery = params.Encode()
	fmt.Println("Updated URL:", u.String())

	// URL encode/decode
	encoded := url.QueryEscape("hello world & more")
	fmt.Println("Encoded:", encoded)
	decoded, _ := url.QueryUnescape(encoded)
	fmt.Println("Decoded:", decoded)

	// =========================================================================
	// math/rand — random numbers
	// =========================================================================
	fmt.Println("\n--- math/rand ---")

	// Create a new source with seed (reproducible)
	rng := rand.New(rand.NewSource(42))

	fmt.Println("Intn(100):", rng.Intn(100))       // random int [0, 100)
	fmt.Println("Float64:", rng.Float64())           // random float [0.0, 1.0)
	fmt.Println("Int63:", rng.Int63())

	// Shuffle a slice
	s2 := []int{1, 2, 3, 4, 5}
	rng.Shuffle(len(s2), func(i, j int) {
		s2[i], s2[j] = s2[j], s2[i]
	})
	fmt.Println("Shuffled:", s2)

	// =========================================================================
	// filepath — OS-specific path operations
	// =========================================================================
	fmt.Println("\n--- filepath ---")

	path := "/home/user/documents/report.txt"
	fmt.Println("Dir:", filepath.Dir(path))
	fmt.Println("Base:", filepath.Base(path))
	fmt.Println("Ext:", filepath.Ext(path))

	// Join — OS-appropriate separator
	joined := filepath.Join("/home", "user", "docs", "file.txt")
	fmt.Println("Join:", joined)

	// Clean — normalize path
	dirty := "/home/user/../user/./docs//file.txt"
	fmt.Println("Clean:", filepath.Clean(dirty))

	// Abs — absolute path
	abs, _ := filepath.Abs("relative/path")
	fmt.Println("Abs:", abs)

	// Match — glob pattern matching
	matched, _ := filepath.Match("*.go", "main.go")
	fmt.Println("Match *.go main.go:", matched)

	// =========================================================================
	// unicode — Unicode inspection
	// =========================================================================
	fmt.Println("\n--- unicode ---")

	chars := []rune{'A', 'a', '5', ' ', '!', '€', 'α'}
	for _, ch := range chars {
		fmt.Printf("%c → IsLetter=%v IsDigit=%v IsSpace=%v IsUpper=%v IsLower=%v\n",
			ch, unicode.IsLetter(ch), unicode.IsDigit(ch),
			unicode.IsSpace(ch), unicode.IsUpper(ch), unicode.IsLower(ch))
	}

	fmt.Println("\n=== MODULE 08 COMPLETE ===")
}
