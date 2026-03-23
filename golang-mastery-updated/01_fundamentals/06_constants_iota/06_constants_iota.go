// FILE: 01_fundamentals/06_constants_iota.go
// TOPIC: Constants & iota — typed vs untyped, const blocks, iota patterns
//
// Run: go run 01_fundamentals/06_constants_iota.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Constants in Go are more powerful than in most languages because of
//   "untyped constants" — values with arbitrary precision that adapt to their
//   context. iota enables elegant enum-like sequences. Understanding these
//   prevents subtle type mismatch errors when working with numeric constants.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// BASIC CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────
//
// const declares a value that is fixed at COMPILE TIME.
// Constants CANNOT be changed after declaration.
// Constants CANNOT be assigned the result of a function call
// (because function calls are runtime operations).
//
// Valid const values: literals, other constants, arithmetic on constants,
//   builtin functions: len(), cap(), unsafe.Sizeof(), real(), imag(), complex()
//
// Invalid: os.Getenv(), time.Now(), rand.Int() — these are runtime values.

const Pi = 3.14159265358979323846  // untyped constant (see below)
const AppVersion = "2.0.0"         // untyped string constant
const MaxConnections = 100         // untyped integer constant

// Typed constant: explicitly given a type
const TypedPi float32 = 3.14159

// Grouped const block:
const (
	MaxRetries = 3
	Timeout    = 30  // seconds
	BaseURL    = "https://api.example.com"
)

// ─────────────────────────────────────────────────────────────────────────────
// UNTYPED CONSTANTS — Go's most subtle constant feature
// ─────────────────────────────────────────────────────────────────────────────
//
// When you write: const Pi = 3.14159...
// Pi is an UNTYPED constant. It has a "kind" (floating point) but no specific type.
//
// BENEFIT: An untyped constant can be used wherever its kind is compatible,
// without explicit conversion. It takes the type of its context.
//
// Example:
//   const X = 5     // untyped integer constant
//   var a int32 = X  // works: X adapts to int32
//   var b int64 = X  // works: X adapts to int64
//   var c float64 = X // works: X adapts to float64
//
// If X were a TYPED constant (const X int = 5), then:
//   var b int64 = X  // COMPILE ERROR: cannot use int as int64
//   var b int64 = int64(X)  // must explicitly convert
//
// PRECISION: Untyped constants can have ARBITRARY precision.
//   They are computed at compile time with 256+ bits of precision.
//   Only when assigned to a variable do they get truncated to that type's precision.

func demonstrateUntypedConstants() {
	fmt.Println("\n── Untyped constants adapt to context ──")

	const X = 5  // untyped integer

	var a int8 = X     // X becomes int8(5)
	var b int64 = X    // X becomes int64(5)
	var c float64 = X  // X becomes float64(5.0)
	var d complex128 = X  // X becomes complex128(5+0i)

	fmt.Printf("  const X=5 used as int8=%v, int64=%v, float64=%v, complex128=%v\n",
		a, b, c, d)

	// Untyped float constant with high precision:
	const HighPrecPi = 3.14159265358979323846264338327950288

	var f32 float32 = HighPrecPi  // truncated to float32 precision
	var f64 float64 = HighPrecPi  // truncated to float64 precision

	fmt.Printf("  HighPrecPi as float32: %.10f\n", f32)
	fmt.Printf("  HighPrecPi as float64: %.15f\n", f64)

	// Typed constant is LESS flexible:
	const TypedX int32 = 5
	var e int32 = TypedX          // ok, same type
	// var f int64 = TypedX       // compile error: cannot use int32 as int64
	var f int64 = int64(TypedX)   // must convert explicitly
	fmt.Printf("  Typed int32 const → int64 requires explicit cast: %v\n", f)
	_ = e
}

// ─────────────────────────────────────────────────────────────────────────────
// iota — The Enum Generator
// ─────────────────────────────────────────────────────────────────────────────
//
// iota is a special predeclared identifier available only in const blocks.
// It starts at 0 for the first constant in the block and increments by 1
// for each subsequent constant.
//
// iota RESETS to 0 at the start of each new const() block.
//
// WHY USE iota?
//   - Creates enum-like sequences without manually typing 0, 1, 2, 3...
//   - When you insert a new value, you don't have to renumber everything
//   - Can use expressions to create non-trivial sequences (powers of 2, etc.)

// Basic iota: starts at 0
type Weekday int

const (
	Sunday    Weekday = iota  // 0
	Monday                    // 1  (iota increments automatically)
	Tuesday                   // 2
	Wednesday                 // 3
	Thursday                  // 4
	Friday                    // 5
	Saturday                  // 6
)

func (d Weekday) String() string {
	names := [...]string{"Sunday", "Monday", "Tuesday", "Wednesday",
		"Thursday", "Friday", "Saturday"}
	if d < Sunday || d > Saturday {
		return "Unknown"
	}
	return names[d]
}

// iota starting at 1 (skip 0)
type Month int

const (
	_           = iota  // discard 0 with blank identifier
	January    Month = iota  // 1
	February                 // 2
	March                    // 3
	// ... etc
)

// iota with expressions — bit flags (powers of 2)
// This is the most powerful iota pattern.
// Use for permission bits, feature flags, option flags.
type Permission uint

const (
	Read    Permission = 1 << iota  // 1 << 0 = 1   (binary: 0001)
	Write                           // 1 << 1 = 2   (binary: 0010)
	Execute                         // 1 << 2 = 4   (binary: 0100)
	Delete                          // 1 << 3 = 8   (binary: 1000)
)

func (p Permission) String() string {
	var s string
	if p&Read != 0 {
		s += "r"
	} else {
		s += "-"
	}
	if p&Write != 0 {
		s += "w"
	} else {
		s += "-"
	}
	if p&Execute != 0 {
		s += "x"
	} else {
		s += "-"
	}
	if p&Delete != 0 {
		s += "d"
	} else {
		s += "-"
	}
	return s
}

// iota with byte-size multipliers (KB, MB, GB, ...)
// 1 << (10 * n) gives 1, 1024, 1048576, ...
type ByteSize float64

const (
	_           = iota  // ignore 0
	KB ByteSize = 1 << (10 * iota)  // 1 << 10 = 1024
	MB                              // 1 << 20 = 1,048,576
	GB                              // 1 << 30 = 1,073,741,824
	TB                              // 1 << 40
	PB                              // 1 << 50
)

// Multiple constants sharing an iota value (same line = same iota)
type Direction int

const (
	North, NorthEast Direction = iota, iota + 4  // iota=0: North=0, NorthEast=4
	East, SouthEast                               // iota=1: East=1, SouthEast=5
	South, SouthWest                              // iota=2: South=2, SouthWest=6
	West, NorthWest                               // iota=3: West=3, NorthWest=7
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Constants & iota")
	fmt.Println("════════════════════════════════════════")

	// ── Basic constants ──────────────────────────────────────────────────
	fmt.Printf("\n── Basic constants ──\n")
	fmt.Printf("  Pi           = %v\n", Pi)
	fmt.Printf("  AppVersion   = %q\n", AppVersion)
	fmt.Printf("  MaxRetries   = %d\n", MaxRetries)
	fmt.Printf("  TypedPi      = %v (type: %T)\n", TypedPi, TypedPi)

	// ── Untyped constants ────────────────────────────────────────────────
	demonstrateUntypedConstants()

	// ── Weekday enum ─────────────────────────────────────────────────────
	fmt.Printf("\n── iota: Weekday enum ──\n")
	fmt.Printf("  Sunday=%d Monday=%d Tuesday=%d Wednesday=%d\n",
		Sunday, Monday, Tuesday, Wednesday)
	fmt.Printf("  Thursday=%d Friday=%d Saturday=%d\n",
		Thursday, Friday, Saturday)
	today := Wednesday
	fmt.Printf("  today=%v (value=%d)\n", today, today)

	// ── Month (skip 0) ───────────────────────────────────────────────────
	fmt.Printf("\n── iota: Month enum (skip 0) ──\n")
	fmt.Printf("  January=%d February=%d March=%d\n", January, February, March)

	// ── Bit flags ────────────────────────────────────────────────────────
	fmt.Printf("\n── iota: Bit flags (permissions) ──\n")
	fmt.Printf("  Read=%d Write=%d Execute=%d Delete=%d\n",
		Read, Write, Execute, Delete)

	// Combining permissions with bitwise OR:
	userPerm := Read | Write           // 1 | 2 = 3 (binary: 0011)
	adminPerm := Read | Write | Execute | Delete  // 1|2|4|8 = 15

	fmt.Printf("  user  permissions = %d = %s\n", userPerm, userPerm)
	fmt.Printf("  admin permissions = %d = %s\n", adminPerm, adminPerm)

	// Checking a permission with bitwise AND:
	fmt.Printf("  user can write? %v\n", userPerm&Write != 0)
	fmt.Printf("  user can delete? %v\n", userPerm&Delete != 0)

	// ── Byte sizes ───────────────────────────────────────────────────────
	fmt.Printf("\n── iota: Byte sizes ──\n")
	fmt.Printf("  KB = %.0f\n", float64(KB))
	fmt.Printf("  MB = %.0f\n", float64(MB))
	fmt.Printf("  GB = %.0f\n", float64(GB))
	fmt.Printf("  TB = %.0f\n", float64(TB))

	// ── Direction ────────────────────────────────────────────────────────
	fmt.Printf("\n── iota: Multiple consts per iota ──\n")
	fmt.Printf("  North=%d NorthEast=%d\n", North, NorthEast)
	fmt.Printf("  East=%d  SouthEast=%d\n", East, SouthEast)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  const = compile-time fixed value")
	fmt.Println("  Untyped const: adapts to context, arbitrary precision")
	fmt.Println("  Typed const: fixed type, requires explicit conversion")
	fmt.Println("  iota: auto-incrementing enum generator, resets per const block")
	fmt.Println("  1 << iota: bit flag pattern (most common iota use)")
	fmt.Println("  _ = iota: skip a value")
}
