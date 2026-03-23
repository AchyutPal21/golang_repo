// FILE: 01_fundamentals/11_fmt_printing.go
// TOPIC: fmt Package — All format verbs, Stringer interface, Fprintf, Errorf
//
// Run: go run 01_fundamentals/11_fmt_printing.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   fmt is the most-used package in Go. Knowing all the format verbs and
//   output functions saves you time every day. The Stringer interface shows
//   how Go's implicit interfaces work — a preview of Module 03.
//   Fprintf targeting any io.Writer is the foundation of all output in Go:
//   HTTP responses, files, buffers — they all use the same pattern.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"os"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// fmt.Stringer INTERFACE — How custom types control their string representation
// ─────────────────────────────────────────────────────────────────────────────
//
// If a type implements the String() string method, fmt automatically uses it
// when printing with %v, %s, or plain fmt.Println.
//
// This is Go's implicit interface system: you don't say "implements Stringer",
// you just write the method. If it matches the interface, Go uses it.
//
// This is called "duck typing": if it has a String() method, it's a Stringer.

type Color int

const (
	Red Color = iota
	Green
	Blue
)

// String() makes Color implement fmt.Stringer
// Now fmt.Println(Red) prints "Red" instead of "0"
func (c Color) String() string {
	switch c {
	case Red:
		return "Red"
	case Green:
		return "Green"
	case Blue:
		return "Blue"
	default:
		return fmt.Sprintf("Color(%d)", int(c))
	}
}

type Point struct {
	X, Y float64
}

// Custom string representation for Point
func (p Point) String() string {
	return fmt.Sprintf("(%.2f, %.2f)", p.X, p.Y)
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: fmt Package — Format Verbs")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// OUTPUT FUNCTIONS
	// ─────────────────────────────────────────────────────────────────────
	//
	// fmt.Print(a, b, c)      → no newline, no formatting
	// fmt.Println(a, b, c)    → adds spaces between args, newline at end
	// fmt.Printf(format, ...) → formatted output using verbs
	//
	// fmt.Sprint(...)         → like Print but RETURNS a string
	// fmt.Sprintf(fmt, ...)   → like Printf but RETURNS a string
	// fmt.Sprintln(...)       → like Println but RETURNS a string
	//
	// fmt.Fprint(w, ...)      → like Print but writes to io.Writer w
	// fmt.Fprintf(w, fmt, ...) → like Printf but writes to io.Writer w
	// fmt.Fprintln(w, ...)    → like Println but writes to io.Writer w
	//
	// fmt.Scan(...)           → reads from stdin
	// fmt.Scanf(fmt, ...)     → reads formatted from stdin
	// fmt.Sscanf(str, fmt, ...) → reads from a string
	//
	// fmt.Errorf(fmt, ...)    → creates a formatted error (wraps with %w)

	fmt.Println("\n── Output functions ──")
	fmt.Print("Print: no newline, ")
	fmt.Print("continues on same line\n")
	fmt.Println("Println: auto newline, spaces between:", 1, true, "hello")
	fmt.Printf("Printf: formatted %s %d %v\n", "string", 42, true)

	// Sprint returns a string:
	s := fmt.Sprintf("Formatted string: %d + %d = %d", 10, 20, 10+20)
	fmt.Println(s)

	// Fprintf writes to any io.Writer:
	// os.Stderr is an io.Writer — great for error output
	fmt.Fprintf(os.Stderr, "This goes to stderr: %s\n", "error message")

	// strings.Builder is an io.Writer — efficient string building with fmt:
	var sb strings.Builder
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&sb, "item%d ", i)
	}
	fmt.Printf("Builder result: %q\n", sb.String())

	// ─────────────────────────────────────────────────────────────────────
	// GENERAL FORMAT VERBS
	// ─────────────────────────────────────────────────────────────────────
	//
	// %v    → default format (most common, works for any type)
	// %+v   → default format + struct field names
	// %#v   → Go-syntax representation (can be pasted back into code)
	// %T    → type of the value
	// %%    → literal percent sign

	type User struct {
		Name string
		Age  int
	}
	u := User{"Alice", 30}

	fmt.Println("\n── General verbs ──")
	fmt.Printf("  %%v  → %v\n", u)
	fmt.Printf("  %%+v → %+v\n", u)
	fmt.Printf("  %%#v → %#v\n", u)
	fmt.Printf("  %%T  → %T  (type)\n", u)
	fmt.Printf("  %%%%  → %%  (literal percent)\n")

	// ─────────────────────────────────────────────────────────────────────
	// INTEGER FORMAT VERBS
	// ─────────────────────────────────────────────────────────────────────
	//
	// %d    → decimal (base 10)
	// %b    → binary (base 2)
	// %o    → octal (base 8)
	// %x    → hexadecimal, lowercase (base 16)
	// %X    → hexadecimal, uppercase
	// %c    → character (rune value as Unicode character)
	// %U    → Unicode format (U+0041)
	// %q    → single-quoted character literal ('A')

	n := 255
	fmt.Println("\n── Integer verbs (n=255) ──")
	fmt.Printf("  %%d → %d   (decimal)\n", n)
	fmt.Printf("  %%b → %b  (binary)\n", n)
	fmt.Printf("  %%o → %o   (octal)\n", n)
	fmt.Printf("  %%x → %x   (hex lowercase)\n", n)
	fmt.Printf("  %%X → %X   (hex uppercase)\n", n)
	fmt.Printf("  %%c → %c   (Unicode char for 255 = ÿ)\n", n)
	fmt.Printf("  %%U → %U (Unicode format)\n", 'A')
	fmt.Printf("  %%q → %q  (quoted char)\n", 'A')

	// Width and padding:
	fmt.Println("\n── Width and padding ──")
	fmt.Printf("  %%5d  → '%5d'  (right-align, width 5)\n", 42)
	fmt.Printf("  %%-5d → '%-5d'  (left-align, width 5)\n", 42)
	fmt.Printf("  %%05d → '%05d' (zero-pad, width 5)\n", 42)
	fmt.Printf("  %%+d  → '%+d'  (always show sign)\n", 42)

	// ─────────────────────────────────────────────────────────────────────
	// FLOAT FORMAT VERBS
	// ─────────────────────────────────────────────────────────────────────
	//
	// %f    → decimal point, no exponent (123.456)
	// %e    → scientific notation lowercase (1.23e+02)
	// %E    → scientific notation uppercase (1.23E+02)
	// %g    → shortest representation: %e for large/small, %f otherwise
	// %G    → %E or %F
	//
	// Width.precision: %10.3f → width 10, 3 decimal places

	f := 123456.789
	fmt.Println("\n── Float verbs (f=123456.789) ──")
	fmt.Printf("  %%f     → %f\n", f)
	fmt.Printf("  %%.2f   → %.2f\n", f)
	fmt.Printf("  %%e     → %e\n", f)
	fmt.Printf("  %%E     → %E\n", f)
	fmt.Printf("  %%g     → %g\n", f)
	fmt.Printf("  %%10.3f → '%10.3f'\n", f)
	fmt.Printf("  %%-10.3f→ '%-10.3f'\n", f)

	// ─────────────────────────────────────────────────────────────────────
	// STRING FORMAT VERBS
	// ─────────────────────────────────────────────────────────────────────
	//
	// %s    → plain string
	// %q    → double-quoted string with escape sequences (e.g., "hello\tworld")
	// %x    → hex encoding of bytes, lowercase
	// %X    → hex encoding of bytes, uppercase

	str := "Hello, 世界"
	fmt.Println("\n── String verbs ──")
	fmt.Printf("  %%s  → %s\n", str)
	fmt.Printf("  %%q  → %q\n", str)
	fmt.Printf("  %%x  → %x\n", str)
	fmt.Printf("  %%10s → '%10s'  (right-pad to width 10)\n", "hi")
	fmt.Printf("  %%-10s→ '%-10s'  (left-pad to width 10)\n", "hi")

	// ─────────────────────────────────────────────────────────────────────
	// BOOLEAN AND POINTER VERBS
	// ─────────────────────────────────────────────────────────────────────

	b := true
	p := &n
	fmt.Println("\n── Boolean and pointer verbs ──")
	fmt.Printf("  %%t → %t  (bool)\n", b)
	fmt.Printf("  %%p → %p  (pointer address)\n", p)

	// ─────────────────────────────────────────────────────────────────────
	// Stringer INTERFACE in action
	// ─────────────────────────────────────────────────────────────────────

	fmt.Println("\n── fmt.Stringer interface ──")
	c := Green
	pt := Point{3.0, 4.0}

	fmt.Printf("  Color: %v  (uses String() method)\n", c)
	fmt.Printf("  Point: %v  (uses String() method)\n", pt)
	fmt.Println("  Color with Println:", c)
	fmt.Println("  Point with Println:", pt)

	// ─────────────────────────────────────────────────────────────────────
	// fmt.Errorf — Creating formatted errors (with wrapping via %w)
	// ─────────────────────────────────────────────────────────────────────
	//
	// fmt.Errorf is how you create error values with context.
	// %w wraps another error so errors.Is / errors.As can unwrap it.
	// (Covered in depth in Module 04)

	baseErr := fmt.Errorf("database connection failed")
	wrappedErr := fmt.Errorf("user lookup: %w", baseErr)
	fmt.Println("\n── fmt.Errorf ──")
	fmt.Printf("  baseErr:    %v\n", baseErr)
	fmt.Printf("  wrappedErr: %v\n", wrappedErr)

	// ─────────────────────────────────────────────────────────────────────
	// fmt.Sscanf — Parsing formatted strings
	// ─────────────────────────────────────────────────────────────────────

	input := "Alice 30"
	var name string
	var age int
	fmt.Sscanf(input, "%s %d", &name, &age)
	fmt.Printf("\n── fmt.Sscanf ──\n")
	fmt.Printf("  Parsed %q → name=%q age=%d\n", input, name, age)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  %v / %+v / %#v / %T  → general, struct, Go-syntax, type")
	fmt.Println("  %d %b %o %x          → integer: decimal, binary, octal, hex")
	fmt.Println("  %f %e %g             → float: fixed, scientific, shortest")
	fmt.Println("  %s %q                → string: plain, quoted-with-escapes")
	fmt.Println("  %t %p                → bool, pointer address")
	fmt.Println("  width.precision: %10.3f, %-10s, %05d")
	fmt.Println("  Stringer: implement String() string for custom formatting")
	fmt.Println("  Fprintf(w, ...) writes to any io.Writer")
	fmt.Println("  Errorf with %w wraps errors (covered in Module 04)")
}
